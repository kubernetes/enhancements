# [KEP-5004](https://github.com/kubernetes/enhancements/issues/5004): DRA: Handle extended resource requests via DRA Driver

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Resource Slice API](#resource-slice-api)
  - [Resource Claim API](#resource-claim-api)
  - [Scheduling for Extended Resource](#scheduling-for-extended-resource)
  - [Scheduling for Resource Claim](#scheduling-for-resource-claim)
  - [Actuation for Extended Resource](#actuation-for-extended-resource)
  - [Actuation for Resource Claim](#actuation-for-resource-claim)
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
<!-- /toc -->

## Release Signoff Checklist

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

Extended resource provides a simple, concise approach to describe resource
capacity, and resource consumption. In constrast, Dynamic Resource
Allocation (DRA) provides more expressive, flexible, powerful, yet more
complicated, and harder to use approach.

This KEP provides a solution to enable cluster administrators and device drivers
to advertise the resources as extended resource, and enables the application
developers, and operators to continue using extended resource to request for such
resources.

This KEP provides dynamic allocation of resources to requests made through
extended resource, or DRA resource claim.

## Motivation

There are three major motivations for the solution in this KEP.

* Enable the existing applications to run without modification.

* Enable application developers, operators to transition to DRA gradually at
  their own pace.

* Enable cluster administrators to transition to DRA gradually at their own pace.

For example, the following `Deployment` can be installed without modification on a
cluster with DRA `ResourceSlice` and `Node` below. The 1 GPU out of the 8 GPUs on
the node is dynamically allocated to the pod, with the remaining 7 GPUs left for
allocation for future requests from either extended resource, or DRA resource claim.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: demo
  template:
    metadata:
      labels:
        app: demo
    spec:
      containers:
      - name: demo
        image: nvidia/cuda:8.0-runtime
        command: ["/bin/sh", "-c"]
        args: ["nvidia-smi && tail -f /dev/null"]
        resources:
          limits:
            nvidia.com/gpu: 1
```

```yaml
apiVersion: resource.k8s.io/v1beta1
kind: ResourceSlice
metadata:
  name: gke-drabeta-n1-standard-4-2xt4-346fe653-zrw2-gpu.nvidia.coqj92d
spec:
  devices:
  - basic:
    name: gpu-0
  - basic:
    name: gpu-1
  - basic:
    name: gpu-2
  - basic:
    name: gpu-3
  - basic:
    name: gpu-4
  - basic:
    name: gpu-5
  - basic:
    name: gpu-6
  - basic:
    name: gpu-7
  driver: gpu.nvidia.com
  nodeName: gke-drabeta-n1-standard-4-2xt4-346fe653-zrw2
  extendedResourceName: nvidia.com/gpu
```

```yaml
apiVersion: v1
kind: Node
metadata:
  name: gke-drabeta-n1-standard-4-2xt4-346fe653-zrw2
status:
  allocatable:
    cpu: 3920m
    ephemeral-storage: "49455357798"
    hugepages-1Gi: "0"
    hugepages-2Mi: "0"
    memory: 12636272Ki
    pods: "110"
    nvidia.com/gpu: "8"
  capacity:
    cpu: "4"
    ephemeral-storage: 101430960Ki
    hugepages-1Gi: "0"
    hugepages-2Mi: "0"
    memory: 15335536Ki
    pods: "110"
    nvidia.com/gpu: "8"
```

With this motivating example in mind. We define the following goals and
non-goals of this KEP.

### Goals

* Introduce the ability for DRA to advertise extended resources listed in a
  ResourceSlice, and for the scheduler to consider them for allocation.

* Enable application operators to use the existing extended resource limits to
request for DRA resources.

* Extended resource support is not added just for easing the transition to DRA
  for the short term. Its ease of use is one big advantage to keep it remaining
  useful for the long term.

### Non-Goals

* Simplify DRA driver developement. The DRA driver needs to support both DRA API
  and extended resource API. This KEP adds complexity and cost to the driver.

## Proposal

The basic idea is the following:

1. Introduce a field `ExtendedResourceName` to `ResourceSlice` and `Device` to
   allow cluster administrators to configure DRA device driver to advertise
   certain devices as extended resource.
1. Introduce a special `ResourceClaim` to keep track of device allocations for
   each extended resource on each node.
1. kube-scheduler uses DRA scheduling algorithm to fit pod's extended
   resource request to a node that advertises the extended resource in DRA
   `ResorceSlice`.
1. DRA driver speaks both device plugin proto for extended resource actuation
   and DRA proto for DRA resource claim actuation.

With these additions in place, the resources can be consumed by extended resouce
requests, or by DRA resouce claims. The scheduler has everything it needs to support
the dynamic allocation of devices to requests made through extended resource and
resource claims. No static partition of resources between extended resources and
resource claims is needed. The kubelet and DRA driver has everything they need
to admit and pass the allocated devices to the pod to run.

## Design Details

### Resource Slice API
The exact set of proposed API changes on Resource Slice can be seen below:
```go
// ResourceSliceSpec contains the information published by the driver in one ResourceSlice.
type ResourceSliceSpec struct {
	...

	// The extended resource name for all the devices in the ResourceSlice
	// advertised as
	//
	// +optional
	ExtendedResourceName string
}

// Device represents one individual hardware instance that can be selected based
// on its attributes. Besides the name, exactly one field must be set.
// +k8s:deepcopy-gen=true
type Device struct {
	// Name is unique identifier among all devices managed by
	// the driver in the pool. It must be a DNS label.
	//
	// +required
	Name string `json:"name"`
	...

	// ExtendedResourceName is the extended resource name
	// the device is advertised as. It must be a DNS label.
	// It overrides the ExtendedResourceName at ResourceSlice if both are
	// present.
	//
	// +optional
	ExtendedResourceName string
}
```

The devices can be advertised with an extended resource name. The extended
resource name can be specified on each individual device. Different
devices can be advertised as different extended resource name, or not
advertised as extended resource at all.

Alternatively, the extended resource name can be specified at the
`ResourceSlice` level, then all the devices in the resource slice are
advertised as the given extended resource name. If a device has a different
extended resource name than that given in the `ResoureSlice`, the device's
extended resource name is used for that device.

### Resource Claim API
There is no API change on `Resource Claim`, i.e. no new API type. However, a special
resource claim object is created to keep track of device allocations for extended
resource. The special resource claim object has following properties:

  * It is namespace scoped, like other resource claim objects.
  * It is a singleton per node. There is at most one such resource claim object for
    a given extended resource in a given namespace per node that advertises the
    extended resource in `ResourceSlice`.
  * It is not owned by a pod, this is different from regular resource claim.
  * it is owned by node instead.
  * Its field `status.allocation.devices` is used, other fields are ununsed,
    such as `status.allocation.reservedFor` and `spec`.
  * It has annotation `resource.kubernetes.io/extended-resource-name:`, and it
    does not have annotation `resource.kubernetes.io/pod-claim-name:`

```yaml
metadata:
  annotations:
    resource.kubernetes.io/extended-resource-name: foo.domain/bar
```

The special resource claim object lifecycle is managed by the resource claim
controller.

  * It is *created* in a namespace when there is a pod with extended resource
  requests, and the extended resource is advertised by `Resource Slice`. For
  example, A resource slice has `ExtendedResourceName: foo-domain/bar`, and a
  pod requests for foo-domain/bar, then a speical resource claim object is
  created in that same namespace as the pod.
  * It is *deleted* with the owning node deletion, as it is owned by a node.
  * It is *deleted* by the resource claim controller when there
  is no pod in the namespace requesting for the extended resource.
  * It is *updated* by the scheduler for scheduling extended resource requests.
  * It is *read* by the kubelet DRA device plugin driver during actuating the
  pod on the node.
  * It is *ignored* by the scheduler DRA plugin as it has no spec, hence
  scheduler does not need to do any allocation for it.
  * It is *read* by scheduler DRA plugin for the devices allocated,  so that
  the scheduler remove considerations for allocation of these devices for
  other DRA resource claim requests.
  * It is *ignored* by the kubelet DRA device driver as it has no reservedFor.

### Scheduling for Extended Resource
When a pod spec uses extended resources to request devices, the scheduler needs
to call `PodFitResources` on each node to find the list of nodes that satifies
the request, then rank the nodes to find the best fit.

If the node runs device plugin for the extended resource, the scheduler does
exactly the same thing as it does today, PodFitsResources checks
`Node.Allocatable` to decide whether there is sufficient devices to satisfy the
request during scheduling phase, and decrements the `Node.Allocatable` during
binding phase.

If the node runs DRA driver that also advertises the extended resource, the
scheduler needs to run a similar algorithm for scheduling resource claim. It
needs to take the following steps:
1. Find all the devices on a node from ResourceSlice.devices
2. Find the allocated devices on a node from ResourceClaim.status.allocation.devices {}
3. Find the allocated devices for extended resources on a node from the special resource
claim object for the extended resource.
4. Subtract the devices list allocated to resourceclaim from the devices list from
resourceslice, the result is the list of available devices on the node.
5. If the number of available devices is sufficient for the request, then the
   node is a fit for the pod.
6. If the node is selected as the best fit for the pod, the pod is scheduled to the pod.
The result of the allocation is recorded in the speical resource claim's
ResourceClaim.status.allocation.devices.

For example, for a pod with pod.containers[].requests nvidia.com/gpu: 1,
scheduler will find the node with available GPU, record the result of the allocation,
e.g. [gpu-5] in status.allocation.devices field of the special resource claim object
for the extended resource nvidia.com/gpu in the same namespace as the pod.

### Scheduling for Resource Claim
When scheduler tries to satisfy a resource claim, it needs to take the following steps:
1. Find all the devices on a node from ResourceSlice.devices
2. Find the allocated devices on a node from ResourceClaim.status.allocation.devices {}
3. Find the allocated devices for extended resources on a node from the special resource
claim object for the extended resource.
4. Subtract the devices list allocated to resourceclaim from the devices list from
resourceslice, the result is the list of available devices on a node
5. Use the result to satisfy the claim
6. The result of the allocation is recorded in the request resource claim's
ResourceClaim.status.allocation.devices

If the resourceclaim has requests.allocationMode: All, then the special resource claim
object for the extended resource MUST not exist in the namespace, or if it exists,
but its status.allocation.devices must be empty.

### Actuation for Extended Resource
When a pod with extended resources is picked up by the kubelet on the node it is scheduled to
run, the following are particularly important:

1. Kubelet tries to admit the pod, it checks the pods’ extended resource requests against the
node’s available devices, if the list of devices stay the same as the time when the pod was
initially scheduled by the kube-scheduler, then the admission check should pass. However, if
some devices allocatd to extended resource in the special resource claim object’s
status.allocations.devices list go offline at the time of the admission check, the check could
fail.

1. Kubelet talks to the driver through the device plugin proto, it passes the devices to the
container runtime through the container device interface. The gRPC method `ListAndWatch` must
return those devices on the special resource claim’s status.allocations.devices list, as this
is the list of devices that have been allocated by the scheduler to extended resources.

Kubelet must only allocate devices listed in special resource claim’s status.allocations.devices
list, though the driver may manage other devices on the node, those devices may be used for
pods with DRA resource claim.

### Actuation for Resource Claim
When a pod with resource claim is picked up by the kublet on the node it is scheduled to run, the
kubelet talks to the driver with DRA proto, it keeps the current behavior without modification.

Briefly, it finds the list of the allocated devices in resource claim, and passes them into
container runtime through container device interface.

### Test Plan

<!--
[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
- `<package>`: `<date>` - `<test coverage>`
-->

Start of v1.33 development cycle (v1.33.0-alpha.x-xxx-xxxxxxxxxxxx):

- `k8s.io/dynamic-resource-allocation/cel`: ??.?%
- `k8s.io/dynamic-resource-allocation/structured`: ??.?%
- `k8s.io/kubernetes/pkg/controller/resourceclaim`: ??.?%
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources`: ??.?%

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

The existing [integration tests for kube-scheduler which measure
performance](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler_perf#readme)
will be extended to cover the overheaad of running the additional logic to
support the features in this KEP. These also serve as [correctness
tests](https://github.com/kubernetes/kubernetes/commit/cecebe8ea2feee856bc7a62f4c16711ee8a5f5d9)
as part of the normal Kubernetes "integration" jobs which cover [the dynamic
resource
controller](https://github.com/kubernetes/kubernetes/blob/294bde0079a0d56099cf8b8cf558e3ae7230de12/test/integration/scheduler_perf/util.go#L135-L139).

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

End-to-end testing depends on a working resource driver and a container runtime
with CDI support. A [test
driver](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/dra/test-driver)
was developed as part of the overall DRA development effort. We will extend
this test driver to enable support for `ExtendedResourceName`s and add tests to
ensure they are handled by the scheduler as described in this KEP.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback
- Additional tests are in Testgrid and linked in KEP

#### GA

- 3 examples of vendors making use of the extensions proposed in this KEP
- Scalability tests that mirror real-world usage as determined by user feedback
- Allowing time for feedback

### Upgrade / Downgrade Strategy

The usual Kubernetes upgrade and downgrade strategy applies for in-tree
components. Vendors must take care that upgrades and downgrades work with the
drivers that they provide to customers.

### Version Skew Strategy

All of the API extensions proposed in this KEP is the optional
`ExtendedResourceName`. There is no risk for version skew downgrades
because these `ResourceSlice` and `Devices` will never have existed in
older clusters.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DRAExtendedResource
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Applications that were already deployed and are running will continue to
work. They will also continue to work when restarted because the CDI devices that
have been prepared for them won't change across the restart.

The DRA driver itself should also be able to survive a rollback, It
will just lose the ability to advertise devices as extended resources.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler may lose track of what devices it has allocated to what pods. Any
pods that had previously allocated devices with the feature enabled will need
to be deleted to ensure they are freed back to their corresponding driver  and
the accounting for them is updated in the scheduler.

###### Are there any tests for feature enablement/disablement?

Objects with the API additions introduced in the KEP are never written by
Kubernetes components themselves. They are written by 3rd-party drivers.
However, the scheduler does consume these objects and track information from
them in order to make scheduling decisions.

Unit tests will be written in the scheduler to verify that enabling /
disabling of the DRAExtendedResource feature gate is non-disruptive to the
scheduler.

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
Will be considered for beta.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->
Will be considered for beta.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->
Will be considered for beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
Will be considered for beta.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->
Will be considered for beta.

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->
Will be considered for beta.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:
-->
Will be considered for beta.

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
Will be considered for beta.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:
-->
Will be considered for beta.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
Will be considered for beta.

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
Will be considered for beta.

### Scalability

No. The API extensions in this KEP are limited to the existing `ResourceSlice`
object, with no additional requirements to consume this object by additional
components.

###### Will enabling / using this feature result in introducing new API types?

No. The this KEP proposes extensions to an existing type, but not a new type itself.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. With the extensions proposed in this KEP, individual
`ResourceSlices` have additional fields available to them, thus increasing
their overall signature.but it is ultimately up to how 3rd party vendors decide to use them.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes. The time to allocate a device to a claim (and thus schedule the first pod
that references that claim) will be affected.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

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

Will be considered for beta.

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
Will be considered for beta.

###### What steps should be taken if SLOs are not being met to determine the problem?

Will be considered for beta.

## Implementation History

- Kubernetes 1.33: KEP accepted as "???".

## Drawbacks

It adds complexity to the scheduler.

## Alternatives

Many different approaches were considered.

Specifically, the following two alternative proposals were considered:

**Option 1:** webhook rewrite extended resource requests in pod spec

This approach requires cluster administrator deploy a mutation webhook to the
cluster, and configure the webhook with rewrite rules that can rewrite the
extended resource requests and node selectors. This approach is not taken due to
the webhook's extra configuration, and maintenance overhead.

**Option 2:** client CLI tool to rewrite extended resource requests in pod spec

This approach requires application developers, operators to run the client CLI
tool to rewrite the application YAML with extended resources to DRA resource
claims. This adds extra overhead to the application deployment flow.
