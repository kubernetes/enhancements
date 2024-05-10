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

# [KEP-4381](https://github.com/kubernetes/enhancements/issues/4381): Dynamic Resource Allocation with Structured Parameters

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Cluster add-on development](#cluster-add-on-development)
    - [Cluster configuration](#cluster-configuration)
    - [Partial GPU allocation](#partial-gpu-allocation)
  - [Publishing node resources](#publishing-node-resources)
  - [Using structured parameters](#using-structured-parameters)
  - [Communicating allocation to the DRA driver](#communicating-allocation-to-the-dra-driver)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Feature not used](#feature-not-used)
    - [Compromised node](#compromised-node)
    - [Compromised resource driver plugin](#compromised-resource-driver-plugin)
    - [User permissions and quotas](#user-permissions-and-quotas)
    - [Usability](#usability)
- [Design Details](#design-details)
  - [Components](#components)
  - [State and communication](#state-and-communication)
  - [Custom parameters](#custom-parameters)
  - [Sharing a single ResourceClaim](#sharing-a-single-resourceclaim)
  - [Ephemeral vs. persistent ResourceClaims lifecycle](#ephemeral-vs-persistent-resourceclaims-lifecycle)
  - [Scheduled pods with unallocated or unreserved claims](#scheduled-pods-with-unallocated-or-unreserved-claims)
  - [Handling non graceful node shutdowns](#handling-non-graceful-node-shutdowns)
  - [API](#api)
    - [resource.k8s.io](#resourcek8sio)
      - [ResourceSlice](#resourceslice)
      - [ResourceClass](#resourceclass)
      - [ResourceClassParameters](#resourceclassparameters)
      - [ResourceClaimParameters](#resourceclaimparameters)
      - [SetupParameters](#setupparameters)
      - [Allocation result](#allocation-result)
      - [ResourceClaimTemplate](#resourceclaimtemplate)
      - [Object references](#object-references)
    - [core](#core)
  - [kube-controller-manager](#kube-controller-manager)
  - [kube-scheduler](#kube-scheduler)
    - [EventsToRegister](#eventstoregister)
    - [PreEnqueue](#preenqueue)
    - [Pre-filter](#pre-filter)
    - [Filter](#filter)
    - [Post-filter](#post-filter)
    - [Reserve](#reserve)
    - [PreBind](#prebind)
    - [Unreserve](#unreserve)
  - [kubelet](#kubelet)
    - [Managing resources](#managing-resources)
    - [Communication between kubelet and resource kubelet plugin](#communication-between-kubelet-and-resource-kubelet-plugin)
      - [NodeListAndWatchResources](#nodelistandwatchresources)
      - [NodePrepareResource](#nodeprepareresource)
      - [NodeUnprepareResources](#nodeunprepareresources)
  - [Simulation with CA](#simulation-with-ca)
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
  - [Publishing resource information in node status](#publishing-resource-information-in-node-status)
  - [Injecting vendor logic into CA](#injecting-vendor-logic-into-ca)
  - [ResourceClaimTemplate](#resourceclaimtemplate-1)
  - [Reusing volume support as-is](#reusing-volume-support-as-is)
  - [Extend volume support](#extend-volume-support)
  - [Extend Device Plugins](#extend-device-plugins)
  - [Webhooks instead of ResourceClaim updates](#webhooks-instead-of-resourceclaim-updates)
  - [ResourceDriver](#resourcedriver)
  - [Complex sharing of ResourceClaim](#complex-sharing-of-resourceclaim)
- [Infrastructure Needed](#infrastructure-needed)
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

This KEP originally defined an extension of the ["classic" DRA #3063
KEP](../3063-dynamic-resource-allocation/README.md). Now the roles are
reversed: this KEP defines the base functionality and #3063 adds an optional
extension.

Users are increasingly deploying Kubernetes as management solution for new
workloads (batch processing) and in new environments (edge computing). Such
workloads no longer need just RAM and CPU, but also access to specialized
hardware. With upcoming enhancements of data center interconnects, accelerators
can be installed outside of specific nodes and be connected to nodes
dynamically as needed.

This KEP introduces a new API for describing which of these new resources
a pod needs. The API supports:

- Network-attached resources. The existing [device plugin API](https://github.com/kubernetes/design-proposals-archive/blob/main/resource-management/device-plugin.md)
  is limited to hardware on a node. However, further work is still
  needed to actually use the new API with those.
- Sharing of a resource allocation between multiple containers or pods.
  The device manager API currently cannot share resources at all. It
  could be extended to share resources between containers in a single pod,
  but supporting sharing between pods would need a completely new
  API similar to the one in this KEP.
- Using a resource that is expensive to initialize multiple times
  in different pods. This is not possible at the moment.
- Custom parameters that describe resource requirements and initialization.
  Parameters are not limited to a single, linear quantity that can be counted.
  With the current Pod API, annotations have to be used to capture such
  parameters and then hacks are needed to access them from a CSI driver or
  device plugin.

Support for new hardware will be provided by hardware vendor add-ons. Those add-ons
are responsible for reporting available resources in a format defined and
understood by Kubernetes and for configuring hardware before it is used. Kubernetes
handles the allocation of those resources as part of pod scheduling.

This KEP does not replace other means of requesting traditional resources
(RAM/CPU, volumes, extended resources). The scheduler will serve as coordinator
between the add-ons which own resources (CSI driver, resource driver) and the
resources owned and assigned by the scheduler (RAM/CPU, extended resources).

At a high-level, DRA with structured parameters takes the following form:

* DRA drivers publish their available resources in the form of a
  `ResourceSlice` object on a node-by-node basis according to one or more of the
  builtin "structured models" known to Kubernetes. This object is stored in the
  API server and available to the scheduler (or Cluster Autoscaler) to query
  when a resource request comes in later on.

* When a user wants to consume a resource, they create a `ResourceClaim`,
  which, in turn, references a claim parameters object. This object defines how
  many resources are needed and which capabilities they must have. Typically, it
  is defined using a vendor-specific type which might also support configuration
  parameters (i.e. parameters that are *not* needed for allocation but *are*
  needed for configuration).

* With such a claim in place, DRA drivers "resolve" the contents of any
  vendor-specific claim parameters into a canonical form (i.e. a generic
  `ResourceClaimParameters` object in the `resource.k8s.io` API group) which
  the scheduler (or Cluster Autoscaler) can evaluate against the
  `ResourceSlice` of any candidate nodes without knowing exactly what is
  being requested. They then use this information to help decide which node to
  schedule a pod on (as well as allocate resources from its `ResourceSlice`
  in the process).

* Once a node is chosen and the allocation decisions made, the scheduler will
  store the result in the API server as well as update its in-memory model of
  available resources. DRA drivers are responsible for using this allocation
  result to inject any allocated resource into the Pod, according to
  the resource choices made by the scheduler. This includes applying any
  configuration information attached to the vendor-specific claim parameters
  object used in the request.

This KEP is specifically focused on defining the framework necessary to enable
different "structured models" to be added to Kuberenetes over time. It is out of
scope to actually define one of these model themselves.

Instead, we provide an example of how one might map the way resources are
exposed by the traditional device-plugin API into a "structured model". We don't
believe this model is expressive enough to satify the majority of the use-cases
we want to cover with DRA, but it's useful enough to demonstrate the overall
"structured parameters" framework.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Originally, Kubernetes and its scheduler only tracked CPU and RAM as
resources for containers. Later, support for storage and discrete,
countable per-node extended resources was added. The kubelet device plugin
interface then made such local resources available to containers. But
for many newer devices, this approach and the Kubernetes API for
requesting these custom resources is too limited. This KEP may eventually
address limitations of the current approach for the following use cases:

- *Device initialization*: When starting a workload that uses
  an accelerator like an FPGA, I’d like to have the accelerator
  reconfigured or reprogrammed without having to deploy my application
  with full hardware access and/or root privileges. Running applications
  with less privileges is better for overall security of the cluster.

  *Limitation*: Currently, it’s impossible to specify the desired
  device properties that are required for reconfiguring devices.
  For the FPGA example, a file containing the desired configuration
  of the FPGA has to be referenced.

- *Device cleanup*: When my workload is finished, I would like to have
  a mechanism for cleanup of the device, that will ensure that device
  does not contain traces/parameters/data from previous workloads and
  appropriate power state/shutdown. For example, an FPGA might have
  to be reset because its configuration for the workload was
  confidential.

  *Limitation*: Post-stop actions are not supported.

- *Partial allocation*: When workloads use only a portion of the device
  capabilities, devices can be partitioned (e.g. using Nvidia MIG or SR-IOV) to
  better match workload needs. Sharing the devices in this way can greatly
  increase HW utilization / reduce costs.

- *Limitation*: currently there's no API to request partial device
  allocation. With the current device plugin API, devices need to be
  pre-partitioned and advertised in the same way a full / static devices
  are. User must then select a pre-partitioned device instead of having one
  created for them on the fly based on their particular resource
  constraints. Without the ability to create devices dynamically (i.e. at the
  time they are requested) the set of pre-defined devices must be carefully
  tuned to ensure that device resources do not go unused because some of the
  pre-partioned devices are in low-demand. It also puts the burden on the user
  to pick a particular device type, rather than declaring the resource
  constraints more abstractly.

- *Optional allocation*: When deploying a workload I’d like to specify
  soft(optional) device requirements. If a device exists and it’s
  allocatable it will be allocated. If not - the workload will be run on
  a node without a device. GPU and crypto-offload engines are
  examples of this kind of device. If they’re not available, workloads
  can still run by falling back to using only the CPU for the same
  task.

  *Limitation*: Optional allocation is supported neither by the device
  plugins nor by current Pod resource declaration.

- *Support Over the Fabric devices*: When deploying a container, I’d
  like to utilize devices available over the Fabric (network, special
  links, etc).

  *Limitation*: The device plugin API is designed for node-local resources that
  get discovered by a plugin running on the node. Projects like
  [Akri](https://www.cncf.io/projects/akri/) have to work around that by
  reporting the same network-attached resource on all nodes that it could
  get attached to and then updating resource availability on all of those
  nodes when resources get used.

Several other limitations are addressed by
[CDI](https://github.com/container-orchestrated-devices/container-device-interface/),
a container runtime extension that this KEP is using to expose resources
inside a container.

### Goals

- Enable cluster autoscaling when pods use resource claims, with correct
  decisions and changing the cluster size by more than one node at a time.

- Support node-local resources

- Support claim parameters that are specified in a vendor CRD as
  an alternative to letting users directly specify parameters with
  the in-tree type. This provides a user experience that is similar to
  what has been possible since Kubernetes 1.26. Ideally, users should not notice
  at all that a driver is using structured parameters under the hood.

- Enable abstraction layers for resource requests (= "give me a network SR-IOV,
  no matter which hardware provides it"). See also the
  [Resource Class Proposal](https://docs.google.com/document/d/1qKiIVs9AMh2Ua5thhtvWqOqW0MSle_RV3lfriO1Aj6U/edit#heading=h.jzfmfdca34kj)
  (access is through some Google groups or ask the authors).

### Non-Goals

* Replace the device plugin API. For resources that fit into its model
  of a single, linear quantity it is a good solution. Other resources
  should use dynamic resource allocation. Both are expected to co-exist, with vendors
  choosing the API that better suits their needs on a case-by-case
  basis. Because the new API is going to be implemented independently of the
  existing device plugin support, there's little risk of breaking stable APIs.

* Define specific abstraction layers for certain domains

* Support network-attached resources

## Proposal

### User Stories

#### Cluster add-on development

As a hardware vendor, I want to make my hardware available also to applications
that run in a container under Kubernetes. I want to make it easy for a cluster
administrator to configure a cluster where some nodes have this hardware.

I develop two components, one that runs as part of the Kubernetes control plane
and one that runs on each node, and package those inside container images. YAML
files describe how to deploy my software on a Kubernetes cluster that supports
dynamic resource allocation.

Documentation for administrators explains how the nodes need to be set
up. Documentation for users explains which parameters control the behavior of
my hardware and how to use it inside a container.

#### Cluster configuration

As a cluster administrator, I want to make GPUs from vendor ACME available to users
of that cluster. I prepare the nodes and deploy the vendor's components with
`kubectl create`.

I create a ResourceClass for the hardware with parameters that only I as the
administrator am allowed to choose, like for example running a command with
root privileges that does some cluster-specific initialization for each allocation:
```
apiVersion: gpu.example.com/v1
kind: GPUInit
metadata:
  name: acme-gpu-init
# DANGER! This option must not be accepted for
# user-supplied parameters. A real driver might
# not even allow it for admins. This is just
# an example to show the conceptual difference
# between ResourceClass and ResourceClaim
# parameters.
initCommand:
- /usr/local/bin/acme-gpu-init
- --cluster
- my-cluster
---
apiVersion: core.k8s.io/v1alpha2
kind: ResourceClass
metadata:
  name: acme-gpu
parametersRef:
  apiGroup: gpu.example.com
  kind: GPUInit
  name: acme-gpu-init
```

#### Partial GPU allocation

As a user, I want to use a GPU as accelerator, but don't need exclusive access
to that GPU. Running my workload with just 2Gb of memory is sufficient. This is
supported by the ACME GPU hardware. I know that the administrator has created
an "acme-gpu" ResourceClass.

For a simple trial, I create a Pod directly where two containers share the same subset
of the GPU:
```
apiVersion: gpu.example.com/v1
kind: GPURequirements
metadata:
  name: device-consumer-gpu-parameters
memory: "2Gi"
---
apiVersion: resource.k8s.io/v1alpha2
kind: ResourceClaimTemplate
metadata:
  name: device-consumer-gpu-template
spec:
  metadata:
    # Additional annotations or labels for the
    # ResourceClaim could be specified here.
  spec:
    resourceClassName: "acme-gpu"
    parametersRef:
      apiGroup: gpu.example.com
      kind: GPURequirements
      name: device-consumer-gpu-parameters
---
apiVersion: v1
kind: Pod
metadata:
  name: device-consumer
spec:
  resourceClaims:
  - name: "gpu" # this name gets referenced below under "claims"
    template:
      resourceClaimTemplateName: device-consumer-gpu-template
  containers:
  - name: workload
    image: my-app
    command: ["/bin/program"]
    resources:
      requests:
        memory: "64Mi"
        cpu: "250m"
      limits:
        memory: "128Mi"
        cpu: "500m"
      claims:
        - "gpu"
  - name: monitor
    image: my-app
    command: ["/bin/other-program"]
    resources:
      requests:
        memory: "32Mi"
        cpu: "25m"
      limits:
        memory: "64Mi"
        cpu: "50m"
      claims:
      - "gpu"
```

This request triggers resource allocation on a node that has a GPU device with
2Gi of memory available and then the Pod runs on that node. The remaining
capacity of the GPU may be usable for other pods, with constrains like alignment
to segment sizes ensured by the resource driver.
The lifecycle of the resource
allocation is tied to the lifecycle of the Pod.

In production, a similar PodTemplateSpec in a Deployment will be used.

### Publishing node resources

The resources available on a node need to be published to the API server. In
the typical case, this is expected to be published by the on-node driver via
the kubelet, as described below. However, the source of this data may vary; for
example, a cloud provider controller could populate this based upon information
from the cloud provider API.

In the kubelet case, each kubelet publishes kubelet publishes a set of
`ResourceSlice` objects to the API server with content provided by the
corresponding DRA drivers running on its node. Access control through the node
authorizer ensures that the kubelet running on one node is not allowed to
create or modify `ResourceSlices` belonging to another node. A `nodeName`
field in each `ResourceSlice` object is used to determine which objects are
managed by which kubelet.

**NOTE:**  `ResourceSlices` are published separately for each driver, using
whatever version of the `resource.k8s.io` API is supported by the kubelet. That
same version is then also used in the gRPC interface between the kubelet and
the DRA drivers providing content for those objects. It might be possible to
support version skew (= keeping kubelet at an older version than the control
plane and the DRA drivers) in the future, but currently this is out of scope.

Embedded inside each `ResourceSlice` is the representation of the resources
managed by a driver according to a specific "structured model". In the example
seen below, the structured model in use is called `namedResources`:

```yaml
kind: ResourceSlice
apiVersion: resource.k8s.io/v1alpha2
...
spec:
  nodeName: worker-1
  driverName: cards.dra.example.com
  namedResources:
    ...
```

Such a model could be created to represent resources in a manner similar to the
opaque strings passed over the tradition device plugin API to the kubelet. The
one addition being that each named resource can have a set of arbitrary
attributes attached to it.

If a driver wanted to use a different structured model to represent its resources,
a new structured model would need to be defined inside Kuberenetes, and a field
would need to be added to this struct at the same level as
`namedResources`. Driver implementors would then have the option
to set this new field instead.

**Note:** If a new model is added to the schema but clients are not updated,
they'll encounter an object with no information from any known structured model
when they serialize into their known version of a `ResourceSlice`. This
tells them that they cannot handle the object because the API has been extended.

Drivers can use different structured models by publishing multiple
`ResourceSlice` objects, as long as each model represents a distinct set of
resources. Whether the information about resources of one particular structured
model must fit into one ResourceSlice object (or be distributed across
many) depends on how that particular structured model describes its resources. In
all cases, the size of each object is a hard limit and one must take this into
account when designing a structured model and preparing ResourceSlice objects
for it.

Below is an example of a driver that provides two discrete GPU cards using the
`namedResources` model described above:

```yaml
kind: ResourceSlice
apiVersion: resource.k8s.io/v1alpha2
...
spec:
  nodeName: worker-1
  driverName: cards.dra.example.com
  namedResources:
  - name: gpu-0
    attributes:
    - name: type
      string: GPU # All named resources with this type have the following attributes.
    - name: UUID
      string: GPU-ceea231c-4257-7af7-6726-efcb8fc2ace9
    - name: driverVersion
      string: 1.2.3
    - name: runtimeVersion
      string: 11.1.42
    - name: memory
      quantity: 16Gi
    - name: productName
      string: ACME T1000 32GB
    - name: isGPU.gpu.k8s.io
      bool: true
  - name: gpu-1
    attributes:
    - name: type
      string: GPU
    - name: UUID
      string: GPU-6aa0af9e-a2be-88c8-d2b3-2240d25318d7
    - name: driverVersion
      string: 1.2.3
    - name: runtimeVersion
      string: 11.1.42
    - name: memory
      quantity: 32Gi
    - name: productName
      string: ACME A4-PCIE-40GB
    - name: isGPU.gpu.k8s.io
      bool: true
```

Where "gpu-0" represents one type of card and "gpu-1" represents another (with
the attributes hanging off each serving to "define" their individual
properties).

Compared to labels, attributes in this model have values of exactly one type. As
described later on, these attributes can be used in CEL expressions to select a
specific resource for allocation on a node.

While this model is still hypothetical, we do imagine real-world models
attaching attributes to their resources in a similar way. To avoid any future
conflicts, we plan to reserve any attributes with the ".k8s.io" suffix for
future use and standardization by Kubernetes. This could be used to describe
topology across resources from different vendors. Here `isGPU.gpu.k8s.io`
specifies that the instance is a GPU. This is just fictional example, this KEP
does not define any standardized attributes.

**Note:** If a driver needs to reduce resource capacity, then there is a risk
that a claim gets allocated using that capacity while the kubelet is updating a
`ResourceSlice`. The implementations of structured models must handle
scenarios where more resources are allocated than available. The kubelet plugin
of a DRA driver ***must*** double-check that the allocated resources are still
available when NodePrepareResource is called. If not, the pod cannot start until
the resource comes back. Treating this as a fatal error during pod admission
would allow us to delete the pod and trying again with a new one.

### Using structured parameters

The following is an example CRD which the developer of the
`cards.dra.example.com` DRA driver might define as a valid claim parameters
object for requesting access to its GPUs:

```yaml
kind: CardParameters
apiVersion: dra.example.com/v1alpha1
metadata:
  name: my-parameters
  namespace: user-namespace
  uid: foobar-uid
...
spec:
  numGPUs: 2
  minimumRuntimeVersion: v12.0.0
  minimumMemory: 32Gi
  # "sharing" is a configuration parameter that does not
  # get translated into the selector below.
  sharing:
    strategy: TimeSliced
```

Note that all fields in this CRD can be fully validated since it is owned by
the DRA driver itself. This includes value ranges that are specific to the
underlying hardware. There's no risk of using invalid attribute names because
only the fields shown here are valid.

With this CRD in place, a DRA driver controller is able to convert instances of
it into a generic, in-tree `ResourceClaimParameters` object that the scheduler
is able to understand.

For the example above, the converted object would look as follows:

```yaml
kind: ResourceClaimParameters
apiVersion: resource.k8s.io/v1alpha2

metadata:
  # This cannot be the same as my-parameters because parameter objects with a different
  # type might also use it. Instead, the original object gets linked to below.
  name: someArbitraryName
  namespace: user-namespace

generatedFrom:
  name: my-parameters
  kind: CardParameters
  apiGroup: dra.example.com
  uid: foobar-uid

# Configuration parameters can be provided at two different levels:
# - for all resources (this example)
# - for a single resource (example below)
#
# At the moment, only configuration parameters defined by a vendor
# are supported. In-tree definition of common configuration parameters
# might get added in the future.
config:
  vendor:
  # The driver name specifies which driver the list entry is intended
  # for. This is a list because a claim might end up using resources
  # from different drivers.
  #
  # A driver could provide an admission webhook to validate its
  # own parameters when users embed them in a ResourceClaimParameters
  # object themselves.
  - driverName: cards.dra.example.com
    parameters:
      # Beware that ResourceClaimParameters have separate RBAC rules than
      # the vendor CRD, so information included here may get visible
      # to more users than the original CRD. Both objects are in the same
      # namespace.
      kind: CardClaimConfiguration
      apiVersion: dra.example.com/v1alpha1
      sharing:
        strategy: TimeSliced

requests:
- namedResource:
    selector: |-
      # Selectors are CEL expressions with access to the attributes of the named resource
      # that is being checked for a match. By default, all named resources are checked,
      # regardless of which driver provides them.
      #
      # Attribute names need to be fully qualified.
      attributes.string.has("type.cards.dra.example.com") &&
      attributes.string["type.cards.dra.example.com"] == "GPU" &&
      attributes.version["runtimeVersion.cards.dra.example.com"].isGreaterThan(semver("12.0.0")) &&
      attributes.quantity["memory.cards.dra.example.com"].isGreaterThan(quantity("32Gi"))
- namedResource:
    driverName: "cards.dra.example.com"
    selector: |-
      # Matching can be restricted to named resources provided by a specific driver.
      # In that case, attribute names can be used without the driver name as suffix
      # and the existence of attributes doesn't have to be checked.
      attributes.version["runtimeVersion"].isGreaterThan(semver("12.0.0")) &&
      attributes.quantity["memory"].isGreaterThan(quantity("32Gi"))
```

The meaning is that the selector expression must evaluate to true for a
particular named resource which has attributes as defined by
`cards.dra.example.com`, like "runtimeVersion" and "memory". Checking for the
type once rules out instances where the "GPU" attributes of the vendor are
not set. Accessing unset attributes is a runtime error.

Future extensions could be added to support partioning of resources as well as a
express constraints that must be satisfied *between* any selected resources. For
example, selecting two cards which are on the same PCI root complex may be
needed to get the required performance.

Instead of defining a vendor-specific CRD, DRA driver authors (or
administrators) could decide to allow users to create and reference
`ResourceClaimParameters` directly within their `ResourceClaims`. This would
avoid the translation step shown above, but at the cost of (1) providing per-
claim configuration parameters for their requested resources, and (2) doing any
sort of validation on the CEL expressions created by the user.

Separate KEPs could be used to standardize attribute names of resources and
what resources with those attributes have to provide. With the fictional
`isGPU.gpu.k8s.io` the following parameters would request "one GPU":

```yaml
kind: ResourceClaimParameters
apiVersion: resource.k8s.io/v1alpha2

metadata:
  name: gpu-request-parameters
  namespace: user-namespace

requests:
- namedResource:
    selector: |-
      attributes.bool.has("isGPU.gpu.k8s.io") &&
      attributes.bool["isGPU.gpu.k8s.io"]
```

Vendor parameters could be used here, too. Here they get set at the level of
a single resource instance, with different parameters for different vendors:

```yaml
kind: ResourceClaimParameters
apiVersion: resource.k8s.io/v1alpha2

metadata:
  name: gpu-request-parameters
  namespace: user-namespace

requests:
- namedResource:
    selector: |-
      attributes.bool.has("isGPU.gpu.k8s.io") &&
      attributes.bool["isGPU.gpu.k8s.io"]
    config:
      vendor:
      - driverName: cards.dra.example.com
        parameters: <per-GPU parameters defined by the "cards" vendor>
      - driverName: cards.someothervendor.example.com
        parameters: <per-GPU parameters defined by the "someothervendor">
```


Resource class parameters are supported the same way. To ensure that
permissions can be limited to administrators, there's a separate cluster-scoped
ResourceClassParameters type. Instead of individual requests, one additional
selector can be specified there which then also must be true for all individual
requests made with that class:

```yaml
kind: ResourceClassParameters
apiVersion: resource.k8s.io/v1alpha2

metadata:
  name: someArbitraryName

generatedFrom:
  name: gpu-parameters
  kind: CardClassParameters
  apiGroup: dra.example.com
  uid: foobar-uid

config:
  vendor:
  - driverName: cards.dra.example.com
    parameters: <parameters with a type defined by that vendor>

filter:
  namedResources:
    driverName: cards.dra.example.com
    selector: |-
      attributes.quantity["memory"] <= "16Gi"
```

In this example, the additional selector expression limits users of this class
to just the cards with less that "16Gi" of memory.

These class parameters are defined so that they select devices from one
particular vendor. It is also possible to define classes that are independent
of any particular vendor:
- `config.vendor` is a list which can contain different entries. Only the entry
  for the driver which provides the allocated resources gets passed on to
  that driver.
- The `driverName` in `filter.namedResources` can be left out and the `selector`
  can filter based on vendor-neutral attributes.

### Communicating allocation to the DRA driver

The scheduler decides which resources to use for a claim and how much of
them. It also needs to pass through the opaque vendor parameters, if there are
any. This accurately captures the configuration parameters as they were set
at the time of allocation.

All of this information gets stored in the allocation result inside the
ResourceClaim status. For the example above, the result produced by the
scheduler is simply the list of IDs of the selected named resource:

```yaml
# Matches with the StructuredResourceHandle Go type defined below.
adminConfig:
  ...
userConfig:
  vendor:
    kind: CardClaimConfiguration
      apiVersion: dra.example.com/v1alpha1
      sharing:
        strategy: TimeSliced

nodeName: worker-1
results:
- namedResources:
    name: gpu-1
```

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

#### Feature not used

In a cluster where the feature is not used (no resource driver installed, no
pods using dynamic resource allocation) the impact is minimal, both for
performance and security. The scheduler plugin will
return quickly without doing any work for pods.

#### Compromised node

Kubelet is intentionally limited to read-only access for ResourceClass and ResourceClaim
to prevent that a
compromised kubelet interferes with scheduling of pending pods, for example
by updating status information normally set by the scheduler.
Faking such information could be used for a denial-of-service
attack against pods using those ResourceClaims, for example by overwriting
their allocation result with a node selector that matches no node. A
denial-of-service attack against the cluster and other pods is harder, but
still possible. For example, frequently updating ResourceSlice objects could
cause new scheduling attempts for pending pods.

Another potential attack goal is to get pods with sensitive workloads to run on
a compromised node. For pods that don't use special resources nothing changes
in that regard. Such an attack is possible for pods with extended resources
because kubelet is in control of which capacity it reports for those: it could
publish much higher values than the device plugin reported and thus attract
pods to the node that normally would run elsewhere. With dynamic resource
allocation, such an attack is still possible, but the attack code would have to
be different for each resource driver because all of them will use structured
parameters differently for reporting resource availability.

#### Compromised resource driver plugin

This is the result of an attack against the resource driver, either from a
container which uses a resource exposed by the driver, a compromised kubelet
which interacts with the plugin, or due to resource driver running on a node
with a compromised root account.

The resource driver plugin only needs read access to objects described in this
KEP, so compromising it does not interfere with dynamic resource allocation for
other drivers.

A resource driver may need root access on the node to manage
hardware. Attacking the driver therefore may lead to root privilege
escalation. Ideally, driver authors should try to avoid depending on root
permissions and instead use capabilities or special permissions for the kernel
APIs that they depend on.

A resource driver may also need privileged access to remote services to manage
network-attached devices. Resource driver vendors and cluster administrators
have to consider what the effect of a compromise could be for that and how such
privileges could get revoked.

#### User permissions and quotas

Similar to generic ephemeral inline volumes, the [ephemeral resource use
case](#ephemeral-vs-persistent-resourceclaims-lifecycle) gets covered by
creating ResourceClaims on behalf of the user automatically through
kube-controller-manager. The implication is that RBAC rules that are meant to
prevent creating ResourceClaims for certain users can be circumvented, at least
for ephemeral resources. Administrators need to be aware of this caveat when
designing user restrictions.

A quota system that is based on the information in the structured parameter model
could be implemented in Kubernetes. When a user has exhausted their
quota, the scheduler then refuses to allocate further ResourceClaims.

#### Usability

Aside from security implications, usability and usefulness of dynamic resource
allocation also may turn out to be insufficient. Some risks are:

- Slower pod scheduling due to more complex decision making.

- Additional complexity when describing pod requirements because
  separate objects must be created for the parameters.

All of these risks will have to be evaluated by gathering feedback from users
and resource driver developers.

## Design Details

### Components

![components](./components.png)

Several components must be implemented or modified in Kubernetes:
- The new API must be added to kube-apiserver.
- A new controller in kube-controller-manager which creates
  ResourceClaims from ResourceClaimTemplates, similar to
  https://github.com/kubernetes/kubernetes/tree/master/pkg/controller/volume/ephemeral.
  It also removes the reservation entry for a consumer in `claim.status.reservedFor`,
  the field that tracks who is allowed to use a claim, when that user no longer exists.
  It clears the allocation and thus makes the underlying resources available again
  when a ResourceClaim is no longer reserved.
- A kube-scheduler plugin must detect Pods which reference a
  ResourceClaim (directly or through a template) and ensure that the
  resource is allocated before the Pod gets scheduled, similar to
  https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/volume/scheduling/scheduler_binder.go
- Kubelet must be extended to retrieve information from ResourceClaims
  and to call a resource kubelet plugin. That plugin returns CDI device ID(s)
  which then must be passed to the container runtime.

A resource driver can have the following components:
- *CRD controller* (optional): a central component which translates parameters
  defined with a vendor CRD into in-tree parameter types. 
- *kubelet plugin* (required): a component which cooperates with kubelet to
  publish resource information and to prepare the usage of the resource on a node.

When a resource driver doesn't use its own CRD for parameters, the CRD controller
is not needed and a ResourceClaim references ResourceClaimParameters directly.

A [utility library](https://github.com/kubernetes/kubernetes/tree/master/staging/src/k8s.io/dynamic-resource-allocation) for resource drivers was developed.
It does not have to be used by drivers, therefore it is not described further
in this KEP.

### State and communication

A ResourceClaim object defines what kind of resource is needed and what
the parameters for it are. It is owned by users and namespaced. Additional
parameters are provided by a cluster admin in ResourceClass objects.

The ResourceClaim spec is immutable. The ResourceClaim
status is reserved for system usage and holds the current state of the
resource. The status must not get lost, which in the past was not ruled
out. For example, status could have been stored in a separate etcd instance
with lower reliability. To recover after a loss, status was meant to be recoverable.
A [recent KEP](https://github.com/kubernetes/enhancements/tree/master/keps/sig-architecture/2527-clarify-status-observations-vs-rbac)
clarified that status will always be stored reliably and can be used as
proposed in this KEP.

Handling state and communication through objects has two advantages:
- Changes for a resource are (almost) atomic, which avoids race conditions.
  One small exception is that changing finalizers and the status have to
  be done in separate operations.
- The only requirement for deployments is that the components can connect to
  the API server.

The entire state of a resource can be determined by looking at its
ResourceClaim (see [API below](#api) for details), for example:

- It is **allocated** if and only if `claim.status.allocation` is non-nil and
  points to the `AllocationResult`, i.e. the struct where information about
  a successful allocation is stored.

- It is in use if and only if `claim.status.reservedFor` contains one or
  more consumers. It does not matter whether those users, usually pods, are
  currently running because that could change at any time.

- A resource is no longer needed when `claim.deletionTimestamp` is set. It must not
  be deallocated yet when it is still in use.

Some of the race conditions that need to be handled are:

- A ResourceClaim gets created and deleted again while the scheduler
  is allocating it. Before it actually starts doing anything, the
  scheduler adds a finalizer. Either adding the finalizer or removing the
  ResourceClaim win. If the scheduler wins, it continues with allocation
  and can either complete or abort the operation when it notices the non-nil
  DeletionTimestamp. Otherwise, allocation gets aborted immediately.

  What this avoids is the situation where an allocation succeed without having
  an object where the result can be stored. The driver can also be killed at
  any time: when it restarts, the finalizer indicates that allocation may be in
  progress and has to be completed or aborted.

  However, users may still force-delete a ResourceClaim, or the entire
  cluster might get deleted. Driver implementations must store enough
  information elsewhere to detect when some allocated resource is no
  longer needed to recover from such scenarios.

- A ResourceClaim gets deleted and recreated while the resource driver is
  adding the finalizer. The driver can update the object to add the finalizer
  and then will get a conflict error, which informs the driver that it must
  work on a new instance of the claim. In general, patching a ResourceClaim
  is only acceptable when it does not lead to race conditions. To detect
  delete+recreate, the UID must be added as precondition for a patch.
  To detect also potentially conflicting other changes, ResourceVersion
  needs to be checked, too.

- In a cluster with multiple scheduler instances, two pods might get
  scheduled concurrently by different schedulers. When they reference
  the same ResourceClaim which may only get used by one pod at a time,
  only one pod can be scheduled.

  Both schedulers try to add their pod to the `claim.status.reservedFor` field, but only the
  update that reaches the API server first gets stored. The other one fails
  with a conflict error and the scheduler which issued it knows that it must
  put the pod back into the queue, waiting for the ResourceClaim to become
  usable again.

- Two pods get created which both reference the same unallocated claim with
  delayed allocation. A single scheduler can detect this special situation
  and then do allocation only for one of the two pods. When the pods
  are handled by different schedulers, only one will succeed with writing
  back the `claim.status.allocation`.

- Scheduling a pod and allocating resources for it has been attempted, but one
  claim needs to be reallocated to fit the overall resource requirements. A second
  pod gets created which references the same claim that is in the process of
  being deallocated. Because that is visible in the claim status, scheduling
  of the second pod cannot proceed.

### Custom parameters

To support arbitrarily complex parameters, both ResourceClass and ResourceClaim
contain one field which references a separate object. The reference contains
API group, kind and name and thus is sufficient for generic clients to
retrieve the parameters. For ResourceClass, that object must be
cluster-scoped. For ResourceClaim, it must be in the same namespace as the
ResourceClaim and thus the Pod. Which kind of objects a resource driver accepts as parameters depends on
the driver.

This approach was chosen because then validation of the parameters can be done
with a CRD and that validation will work regardless of where the parameters
are needed.

It is the responsibility of the resource driver to convert these CRD parameters
into in-tree ResourceClaimParameters and ResourceClassParameters. Kubernetes
finds those generated parameters based on their `generatedFrom` back-reference.

Parameters may get deleted before the ResourceClaim or ResourceClass that
references them. In that case, a pending resource cannot be allocated until the
parameters get recreated. An allocated resource must remain usable and
deallocating it must be possible. To support this, resource drivers must copy
all relevant information:
- For usage, the `claim.status.allocation.resourceHandle` can be hold some copied information
  because the ResourceClaim and thus this field must exist.
- For deallocation, drivers should use some other location to handle
  cases where a user force-deletes a ResourceClaim or the entire
  cluster gets removed.

### Sharing a single ResourceClaim

Pods reference resource claims in a new `pod.spec.resourceClaims` list. Each
resource in that list can then be made available to one or more containers in
that Pod. Depending on the capabilities defined in the
`claim.status.allocation` by the driver, a ResourceClaim can be used exclusively
by one pod at a time or an unlimited number of pods. Support for additional
constraints (maximum number of pods, maximum number of nodes) could be
added once there are use cases for those.

Consumers of a ResourceClaim are listed in `claim.status.reservedFor`. They
don't need to be Pods. At the moment, Kubernetes itself only handles Pods and
allocation for Pods.

### Ephemeral vs. persistent ResourceClaims lifecycle

A persistent ResourceClaim has a lifecyle that is independent of any particular
pod. It gets created and deleted by the user. This is useful for resources
which are expensive to configure and that can be used multiple times by pods,
either at the same time or one after the other. Such persistent ResourceClaims
get referenced in the pod spec by name. When a PodTemplateSpec in an app
controller spec references a ResourceClaim by name, all pods created by that
controller also use that name and thus share the resources allocated for that
ResourceClaim.

But often, each Pod is meant to have exclusive access to its own ResourceClaim
instance instead. To support such ephemeral resources without having to modify
all controllers that create Pods, an entry in the new PodSpec.ResourceClaims
list can also be a reference to a ResourceClaimTemplate. When a Pod gets created, such a
template will be used to create a normal ResourceClaim with the Pod as owner
with an
[OwnerReference](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#OwnerReference)),
and then the normal allocation of the resource takes place. Once the pod got
deleted, the Kubernetes garbage collector will also delete the
ResourceClaim.

This mechanism documents ownership and serves as a fallback for scenarios where
dynamic resource allocation gets disabled in a cluster (for example, during a
downgrade). But it alone is not sufficient: for example, the job controller
does not delete pods immediately when they have completed, which would keep
their resources allocated. Therefore the resource controller watches for pods
that have completed and releases their resource allocations.

The difference between persistent and ephemeral resources for kube-scheduler
and kubelet is that the name of the ResourceClaim needs to be determined
differently: the name of an ephemeral ResourceClaim is recorded in the Pod status.
Ownership must be checked to detect accidental conflicts with
persistent ResourceClaims or previous incarnations of the same ephemeral
resource.

### Scheduled pods with unallocated or unreserved claims

There are several scenarios where a Pod might be scheduled (= `pod.spec.nodeName`
set) while the claims that it depends on are not allocated or not reserved for
it:

* A user might manually create a pod with `pod.spec.nodeName` already set.
* Some special cluster might use its own scheduler and schedule pods without
  using kube-scheduler.
* The feature might have been disabled in kube-scheduler while scheduling
  a pod with claims.

The kubelet is refusing to run such pods and reports the situation through
an event (see below). It's an error scenario that should better be avoided.

Users should avoid this situation by not scheduling pods manually. If they need
it for some reason, they can use a node selector which matches only the desired
node and then let kube-scheduler do the normal scheduling.

Custom schedulers should emulate the behavior of kube-scheduler and ensure that
claims are allocated and reserved before setting `pod.spec.nodeName`.

The last scenario might occur during a downgrade or because of an
administrator's mistake. Administrators can fix this by deleting such pods.

### Handling non graceful node shutdowns

When a node is shut down unexpectedly and is tainted with an `out-of-service`
taint with NoExecute effect as explained in the [Non graceful node shutdown KEP](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/2268-non-graceful-shutdown),
all running pods on the node will be deleted by the GC controller and the
resources used by the pods will be deallocated. However, they will not be
un-prepared as the node is down and Kubelet is not running on it.

Resource drivers should be able to handle this situation correctly and
should not expect `UnprepareNodeResources` to be always called.
If resources are unprepared when `Deallocate` is called, `Deallocate`
might need to perform additional actions to correctly deallocate
resources.

### API

```
<<[UNRESOLVED @pohly @johnbelamaric]>>
Before 1.31, we need to re-evaluate the API, including, but not limited to:
- Do we really need a separate ResourceClaim?
- Does "Device" instead of "Resource" make the API easier to understand?
- Avoid separate parameter objects if and when possible.
<<[/UNRESOLVED]>>
```

The PodSpec gets extended. To minimize the changes in core/v1, all new types
get defined in a new resource group. This makes it possible to revise those
more experimental parts of the API in the future. The new fields in the
PodSpec are gated by the DynamicResourceAllocation feature gate and can only be
set when it is enabled. Initially, they are declared as alpha. Even though they
are alpha, changes to their schema are discouraged and would have to be done by
using new field names.

ResourceClaim, ResourceClass and ResourceClaimTemplate are new built-in types
in `resource.k8s.io/v1alpha2`. This alpha group must be explicitly enabled in
the apiserver's runtime configuration. Using builtin types was chosen instead
of using CRDs because core Kubernetes components must interact with the new
objects and installation of CRDs as part of cluster creation is an unsolved
problem.

Secrets are not part of this API: if a resource driver needs secrets, for
example to access its own backplane, then it can define custom parameters for
those secrets and retrieve them directly from the apiserver. This works because
drivers are expected to be written for Kubernetes.

#### resource.k8s.io

##### ResourceSlice

For each node, one or more ResourceSlice objects get created. The kubelet
publishes them with the node as the owner, so they get deleted when a node goes
down and then gets removed.

All list types are atomic because that makes tracking the owner for
server-side-apply (SSA) simpler. Patching individual list elements is not
needed and there is a single owner (kubelet).

```go
// ResourceSlice provides information about available
// resources on individual nodes.
type ResourceSlice struct {
    metav1.TypeMeta
    // Standard object metadata
    metav1.ObjectMeta

    // NodeName identifies the node which provides the resources
    // if they are local to a node.
    //
    // A field selector can be used to list only ResourceSlice
    // objects with a certain node name.
    NodeName string

    // DriverName identifies the DRA driver providing the capacity information.
    // A field selector can be used to list only ResourceSlice
    // objects with a certain driver name.
    DriverName string

    ResourceModel
}
```

The ResourceSlice object holds the
information about available resources. A status is not strictly needed because
the information in the allocated claim statuses is sufficient to determine
which of those resources are reserved for claims.

However, despite the finalizer on the claims it could happen that a well
intentioned but poorly informed user deletes a claim while it is in use.
Therefore adding a status is a useful future extension. That status will
include information about reserved resources (set by schedulers before
allocating a claim) and in-use resources (set by the kubelet). This then
enables conflict resolution when multiple schedulers schedule pods to the same
node because they would be required to set a reservation before proceeding with
the allocation. It also enables detecting inconsistencies and taking actions to
fix those, like deleting pods which use a deleted claim.

At the moment, there is a single structured model. To enable adding alternative
models in the future, one-of-many structs are used. If a component encounters
such a struct with no known field set, it knows that it cannot handle the
struct because some newer, unsupported model is used:

```go
// ResourceModel must have one and only one field set.
type ResourceModel struct {
    // NamedResources describes available resources using the named resources model.
    NamedResources *NamedResourcesResources
}
```

The "named resources" model lists individual resource instances and their
attributes:

```go
// NamedResourcesResources is used in NodeResourceModel.
type NamedResourcesResources struct {
    // The list of all individual resources instances currently available.
    Instances []NamedResourcesInstance
}

// NamedResourcesInstance represents one individual hardware instance that can be selected based
// on its attributes.
type NamedResourcesInstance struct {
    // Name is unique identifier among all resource instances managed by
    // the driver on the node. It must be a DNS subdomain.
    Name string

    // Attributes defines the attributes of this resource instance.
    // The name of each attribute must be unique.
    Attributes []NamedResourcesAttribute
}

// NamedResourcesAttribute is a combination of an attribute name and its value.
type NamedResourcesAttribute struct {
    // Name must be a DNS subdomain. If the name contains no dot, the name of
    // the driver which provides the instance gets added as suffix.
    Name string

    NamedResourcesAttributeValue
}

// NamedResourcesAttributeValue must have one and only one field set.
type NamedResourcesAttributeValue struct {
    // QuantityValue is a quantity.
    QuantityValue *resource.Quantity
    // BoolValue is a true/false value.
    BoolValue *bool
    // IntValue is a 64-bit integer.
    IntValue *int64
    // IntSliceValue is an array of 64-bit integers.
    IntSliceValue *NamedResourcesIntSlice
    // StringValue is a string.
    StringValue *string
    // StringSliceValue is an array of strings.
    StringSliceValue *NamedResourcesStringSlice
    // VersionValue is a semantic version according to semver.org spec 2.0.0.
    VersionValue *string
}

// NamedResourcesIntSlice contains a slice of 64-bit integers.
type NamedResourcesIntSlice struct {
    // Ints is the slice of 64-bit integers.
    Ints []int64
}

// NamedResourcesStringSlice contains a slice of strings.
type NamedResourcesStringSlice struct {
    // Strings is the slice of strings.
    Strings []string
}
```

All names must be DNS sub-domains. This excludes the "/" character, therefore
combining different names with that separator to form an ID is valid.

In the Go types above, all structs starting with `NamedResources` are part of
that structured model. Code generators (more specifically, the applyconfig
generator) assume that all Go types of an API are defined in the same Go
package. If it wasn't for that, defining those structs in their own package
without the `NamedResources` prefix would be possible and make the Go code
cleaner without affecting the Kubernetes API.

##### ResourceClass

```go
// ResourceClass is used by administrators to influence how resources
// are allocated.
//
// This is an alpha type and requires enabling the DynamicResourceAllocation
// feature gate.
type ResourceClass struct {
    metav1.TypeMeta
    // Standard object metadata
    // +optional
    metav1.ObjectMeta

    // ParametersRef references an arbitrary separate object that may hold
    // parameters that will be used by the driver when allocating a
    // resource that uses this class. A dynamic resource driver can
    // distinguish between parameters stored here and and those stored in
    // ResourceClaimSpec.
    // +optional
    ParametersRef *ResourceClassParametersReference

    // Only nodes matching the selector will be considered by the scheduler
    // when trying to find a Node that fits a Pod when that Pod uses
    // a ResourceClaim that has not been allocated yet.
    //
    // Setting this field is optional. If null, all nodes are candidates.
    // +optional
    SuitableNodes *core.NodeSelector

    // DefaultClaimParametersRef is an optional reference to an object that holds parameters
    // used as default when allocating a claim which references this class. This field is utilized
    // only when the ParametersRef of the claim is nil. If both ParametersRef
    // and DefaultClaimParametersRef are nil, the claim requests no resources and thus
    // can always be allocated.
    // +optional
    DefaultClaimParametersRef *ResourceClassParametersReference
}
```

##### ResourceClassParameters

```go
// ResourceClassParameters defines resource requests for a ResourceClass in an
// in-tree format understood by Kubernetes.
type ResourceClassParameters struct {
    metav1.TypeMeta
    // Standard object metadata
    metav1.ObjectMeta

    // If this object was created from some other resource, then this links
    // back to that resource. This field is used to find the in-tree representation
    // of the class parameters when the parameter reference of the class refers
    // to some unknown type.
    GeneratedFrom *ResourceClassParametersReference

    // Config defines configuration parameters that apply to each claim using this class
    // and all named resources in those claims. They are ignored while allocating the claim.
    // +optional
    Config *ConfigurationParameters

    // Filter describes additional contraints that must be met when using the class.
    // +optional
    Filter *ResourceFilterModel
}

// ResourceFilterModel must have one and only one field set.
type ResourceFilterModel struct {
    // NamedResources describes a resource filter using the named resources model.
    NamedResources *NamedResourcesFilter
}

// NamedResourcesFilter is used in ResourceFilterModel.
type NamedResourcesFilter struct {
    // DriverName excludes any named resource not provided by this driver.
    // +optional
    DriverName *string

    // Selector is a CEL expression which must evaluate to true if a
    // resource instance is suitable. The language is as defined in
    // https://kubernetes.io/docs/reference/using-api/cel/
    //
    // In addition, for each type in NamedResourcesAttributeValue there is a map that
    // resolves to the corresponding value of the instance under evaluation. Unknown
    // names cause a runtime error. Note that the CEL expression is applied to
    // *all* available resource instances by default, regardless of which driver provides it.
    // In that case. the CEL expression must first check that the instance has certain
    // attributes before using them.
    //
    // For example:
    //    attributes.quantity.has("a.dra.example.com") &&
    //    attributes.quantity["a.dra.example.com"].isGreaterThan(quantity("0")) &&
    //    # No separate check, b.dra.example.com is set whenever a.dra.example.com is,
    //    attributes.stringslice["b.dra.example.com"].isSorted()
    //
    // If a driver name is set, then such a check is not be needed if all instances
    // are known to have the attribute. Attributes names don't have to have
    // the driver name suffix.
    //
    // For example:
    //    attributes.quantity["a"].isGreaterThan(quantity("0")) &&
    //    attributes.stringslice["b"].isSorted()
    Selector string
}
```

###### ResourceClaim


```go
// ResourceClaim describes which resources are needed by a resource consumer.
// Its status tracks whether the resource has been allocated and what the
// resulting attributes are.
//
// This is an alpha type and requires enabling the DynamicResourceAllocation
// feature gate.
type ResourceClaim struct {
    metav1.TypeMeta
    // Standard object metadata
    // +optional
    metav1.ObjectMeta

    // Spec describes the desired attributes of a resource that then needs
    // to be allocated. It can only be set once when creating the
    // ResourceClaim.
    Spec ResourceClaimSpec

    // Status describes whether the resource is available and with which
    // attributes.
    // +optional
    Status ResourceClaimStatus
}

// Finalizer is the finalizer that gets set for claims
// which were allocated through a builtin controller.
const Finalizer = "dra.k8s.io/delete-protection"
```

The scheduler must set a finalizer in a ResourceClaim before it adds
an allocation. This ensures that an allocated, reserved claim cannot
be removed accidentally by a user.

If storing the status fails, the scheduler will retry on the next
scheduling attempt. If the ResourceClaim gets deleted in the meantime,
the scheduler will not try to schedule again. This situation is handled
by the kube-controller-manager by removing the finalizer.

Force-deleting a ResourceClaim by clearing its finalizers (something that users
should never do without being aware of the consequences) cannot be
prevented. Deleting the entire cluster also leaves resources allocated outside
of the cluster in an allocated state.

```go
// ResourceClaimSpec defines how a resource is to be allocated.
type ResourceClaimSpec struct {
    // ResourceClassName references the driver and additional parameters
    // via the name of a ResourceClass that was created as part of the
    // driver deployment.
    // +optional
    ResourceClassName string

    // ParametersRef references a separate object with arbitrary parameters
    // that will be used by the driver when allocating a resource for the
    // claim.
    //
    // The object must be in the same namespace as the ResourceClaim.
    // +optional
    ParametersRef *ResourceClaimParametersReference
}
```

The `ResourceClassName` field may be left empty. The parameters are sufficient
to determine which driver needs to provide resources. This leads to some corner cases:
- Empty `ResourceClassName` and nil `ParametersRef`: this is a claim which requests
  no resources. Such a claim can always be allocated with an empty result. Allowing
  this simplifies code generators which dynamically fill in the resource requests
  because they are allowed to generate an empty claim.
- Non-empty `ResourceClassName`, nil `ParametersRef`, nil
  `ResourceClass.DefaultClaimParametersRef`: this is handled the same way, the
  only difference is that the cluster admin has decided that such claims need
  no resources by not providing default parameters.

There is no default ResourceClass. If that is desirable, then it can be
implemented with a mutating and/or admission webhook.

```
// ResourceClaimStatus tracks whether the resource has been allocated and what
// the resulting attributes are.
type ResourceClaimStatus struct {
    // Allocation is set by the resource driver once a resource or set of
    // resources has been allocated successfully. If this is not specified, the
    // resources have not been allocated yet.
    // +optional
    Allocation *AllocationResult

    // ReservedFor indicates which entities are currently allowed to use
    // the claim. A Pod which references a ResourceClaim which is not
    // reserved for that Pod will not be started.
    //
    // There can be at most 32 such reservations. This may get increased in
    // the future, but not reduced.
    // +optional
    ReservedFor []ResourceClaimConsumerReference
}

// ReservedForMaxSize is the maximum number of entries in
// claim.status.reservedFor.
const ResourceClaimReservedForMaxSize = 32
```

##### ResourceClaimParameters

```go
// ResourceClaimParameters defines resource requests for a ResourceClaim in an
// in-tree format understood by Kubernetes.
type ResourceClaimParameters struct {
    metav1.TypeMeta
    // Standard object metadata
    metav1.ObjectMeta

    // If this object was created from some other resource, then this links
    // back to that resource. This field is used to find the in-tree representation
    // of the claim parameters when the parameter reference of the claim refers
    // to some unknown type.
    GeneratedFrom *ResourceClaimParametersReference

    // Shareable indicates whether the allocated claim is meant to be shareable
    // by multiple consumers at the same time.
    Shareable bool

    // Config defines configuration parameters that apply to the entire claim.
    // They are ignored while allocating the claim.
    Config *ConfigurationParameters

    ResourceRequestModel
}

// ResourceRequestModel must have one and only one field set.
type ResourceRequestModel struct {
    // NamedResources describes a request for resources with the named resources model.
    // The list may be empty, in which case the claim can be allocated without
    // using any instances.
    NamedResources *[]NamedResourcesRequest
}

// NamedResourcesRequest is used in ResourceRequestModel.
type NamedResourcesRequest struct {
    // Config defines configuration parameters that apply to the requested
    // instance. They are ignored while allocating the claim.
    Config *ConfigurationParameters

    // DriverName excludes any named resource not provided by this driver.
    // +optional
    DriverName *string

    // Selector is a CEL expression which must evaluate to true if a
    // resource instance is suitable. The language is as defined in
    // https://kubernetes.io/docs/reference/using-api/cel/
    //
    // In addition, for each type in NamedResourcesAttributeValue there is a map that
    // resolves to the corresponding value of the instance under evaluation. Unknown
    // names cause a runtime error. Note that the CEL expression is applied to
    // *all* available resource instances by default, regardless of which driver provides it.
    // In that case. the CEL expression must first check that the instance has certain
    // attributes before using them.
    //
    // For example:
    //    attributes.quantity.has("a.dra.example.com") &&
    //    attributes.quantity["a.dra.example.com"].isGreaterThan(quantity("0")) &&
    //    # No separate check, b.dra.example.com is set whenever a.dra.example.com is,
    //    attributes.stringslice["b.dra.example.com"].isSorted()
    //
    // If a driver name is set, then such a check is not be needed if all instances
    // are known to have the attribute. Attributes names don't have to have
    // the driver name suffix.
    //
    // For example:
    //    attributes.quantity["a"].isGreaterThan(quantity("0")) &&
    //    attributes.stringslice["b"].isSorted()
    Selector string
}
```

##### ConfigurationParameters

ConfigurationParameters is a one-of-many because in-tree configuration parameters might get
defined in the future.

```yaml
// ConfigurationParameters must have one and only one field set.
type ConfigurationParameters struct {
    Vendor []VendorConfigurationParameters
}

// VendorConfigurationParameters contains configuration parameters for a driver.
type VendorConfigurationParameters struct {
    // DriverName is used to determine which kubelet plugin needs
    // to be passed these configuration parameters.
    //
    // An admission webhook provided by the vendor could use this
    // to decide whether it needs to validate them.
    DriverName string

    // Parameters can contain arbitrary data. It is the
    // responsibility of the vendor to handle validation and
    // versioning.
    Parameters runtime.RawExtension
}
```

Strictly speaking, a pointer to a slice would be better because it allows
distinguishing between an unset list and an empty list. Unfortunately the gogo
protobuf code generator produces incorrect code for that. Treating the empty
list like "not set" is good enough here because if someone really wants empty
vendor parameters, they can simply not provide a ConfigurationParameters
instance.

##### Allocation result

```go
// AllocationResult contains attributes of an allocated resource.
type AllocationResult struct {
    // ResourceHandles contain the state associated with an allocation that
    // should be maintained throughout the lifetime of a claim. Each
    // ResourceHandle contains data that should be passed to a specific kubelet
    // plugin once it lands on a node.
    //
    // Setting this field is optional. It has a maximum size of 32 entries.
    // If null (or empty), it is assumed this allocation will be processed by a
    // single kubelet plugin with no ResourceHandle data attached. The name of
    // the kubelet plugin invoked will match the DriverName set in the
    // ResourceClaimStatus this AllocationResult is embedded in.
    //
    // +listType=atomic
    ResourceHandles []ResourceHandle

    // This field will get set by the resource driver after it has allocated
    // the resource to inform the scheduler where it can schedule Pods using
    // the ResourceClaim.
    //
    // Setting this field is optional. If null, the resource is available
    // everywhere.
    // +optional
    AvailableOnNodes *core.NodeSelector

    // Shareable determines whether the resource supports more
    // than one consumer at a time.
    // +optional
    Shareable bool
}

// AllocationResultResourceHandlesMaxSize represents the maximum number of
// entries in allocation.resourceHandles.
const AllocationResultResourceHandlesMaxSize = 32

// ResourceHandle holds opaque resource data for processing by a specific kubelet plugin.
type ResourceHandle struct {
    // DriverName specifies the name of the resource driver whose kubelet
    // plugin should be invoked to process this ResourceHandle's data once it
    // lands on a node. This may differ from the DriverName set in
    // ResourceClaimStatus this ResourceHandle is embedded in.
    DriverName string

    // StructuredData captures the result of the allocation for this
    // particular driver.
    StructuredData *StructuredResourceHandle
}

// StructuredResourceHandle is the in-tree representation of the allocation result.
type StructuredResourceHandle struct {
    // AdminConfig are the configuration parameters
    // from the resource class at the time that the claim was allocated.
    AdminConfig *DriverConfigurationParameters

    // UserConfig are the per-claim configuration parameters
    // from the resource claim parameters at the time that the claim was
    // allocated.
    UserConfig *DriverConfigurationParameters

    // NodeName is the name of the node providing the necessary resources
    // if the resources are local to a node.
    NodeName string

    // Results lists all allocated driver resources.
    Results []DriverAllocationResult
}

// DriverAllocationResult contains vendor parameters and the allocation result for
// one request.
type DriverAllocationResult struct {
    // UserConfig are the per-instance configuration parameters
    // from the resource claim parameters at the time that the claim was
    // allocated.
    UserConfig *DriverConfigurationParameters

    AllocationResultModel
}

// AllocationResultModel must have one and only one field set.
type AllocationResultModel struct {
    // NamedResources describes the allocation result when using the named resources model.
    NamedResources *NamedResourcesAllocationResult
}

// DriverConfigurationParameters must have one and only one field set.
//
// In contrast to VendorConfigurationParameters, the driver name is
// not included and has to be infered from the context.
type DriverConfigurationParameters struct {
	Vendor *runtime.RawExtension
}

```

##### ResourceClaimTemplate

```go
// ResourceClaimTemplate is used to produce ResourceClaim objects.
type ResourceClaimTemplate struct {
    metav1.TypeMeta
    // Standard object metadata
    // +optional
    metav1.ObjectMeta

    // Describes the ResourceClaim that is to be generated.
    //
    // This field is immutable. A ResourceClaim will get created by the
    // control plane for a Pod when needed and then not get updated
    // anymore.
    Spec ResourceClaimTemplateSpec
}

// ResourceClaimTemplateSpec contains the metadata and fields for a ResourceClaim.
type ResourceClaimTemplateSpec struct {
    // ObjectMeta may contain labels and annotations that will be copied into the PVC
    // when creating it. No other fields are allowed and will be rejected during
    // validation.
    // +optional
    metav1.ObjectMeta

    // Spec for the ResourceClaim. The entire content is copied unchanged
    // into the ResourceClaim that gets created from this template. The
    // same fields as in a ResourceClaim are also valid here.
    Spec ResourceClaimSpec
}
```

##### Object references

```go
// ResourceClassParametersReference contains enough information to let you
// locate the parameters for a ResourceClass.
type ResourceClassParametersReference struct {
    // APIGroup is the group for the resource being referenced. It is
    // empty for the core API. This matches the group in the APIVersion
    // that is used when creating the resources.
    // +optional
    APIGroup string
    // Kind is the type of resource being referenced. This is the same
    // value as in the parameter object's metadata.
    Kind string
    // Name is the name of resource being referenced.
    Name string
    // Namespace that contains the referenced resource. Must be empty
    // for cluster-scoped resources and non-empty for namespaced
    // resources.
    // +optional
    Namespace string
}

// ResourceClaimParametersReference contains enough information to let you
// locate the parameters for a ResourceClaim. The object must be in the same
// namespace as the ResourceClaim.
type ResourceClaimParametersReference struct {
    // APIGroup is the group for the resource being referenced. It is
    // empty for the core API. This matches the group in the APIVersion
    // that is used when creating the resources.
    // +optional
    APIGroup string
    // Kind is the type of resource being referenced. This is the same
    // value as in the parameter object's metadata, for example "ConfigMap".
    Kind string
    // Name is the name of resource being referenced.
    Name string
}

// ResourceClaimConsumerReference contains enough information to let you
// locate the consumer of a ResourceClaim. The user must be a resource in the same
// namespace as the ResourceClaim.
type ResourceClaimConsumerReference struct {
    // APIGroup is the group for the resource being referenced. It is
    // empty for the core API. This matches the group in the APIVersion
    // that is used when creating the resources.
    // +optional
    APIGroup string
    // Resource is the type of resource being referenced, for example "pods".
    Resource string
    // Name is the name of resource being referenced.
    Name string
    // UID identifies exactly one incarnation of the resource.
    UID types.UID
}
```

`ResourceClassParametersReference` and `ResourceClaimParametersReference` use
the more user-friendly "kind" to identify the object type because those
references are provided by users. `ResourceClaimConsumerReference` is typically
set by the control plane and therefore uses the more technically correct
"resource" name.

#### core

```go
type PodSpec {
   ...
    // ResourceClaims defines which ResourceClaims must be allocated
    // and reserved before the Pod is allowed to start. The resources
    // will be made available to those containers which consume them
    // by name.
    //
    // This is an alpha field and requires enabling the
    // DynamicResourceAllocation feature gate.
    //
    // This field is immutable.
    //
    // +featureGate=DynamicResourceAllocation
    // +optional
    ResourceClaims []PodResourceClaim
   ...
}

type  ResourceRequirements {
   Limits ResourceList
   Requests ResourceList
   ...
    // Claims lists the names of resources, defined in spec.resourceClaims,
    // that are used by this container.
    //
    // This is an alpha field and requires enabling the
    // DynamicResourceAllocation feature gate.
    //
    // This field is immutable.
    //
    // +featureGate=DynamicResourceAllocation
    // +optional
    Claims []ResourceClaim
}

// ResourceClaim references one entry in PodSpec.ResourceClaims.
type ResourceClaim struct {
    // Name must match the name of one entry in pod.spec.resourceClaims of
    // the Pod where this field is used. It makes that resource available
    // inside a container.
    Name string
}
```

`Claims` is a list of structs with a single `Name` element because that struct
can be extended later, for example to add parameters that influence how the
resource is made available to a container. This wouldn't be possible if
it was a list of strings.

```go
// PodResourceClaim references exactly one ResourceClaim through a ClaimSource.
// It adds a name to it that uniquely identifies the ResourceClaim inside the Pod.
// Containers that need access to the ResourceClaim reference it with this name.
type PodResourceClaim struct {
    // Name uniquely identifies this resource claim inside the pod.
    // This must be a DNS_LABEL.
    Name string

    // Source describes where to find the ResourceClaim.
    Source ClaimSource
}

// ClaimSource describes a reference to a ResourceClaim.
//
// Exactly one of these fields should be set.  Consumers of this type must
// treat an empty object as if it has an unknown value.
type ClaimSource struct {
    // ResourceClaimName is the name of a ResourceClaim object in the same
    // namespace as this pod.
    ResourceClaimName *string

    // ResourceClaimTemplateName is the name of a ResourceClaimTemplate
    // object in the same namespace as this pod.
    //
    // The template will be used to create a new ResourceClaim, which will
    // be bound to this pod. When this pod is deleted, the ResourceClaim
    // will also be deleted. The pod name and resource name, along with a
    // generated component, will be used to form a unique name for the
    // ResourceClaim, which will be recorded in pod.status.resourceClaimStatuses.
    //
    // This field is immutable and no changes will be made to the
    // corresponding ResourceClaim by the control plane after creating the
    // ResourceClaim.
    ResourceClaimTemplateName *string
}

struct PodStatus {
    ...
    // Status of resource claims.
    // +featureGate=DynamicResourceAllocation
    // +optional
    ResourceClaimStatuses []PodResourceClaimStatus
}

// PodResourceClaimStatus is stored in the PodStatus for each PodResourceClaim
// which references a ResourceClaimTemplate. It stores the generated name for
// the corresponding ResourceClaim.
type PodResourceClaimStatus struct {
    // Name uniquely identifies this resource claim inside the pod.
    // This must match the name of an entry in pod.spec.resourceClaims,
    // which implies that the string must be a DNS_LABEL.
    Name string

    // ResourceClaimName is the name of the ResourceClaim that was
    // generated for the Pod in the namespace of the Pod. If this is
    // unset, then generating a ResourceClaim was not necessary. The
    // pod.spec.resourceClaims entry can be ignored in this case.
    ResourceClaimName *string
}
```

### kube-controller-manager

The code that creates a ResourceClaim from a ResourceClaimTemplate started
as an almost verbatim copy of the [generic ephemeral volume
code](https://github.com/kubernetes/kubernetes/tree/master/pkg/controller/volume/ephemeral),
just with different types. Later, generating the name of the ephemeral ResourceClaim
was added.

kube-controller-manager needs [RBAC
permissions](https://github.com/kubernetes/kubernetes/commit/ff3e5e06a79bc69ad3d7ccedd277542b6712514b#diff-2ad93af2302076e0bdb5c7a4ebe68dd3188eee8959c72832181a7597417cd196) that allow creating and updating ResourceClaims.

kube-controller-manager also removes `claim.status.reservedFor` entries that reference
removed pods or pods that have completed ("Phase" is "done" or will never start).
This is required for pods because kubelet does not have write
permission for ResourceClaimStatus. Pods as user is the common case, so special
code based on a shared pod informer will handle it. Other consumers
need to be handled by whatever controller added them.

In addition to updating `claim.status.reservedFor`, kube-controller-manager also
removes the allocation from ResourceClaims that are no longer in use.
Updating the claim during deallocation will be observed by kube-scheduler and
tells it that it can use the capacity set aside for the claim
again. kube-controller-manager itself doesn't need to support specific structured
models.

### kube-scheduler

The scheduler plugin for ResourceClaims ("claim plugin" in this section)
needs to implement several extension points. It is responsible for
ensuring that a ResourceClaim is allocated and reserved for a Pod before
the final binding of a Pod to a node.

The following extension points are implemented in the new claim plugin. Except
for some unlikely edge cases (see below) there are no API calls during the main
scheduling cycle. Instead, the plugin collects information and updates the
cluster in the separate goroutine which invokes PreBind.


#### EventsToRegister

This registers all cluster events that might make an unschedulable pod
schedulable, like creating a claim that the pod needs or finishing the
allocation of a claim.

[Queuing hints](https://github.com/kubernetes/enhancements/issues/4247) are
supported. These are callbacks that can limit the effect of a cluster event to
specific pods. For example, allocating a claim only makes those pods
scheduleable which reference the claim. There is no need to try scheduling a pod
which waits for some other claim. Hints are also used to trigger the next
scheduling cycle for a pod immediately when some expected and require event
like "drivers have provided information" occurs, instead of forcing the pod to
go through the backoff queue and the usually 5 second long delay associated
with that.

Queuing hints are an optional feature of the scheduler, with (as of Kubernetes
1.29) their own `SchedulerQueueingHints` feature gate that defaults to
off. When turned off, performance of scheduling pods with resource claims is
slower compared to a cluster configuration where they are turned on.

#### PreEnqueue

This checks whether all claims referenced by a pod exist. If they don't,
scheduling the pod has to wait until the kube-controller-manager or user create
the claims. PreEnqueue tries to finish quickly because it is called from
event handlers, so not everything is checked.

#### Pre-filter

This is a more thorough version of the checks done by PreEnqueue. It ensures
that all information that is needed (ResourceClaim, ResourceClass, parameters)
is available.

Another reason why a Pod might not be schedulable is when it depends on claims
which are in the process of being allocated. That process starts in Reserve and
ends in PreBind or Unreserve (see below).

It then prepares for filtering by converting information stored in various
places (node filter in ResourceClass, available resources in ResourceSlices,
allocated resources in ResourceClaim statuses, in-flight allocations) into a
format that can be used efficiently by Filter.

#### Filter

This checks whether the given node has access to those ResourceClaims which
were already allocated. For ResourceClaims that were not, it checks that the
allocation can succeed for a node.

#### Post-filter

This is called when no suitable node could be found. If the Pod depends on ResourceClaims with delayed
allocation, then deallocating one or more of these ResourceClaims may make the
Pod schedulable after allocating the resource elsewhere. Therefore each
ResourceClaim with delayed allocation is checked whether all of the following
conditions apply:
- allocated
- not currently in use
- it was the reason why some node could not fit the Pod, as recorded earlier in
  Filter

One of the ResourceClaims satisfying these criteria is picked randomly and gets
deallocated by clearing the allocation in its status. This may make it possible to run the Pod
elsewhere. If it still doesn't help, deallocation may continue with another
ResourceClaim, if there is one.

This is currently using blocking API calls. It's quite rare because this
situation can only arise when there are multiple claims per pod and writing
the status of one of them fails, thus leaving the other claims in the
allocated state.

#### Reserve

A node has been chosen for the Pod.

For each unallocated claim, the actual allocation result is determined now. To
avoid blocking API calls, that result is not written to the status yet. Instead,
it gets stored in a map of in-flight claims.

#### PreBind

This is called in a separate goroutine. The plugin now checks all the
information gathered earlier and updates the cluster accordingly. If some
some API request fails now, PreBind fails and the pod must be
retried.

Claims whose status got written back get removed from the in-flight claim map.

#### Unreserve

The claim plugin removes the Pod from the `claim.status.reservedFor` field if
set there because it cannot be scheduled after all.

This is necessary to prevent a deadlock: suppose there are two stand-alone
claims that only can be used by one pod at a time and two pods which both
reference them. Both pods will get scheduled independently, perhaps even by
different schedulers. When each pod manages to allocate and reserve one claim,
then neither of them can get scheduled because they cannot reserve the other
claim.

Giving up the reservations in Unreserve means that the next pod scheduling
attempts have a chance to succeed. It's non-deterministic which pod will win,
but eventually one of them will. Not giving up the reservations would lead to a
permanent deadlock that somehow would have to be detected and resolved to make
progress.

All claims get removed from the in-flight claim map.

Unreserve is called in two scenarios:
- In the main goroutine when scheduling a pod has failed: in that case the plugin's
  Reserve call hasn't actually changed the claim status yet, so there is nothing
  that needs to be rolled back.
- After binding has failed: this runs in a goroutine, so reverting the
  `claim.status.reservedFor` with a blocking call is acceptable.

### kubelet

#### Managing resources

kubelet must ensure that resources are ready for use on the node before running
the first Pod that uses a specific resource instance and make the resource
available elsewhere again when the last Pod has terminated that uses it. For
both operations, kubelet calls a resource kubelet plugin as explained in the next
section.

Pods that are not listed in ReservedFor or where the ResourceClaim doesn't
exist at all must not be allowed to run. Instead, a suitable event must be
emitted which explains the problem. Such a situation can occur as part of
downgrade scenarios.

If this was the last Pod on the node that uses the specific
resource instance, then NodeUnprepareResource (see below) must have been called
successfully before allowing the pod to be deleted. This ensures that network-attached resource are available again
for other Pods, including those that might get scheduled to other nodes. It
also signals that it is safe to deallocate and delete the ResourceClaim.


![kubelet](./kubelet.png)

#### Communication between kubelet and resource kubelet plugin

Resource kubelet plugins are discovered through the [kubelet plugin registration
mechanism](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/device-plugins/#device-plugin-registration). A
new "ResourcePlugin" type will be used in the Type field of the
[PluginInfo](https://pkg.go.dev/k8s.io/kubelet/pkg/apis/pluginregistration/v1#PluginInfo)
response to distinguish the plugin from device and CSI plugins.

Under the advertised Unix Domain socket the kubelet plugin provides the
k8s.io/kubelet/pkg/apis/dra gRPC interface. It was inspired by
[CSI](https://github.com/container-storage-interface/spec/blob/master/spec.md),
with “volume” replaced by “resource” and volume specific parts removed.

##### NodeListAndWatchResources

NodeListAndWatchResources returns a stream of NodeResourcesResponse objects.
At the start and whenever resource availability changes, the
plugin must send one such object with all information to the kubelet. The
kubelet then syncs that information with ResourceSlice objects.

```
message NodeListAndWatchResourcesRequest {
}

message NodeListAndWatchResourcesResponse {
    repeated k8s.io.api.resource.v1alpha2.ResourceModel resources = 1;
}
```

##### NodePrepareResource

This RPC is called by the kubelet when a Pod that wants to use the specified
resource is scheduled on a node. The Plugin SHALL assume that this RPC will be
executed on the node where the resource will be used.

ResourceClaim.meta.Namespace, ResourceClaim.meta.UID, ResourceClaim.Name and
one of the ResourceHandles from the ResourceClaimStatus.AllocationResult with
a matching DriverName should be passed to the Plugin as parameters to identify
the claim and perform resource preparation.

ResourceClaim parameters (namespace, UUID, name) are useful for debugging.
They enable the Plugin to retrieve the full ResourceClaim object, should it
ever be needed (normally it shouldn't).

The Plugin SHALL return fully qualified device name[s].

The Plugin SHALL ensure that there are json file[s] in CDI format
for the allocated resource. These files SHALL be used by runtime to
update runtime configuration before creating containers that use the
resource.

This operation SHALL do as little work as possible as it’s called
after a pod is scheduled to a node. All potentially failing operations
SHALL be done during allocation phase.

This operation MUST be idempotent. If the resource corresponding to
the `resource_id` has already been prepared, the Plugin MUST reply `0
OK`.

If this RPC failed, or kubelet does not know if it failed or not, it
MAY choose to call `NodePrepareResource` again, or choose to call
`NodeUnprepareResource`.

On a successful call this RPC should return set of fully qualified
CDI device names, which kubelet MUST pass to the runtime through the CRI
protocol. For version v1alpha3, the RPC should return multiple sets of
fully qualified CDI device names, one per claim that was sent in the input parameters.

```protobuf
message NodePrepareResourcesRequest {
     // The list of ResourceClaims that are to be prepared.
     repeated Claim claims = 1;
}

message Claim {
    // The ResourceClaim namespace (ResourceClaim.meta.Namespace).
    // This field is REQUIRED.
    string namespace = 1;
    // The UID of the Resource claim (ResourceClaim.meta.UUID).
    // This field is REQUIRED.
    string uid = 2;
    // The name of the Resource claim (ResourceClaim.meta.Name)
    // This field is REQUIRED.
    string name = 3;
    // Resource handle (AllocationResult.ResourceHandles[*].Data)
    // This field is OPTIONAL.
    string resource_handle = 4;
    // Structured parameter resource handle (AllocationResult.ResourceHandles[*].StructuredData).
    // This field is OPTIONAL. If present, it needs to be used
    // instead of resource_handle. It will only have a single entry.
    //
    // Using "repeated" instead of "optional" is a workaround for https://github.com/gogo/protobuf/issues/713.
    repeated k8s.io.api.resource.v1alpha2.StructuredResourceHandle structured_resource_handle = 5;
}
```

`resource_handle` and `structured_resource_handle` will be set depending on how
the claim was allocated. See also KEP #3063.

```
message NodePrepareResourcesResponse {
    // The ResourceClaims for which preparation was done
    // or attempted, with claim_uid as key.
    //
    // It is an error if some claim listed in NodePrepareResourcesRequest
    // does not get prepared. NodePrepareResources
    // will be called again for those that are missing.
    map<string, NodePrepareResourceResponse> claims = 1;
}
```

CRI protocol MUST be extended for this purpose:

 * CDIDevice structure should be added to the CRI specification
```protobuf
// CDIDevice specifies a CDI device information.
message CDIDevice {
    // Fully qualified CDI device name
    // for example: vendor.com/gpu=gpudevice1
    // see more details in the CDI specification:
    // https://github.com/container-orchestrated-devices/container-device-interface/blob/main/SPEC.md
    string name = 1;
}
```
 * CDI devices should be added to the ContainerConfig structure:
```protobuf
// ContainerConfig holds all the required and optional fields for creating a
// container.
message ContainerConfig {
    // Metadata of the container. This information will uniquely identify the
    // container, and the runtime should leverage this to ensure correct
    // operation. The runtime may also use this information to improve UX, such
    // as by constructing a readable name.
    ContainerMetadata metadata = 1 ;
    // Image to use.
    ImageSpec image = 2;
    // Command to execute (i.e., entrypoint for docker)
    repeated string command = 3;
...
    // CDI devices for the container.
    repeated CDIDevice cdi_devices = 17;
}
```

###### NodePrepareResource Errors

If the plugin is unable to complete the NodePrepareResource call
successfully, it MUST return a non-ok gRPC code in the gRPC status.
If the conditions defined below are encountered, the plugin MUST
return the specified gRPC error code.  Kubelet MUST implement the
specified error recovery behavior when it encounters the gRPC error
code.

| Condition | gRPC Code | Description | Recovery Behavior |
|-----------|-----------|-------------|-------------------|
| Resource does not exist | 5 NOT_FOUND | Indicates that a resource corresponding to the specified `resource_id` does not exist. | Caller MUST verify that the `resource_id` is correct and that the resource is accessible and has not been deleted before retrying with exponential back off. |


##### NodeUnprepareResources

A Kubelet Plugin MUST implement this RPC call. This RPC is a reverse
operation of `NodePrepareResource`. This RPC MUST undo the work by
the corresponding `NodePrepareResource`. This RPC SHALL be called by
kubelet at least once for each successful `NodePrepareResource`. The
Plugin SHALL assume that this RPC will be executed on the node where
the resource is being used.

This RPC is called by the kubelet when the last Pod using the resource is being
deleted or has reached a final state ("Phase" is "done").

This operation MUST be idempotent. If this RPC failed, or kubelet does
not know if it failed or not, it can choose to call
`NodeUnprepareResource` again.

```protobuf
message NodeUnprepareResourcesRequest {
    // The list of ResourceClaims that are to be unprepared.
    repeated Claim claims = 1;
}

message NodeUnprepareResourcesResponse {
    // The ResourceClaims for which preparation was reverted.
    // The same rules as for NodePrepareResourcesResponse.claims
    // apply.
    map<string, NodeUnprepareResourceResponse> claims = 1;
}

message NodeUnprepareResourceResponse {
    // If non-empty, unpreparing the ResourceClaim failed.
    string error = 1;
}
```

###### NodeUnprepareResource Errors

If the plugin is unable to complete the NodeUprepareResource call
successfully, it MUST return a non-ok gRPC code in the gRPC status.
If the conditions defined below are encountered, the plugin MUST
return the specified gRPC error code.  Kubelet MUST implement the
specified error recovery behavior when it encounters the gRPC error
code.

| Condition | gRPC Code | Description | Recovery Behavior |
|-----------|-----------|-------------|-------------------|
| Resource does not exist | 5 NOT_FOUND | Indicates that a resource corresponding to the specified `resource_id` does not exist. | Caller MUST verify that the `resource_id` is correct and that the resource is accessible and has not been deleted before retrying with exponential back off. |


### Simulation with CA

The usual call sequence of a scheduler plugin when used in the scheduler is
at program startup:
- instantiate plugin
- EventsToRegister

For each new pod:
- PreEnqueue

For each pod that is ready to be scheduled, one pod at a time:
- PreFilter, Filter, etc.

Scheduling a pod gets finalized with:
- Reserve, PreBind, Bind

CA works a bit differently. It identifies all pending pods,
takes a snapshot of the current cluster state, and then simulates the effect
of scheduling those pods with additional nodes added to the cluster. To
determine whether a pod fits into one of these simulated nodes, it
uses the same PreFilter and Filter plugins as the scheduler. Other extension
points (Reserve, Bind) are not used. Plugins which modify the cluster state
therefore need a different way of recording the result of scheduling
a pod onto a node.

One option for this is to add a new optional plugin interface that is
implemented by the dynamic resource plugin. Through that interface the
autoscaler can then inform the plugin about events like starting simulation,
binding pods, and adding new nodes. With this approach, the autoscaler doesn't
need to know what the persistent state of each plugin is.

Another option is to extend the state that the autoscaler keeps for
plugins. The plugin then shouldn't need to know that it runs inside the
autoscaler. This implies that the autoscaler will have to call Reserve and
PreBind as that is where the state gets updated.

Which of these options is chosen will be decided during the implementation
phase. Autoscalers which don't use the in-tree scheduler plugin will have
to implement a similar logic.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None.

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

- `k8s.io/kubernetes/pkg/scheduler`: 2022-05-24 - 75.0%
- `k8s.io/kubernetes/pkg/scheduler/framework`: 2022-05-24 - 76.3%
- `k8s.io/kubernetes/pkg/controller`: 2022-05-24 - 69.4%
- `k8s.io/kubernetes/pkg/kubelet`: 2022-05-24 - 64.5%

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

End-to-end testing depends on a working resource driver and a container runtime
with CDI support. A [test driver](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/dra/test-driver)
was developed in parallel to developing the
code in Kubernetes.

That test driver simply takes parameters from ResourceClass
and ResourceClaim and turns them into environment variables that then get
checked inside containers. Tests for different behavior of an driver in various
scenarios can be simulated by running the control-plane part of it in the E2E
test itself. For interaction with kubelet, proxying of the gRPC interface can
be used, as in the
[csi-driver-host-path](https://github.com/kubernetes-csi/csi-driver-host-path/blob/16251932ab81ad94c9ec585867104400bf4f02e5/cmd/hostpathplugin/main.go#L61-L63):
then the kubelet plugin runs on the node(s), but the actual processing of gRPC
calls happens inside the E2E test.

All tests that don't involve actually running a Pod can become part of
conformance testing. Those tests that run Pods cannot be because CDI support in
runtimes is not required.

For beta:
- pre-merge with kind (optional, triggered for code which has an impact on DRA): https://testgrid.k8s.io/sig-node-dynamic-resource-allocation#pull-kind-dra
- periodic with kind: https://testgrid.k8s.io/sig-node-dynamic-resource-allocation#ci-kind-dra
- pre-merge with CRI-O: https://testgrid.k8s.io/sig-node-dynamic-resource-allocation#pull-node-dra
- periodic with CRI-O: https://testgrid.k8s.io/sig-node-dynamic-resource-allocation#ci-node-e2e-crio-dra


### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Fully implemented
- Additional tests are in Testgrid and linked in KEP

#### GA

- 3 examples of real-world usage
- Allowing time for feedback

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

### Upgrade / Downgrade Strategy

Because of the strongly-typed versioning of resource attributes and allocation
results, the gRPC interface between kubelet and the DRA driver is tied to the
version of the supported structured models. A DRA driver has to implement all
gRPC interfaces that might be used by older releases of kubelet. The same
applies when upgrading kubelet while the DRA driver remains at an older
version.

### Version Skew Strategy

Ideally, the latest release of a DRA driver should be used and it should
support a wide range of structured type versions. Then problems due to version
skew are less likely to occur.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate
  - Feature gate name: DynamicResourceAllocation
  - Components depending on the feature gate:
    - kube-apiserver
    - kubelet
    - kube-scheduler
    - kube-controller-manager

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Applications that were already deployed and are running will continue to
work, but they will stop working when containers get restarted because those
restarted containers won't have the additional resources.

###### What happens if we reenable the feature if it was previously rolled back?

Pods might have been scheduled without handling resources. Those Pods must be
deleted to ensure that the re-created Pods will get scheduled properly.

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

Tests for apiserver will cover disabling the feature. This primarily matters
for the extended PodSpec: the new fields must be preserved during updates even
when the feature is disabled.

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

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

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

Metrics in kube-scheduler (names to be decided):
- number of classes using structured parameters
- number of claims which currently are allocated with structured parameters

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [X] API .status
  - Other field: ".status.allocation" will be set for a claim using structured parameters
    when needed by a pod.

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

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

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

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

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

- Kubernetes 1.30: Code merged as "alpha"

## Drawbacks

DRA driver developers have to give up some flexibility with regards to
parameters compared to opaque parameters in KEP #3063.
They have to learn and understand how structured models
work to pick something which fits their needs.

## Alternatives

### Publishing resource information in node status

This is not desirable for several reasons (most important one first):
- All data from all drivers must be in a single object which is already
  large. It might become too large, with no chance of mitigating that by
  splitting up the information.
- All watchers of node objects get the information even if they don't need it.
- It puts complex alpha quality fields into a GA API.

### Injecting vendor logic into CA

With this KEP, vendor's use resource tracking and simulation that gets
implemented in core Kubernetes. Alternatively, CA could support vendor logic in
several different ways:

- Call out to a vendor server via some RPC mechanism (similar to scheduler
  webhooks). The risk here is that simulation becomes to slow. Configuration
  and security would be more complex.

- Load code provided by a vendor as [Web Assembly
  (WASM)](https://webassembly.org/) at runtime and invoke it similar to the
  builtin controllers in this KEP.  WASM is currently too experimental and has
  several drawbacks (single-threaded, all data must be
  serialized). https://github.com/kubernetes-sigs/kube-scheduler-wasm-extension
  is currently exploring usage of WASM for writing scheduler plugins. If this
  becomes feasible, then implementing a builtin controller which delegates its
  logic to vendor WASM code will be possible.

- Require that vendors provide Go code with their custom logic and rebuild CA
  with that code included. The scheduler could continue to use
  PodSchedulingContext, as long as the custom logic exactly matches what the
  DRA driver controller does. This approach is not an option when a pre-built
  CA binary has to be used and leads to challenges around maintenance and
  support of such a rebuilt CA binary. However, technically it [becomes
  possible](https://github.com/kubernetes-sigs/kube-scheduler-wasm-extension)
  with this KEP.

### ResourceClaimTemplate

Instead of creating a ResourceClaim from a template, the
PodStatus could be extended to hold the same information as a
ResourceClaimStatus. Every component which works with that information
then needs permission and extra code to work with PodStatus. Creating
an extra object seems simpler.

### Reusing volume support as-is

ResourceClaims are similar to PersistentVolumeClaims and also a lot of
the associated logic is similar. An [early
prototype](https://github.com/intel/proof-of-concept-cdi) used a
custom CSI driver to manage resources.

The user experience with that approach is poor because per-resource
parameters must be stored in annotations of a PVC due to the lack of
custom per-PVC parameters. Passing annotations as additional parameters was [proposed
before](https://github.com/kubernetes-csi/external-provisioner/issues/86)
but were essentially [rejected by
SIG-Storage](https://github.com/kubernetes-csi/external-provisioner/issues/86#issuecomment-465836185)
because allowing apps to set custom parameters would make apps
non-portable.

The current volume support also has open issues that affect the
“volume as resource” approach: Multiple different Pods on a node are
allowed to use the same
volume. https://github.com/kubernetes/enhancements/pull/2489 will
address that, but is still work in progress.  Recovery from a bad node
selection during delayed binding may get stuck when a Pod has multiple
volumes because volumes are not getting deleted after a partial
provisioning. A proposal to fix that needs further work
(https://github.com/kubernetes/enhancements/pull/1703).  Each “fake”
CSI driver would have to implement and install a scheduler extender
because storage capacity tracking only considers volume size as
criteria for selecting nodes, which is not applicable for custom
resources.

### Extend volume support

The StorageClass and PersistentVolumeClaim structs could be extended
to allow custom parameters. Together with an extension of the CSI
standard that would address the main objection against the previous
alternative.

However, SIG-Storage and the CSI community would have to agree to this
kind of reuse and accept that some of the code maintained by them
becomes more complex because of these new use cases.

### Extend Device Plugins

The device plugins API could be extended to implement some of the
requirements mentioned in the “Motivation” section of this
document. There were certain attempts to do it, for example an attempt
to [add ‘Deallocate’ API call](https://github.com/kubernetes/enhancements/pull/1949) and [pass pod annotations to 'Allocate' API call](https://github.com/kubernetes/kubernetes/pull/61775)

However, most of the requirements couldn’t be satisfied using this
approach as they would require major incompatible changes in the
Device Plugins API. For example: partial and optional resource
allocation couldn’t be done without changing the way resources are
currently declared on the Pod and Device Plugin level.

Extending the device plugins API to use [Container Device Interface](https://github.com/container-orchestrated-devices/container-device-interface)
would help address some of the requirements, but not all of them.

NodePrepareResource and NodeUnprepareResource could be added to the Device Plugins API and only get called for
resource claims.

However, this would mean that
developers of the device plugins would have to implement mandatory
API calls (ListAndWatch, Allocate), which could create confusion
as those calls are meaningless for the Dynamic Resource Allocation
purposes.

Even worse, existing device plugins would have to implement the new
calls with stubs that return errors because the generated Go interface
will require them.

It should be also taken into account that device plugins API is
beta. Introducing incompatible changes to it may not be accepted by
the Kubernetes community.

### Webhooks instead of ResourceClaim updates

In the current design, scheduler and the third-party resource driver communicate by
updating fields in a ResourceClaim. This has several advantages compared to an
approach were kube-scheduler retrieves information from the resource driver
via HTTP:
* No need for a new webhook API.
* Simpler deployment of a resource driver because all it needs are
  credentials to communicate with the apiserver.
* Current status can be checked by querying the ResourceClaim.

The downside is higher load on the apiserver and an increase of the size of
ResourceClaim objects.

### ResourceDriver

Similar to CSIDriver for storage, a separate object describing a resource
driver might be useful at some point. At the moment it is not needed yet and
therefore not part of the v1alpha2 API. If it becomes necessary to describe
optional features of a resource driver, such a ResourceDriver type might look
like this:

```
type ResourceDriver struct {
    // The name of the object is the unique driver name.
    ObjectMeta

    // Features contains a list of features supported by the driver.
    // New features may be added over time and must be ignored
    // by code that does not know about them.
    Features []ResourceDriverFeature
}

type ResourceDriverFeature struct {
    // Name is one of the pre-defined names for a feature.
    Name ResourceDriverFeatureName
    // Parameters might provide additional information about how
    // the driver supports the feature. Boolean features have
    // no parameters, merely listing them indicates support.
    Parameters runtime.RawExtension
}
```

### Complex sharing of ResourceClaim

At the moment, the allocation result marks as a claim as either "shareable" by
an unlimited number of consumers or "not shareable". More complex scenarios
might be useful like "may be shared by a certain number of consumers", but so
far such use cases have not come up yet. If they do, the `AllocationResult` can
be extended with new fields as defined by a follow-up KEP.

## Infrastructure Needed

Initially, all development will happen inside the main Kubernetes
repository. The mock driver can be developed inside test/e2e/dra.  For the
generic part of that driver, i.e. the code that other drivers can reuse, and
other common code a new staging repo `k8s.io/dynamic-resource-allocation` is
needed.
