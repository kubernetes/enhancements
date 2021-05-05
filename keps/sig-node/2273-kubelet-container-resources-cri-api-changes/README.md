# Container Resources CRI API Changes for Pod Vertical Scaling

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Expected Behavior of CRI Runtime](#expected-behavior-of-cri-runtime)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [Stable](#stable)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

This proposal aims to improve the Container Runtime Interface (CRI) APIs for
managing a Container's CPU and memory resource configurations on the runtime.
It seeks to extend UpdateContainerResources CRI API such that it works for
Windows, and other future runtimes besides Linux. It also seeks to extend
ContainerStatus CRI API to allow Kubelet to discover the current resources
configured on a Container.

## Motivation

In-Place Pod Vertical Scaling feature relies on Container Runtime Interface
(CRI) to update the CPU and/or memory limits for Container(s) in a Pod.

The current CRI API set has a few drawbacks that need to be addressed:
1. UpdateContainerResources CRI API takes a parameter that describes Container
   resources to update for Linux Containers, and this may not work for Windows
   Containers or other potential non-Linux runtimes in the future.
1. There is no CRI mechanism that lets Kubelet query and discover the CPU and
   memory limits configured on a Container from the Container runtime.
1. The expected behavior from a runtime that handles UpdateContainerResources
   CRI API is not very well defined or documented.

### Goals

This proposal has two primary goals:
  - Modify UpdateContainerResources to allow it to work for Windows Containers,
    as well as Containers managed by other runtimes besides Linux,
  - Provide CRI API mechanism to query the Container runtime for CPU and memory
    resource configurations that are currently applied to a Container.

An additional goal of this proposal is to better define and document the
expected behavior of a Container runtime when handling resource updates.

### Non-Goals

Definition of expected behavior of a Container runtime when it handles CRI APIs
related to a Container's resources is intended to be a high level guide.  It is
a non-goal of this proposal to define a detailed or specific way to implement
these functions. Implementation specifics are left to the runtime, within the
bounds of expected behavior.

## Proposal

One key change is to make UpdateContainerResources API work for Windows, and
any other future runtimes, besides Linux by making the resources parameter
passed in the API specific to the target runtime.

Another change in this proposal is to extend ContainerStatus CRI API such that
Kubelet can query and discover the CPU and memory resources that are presently
applied to a Container.

To accomplish aforementioned goals:

* A new protobuf message object named *ContainerResources* that encapsulates
LinuxContainerResources and WindowsContainerResources is introduced as below.
  - This message can easily be extended for future runtimes by simply adding a
    new runtime-specific resources struct to the ContainerResources message.
```
// ContainerResources holds resource configuration for a container.
message ContainerResources {
    // Resource configuration specific to Linux container.
    LinuxContainerResources linux = 1;
    // Resource configuration specific to Windows container.
    WindowsContainerResources windows = 2;
}
```

* UpdateContainerResourcesRequest message is extended to carry
  ContainerResources field as below.
  - For Linux runtimes, Kubelet fills UpdateContainerResourcesRequest.Linux in
    additon to UpdateContainerResourcesRequest.Resources.Linux fields.
    - This keeps backward compatibility by letting runtimes that rely on the
      current LinuxContainerResources continue to work, while enabling newer
      runtime versions to use UpdateContainerResourcesRequest.Resources.Linux,
    - It enables deprecation of UpdateContainerResourcesRequest.Linux field.
```
message UpdateContainerResourcesRequest {
    // ID of the container to update.
    string container_id = 1;
    // Resource configuration specific to Linux container.
    LinuxContainerResources linux = 2;
    // Resource configuration for the container.
    ContainerResources resources = 3;
}
```

* ContainerStatus message is extended to return ContainerResources as below.
  - This enables Kubelet to query the runtime and discover resources currently
    applied to a Container using ContainerStatus CRI API.
```
@@ -914,6 +912,8 @@ message ContainerStatus {
     repeated Mount mounts = 14;
     // Log path of container.
     string log_path = 15;
+    // Resource configuration of the container.
+    ContainerResources resources = 16;
 }
```

* ContainerManager CRI API service interface is modified as below.
  - UpdateContainerResources takes ContainerResources parameter instead of
    LinuxContainerResources.
```
--- a/staging/src/k8s.io/cri-api/pkg/apis/services.go
+++ b/staging/src/k8s.io/cri-api/pkg/apis/services.go
@@ -43,8 +43,10 @@ type ContainerManager interface {
        ListContainers(filter *runtimeapi.ContainerFilter) ([]*runtimeapi.Container, error)
        // ContainerStatus returns the status of the container.
        ContainerStatus(containerID string) (*runtimeapi.ContainerStatus, error)
-       // UpdateContainerResources updates the cgroup resources for the container.
-       UpdateContainerResources(containerID string, resources *runtimeapi.LinuxContainerResources) error
+       // UpdateContainerResources updates resource configuration for the container.
+       UpdateContainerResources(containerID string, resources *runtimeapi.ContainerResources) error
        // ExecSync executes a command in the container, and returns the stdout output.
        // If command exits with a non-zero exit code, an error is returned.
        ExecSync(containerID string, cmd []string, timeout time.Duration) (stdout []byte, stderr []byte, err error)
```

* Kubelet code is modified to leverage these changes.

## Design Details

Below diagram is an overview of Kubelet using UpdateContainerResources and
ContainerStatus CRI APIs to set new container resource limits, and update the
Pod Status in response to user changing the desired resources in Pod Spec.

```
   +-----------+                   +-----------+                  +-----------+
   |           |                   |           |                  |           |
   | apiserver |                   |  kubelet  |                  |  runtime  |
   |           |                   |           |                  |           |
   +-----+-----+                   +-----+-----+                  +-----+-----+
         |                               |                              |
         |       watch (pod update)      |                              |
         |------------------------------>|                              |
         |     [Containers.Resources]    |                              |
         |                               |                              |
         |                            (admit)                           |
         |                               |                              |
         |                               |  UpdateContainerResources()  |
         |                               |----------------------------->|
         |                               |                         (set limits)
         |                               |<- - - - - - - - - - - - - - -|
         |                               |                              |
         |                               |      ContainerStatus()       |
         |                               |----------------------------->|
         |                               |                              |
         |                               |     [ContainerResources]     |
         |                               |<- - - - - - - - - - - - - - -|
         |                               |                              |
         |      update (pod status)      |                              |
         |<------------------------------|                              |
         | [ContainerStatuses.Resources] |                              |
         |                               |                              |

```

* Kubelet invokes UpdateContainerResources() CRI API in ContainerManager
  interface to configure new CPU and memory limits for a Container by
  specifying those values in ContainerResources parameter to the API. Kubelet
  sets ContainerResources parameter specific to the target runtime platform
  when calling this CRI API.

* Kubelet calls ContainerStatus() CRI API in ContainerManager interface to get
  the CPU and memory limits applied to a Container. It uses the values returned
  in ContainerStatus.Resources to update ContainerStatuses[i].Resources.Limits
  for that Container in the Pod's Status.

### Expected Behavior of CRI Runtime

TBD

### Test Plan

* Unit tests are updated to reflect use of ContainerResources object in
  UpdateContainerResources and ContainerStatus APIs.

* E2E test is added to verify UpdateContainerResources API with docker runtime.

* E2E test is added to verify ContainerStatus API using docker runtime.

* E2E test is added to verify backward compatibility usign docker runtime.

### Graduation Criteria

#### Alpha

* UpdateContainerResources and ContainerStatus API changes are done and tested
  with dockershim and docker runtime, backward compatibility is maintained.

#### Beta

* UpdateContainerResources and ContainerStatus API changes are completed and
  tested for Windows runtime.

#### Stable

* No major bugs reported for three months.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/20190731-production-readiness-review-process.md.

The production readiness review questionnaire must be completed for features in
v1.19 or later, but is non-blocking at this time. That is, approval is not
required in order to be in the release.

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

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: InPlacePodVerticalScaling
    - Components depending on the feature gate: kubelet
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?** No

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?** Yes

* **What happens if we reenable the feature if it was previously rolled back?**

* **Are there any tests for feature enablement/disablement?** Unit tests

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
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

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
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

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2019-10-25 - Initial KEP draft created
- 2020-01-14 - Test plan and graduation criteria added

