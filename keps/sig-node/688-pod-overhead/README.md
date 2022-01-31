# Pod Overhead

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API Design](#api-design)
    - [Pod overhead](#pod-overhead)
    - [Container Runtime Interface (CRI)](#container-runtime-interface-cri)
  - [ResourceQuota changes](#resourcequota-changes)
  - [RuntimeClass changes](#runtimeclass-changes)
  - [RuntimeClass admission controller](#runtimeclass-admission-controller)
  - [Implementation Details](#implementation-details)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Graduation Criteria](#graduation-criteria)
  - [Test Plan](#test-plan)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Introduce pod level resource requirements](#introduce-pod-level-resource-requirements)
  - [Leaving the PodSpec unchanged](#leaving-the-podspec-unchanged)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
  - [Version 1.16](#version-116)
  - [Version 1.18](#version-118)
  - [Version 1.24](#version-124)
<!-- /toc -->

## Release Signoff Checklist

- [X] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [X] KEP approvers have set the KEP status to `implementable`
- [X] Design details are appropriately documented
- [X] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] Graduation criteria is in place
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those
approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Sandbox runtimes introduce a non-negligible overhead at the pod level which must be accounted for
effective scheduling, resource quota management, and constraining.

## Motivation

Pods have some resource overhead. In our traditional linux container (Docker) approach,
the accounted overhead is limited to the infra (pause) container, but also invokes some
overhead accounted to various system components including: Kubelet (control loops), Docker,
kernel (various resources), fluentd (logs). The current approach is to reserve a chunk
of resources for the system components (system-reserved, kube-reserved, fluentd resource
request padding), and ignore the (relatively small) overhead from the pause container, but
this approach is heuristic at best and doesn't scale well.

With sandbox pods, the pod overhead potentially becomes much larger, maybe O(100mb). For
example, Kata containers must run a guest kernel, kata agent, init system, etc. Since this
overhead is too big to ignore, we need a way to account for it, starting from quota enforcement
and scheduling.

### Goals

* Provide a mechanism for accounting pod overheads which are specific to a given runtime solution

### Non-Goals

* making runtimeClass selections
* auto-detecting overhead
* per-container overhead
* creation of pod-level resource requirements

## Proposal

Augment the RuntimeClass definition and the `PodSpec` to introduce
the field `Overhead *ResourceList`. This field represents the overhead associated
with running a pod for a given runtimeClass.  A mutating admission controller is
introduced which will update the `Overhead` field in the workload's `PodSpec` to match
what is provided for the selected RuntimeClass, if one is specified.

Kubelet's creation of the pod cgroup will be calculated as the sum of container
`ResourceRequirements.Limits` fields, plus the Overhead associated with the pod.

The scheduler, resource quota handling, and Kubelet's pod cgroup creation and eviction handling
will take Overhead into account, as well as the sum of the pod's container requests.

Horizontal and Veritical autoscaling are calculated based on container level statistics,
so should not be impacted by pod Overhead.

### API Design

#### Pod overhead
Introduce a Pod.Spec.Overhead field on the pod to specify the pods overhead.

```
Pod {
  Spec PodSpec {
    // Overhead is the resource overhead incurred from the runtime.
    // +optional
    Overhead *ResourceList
  }
}
```

All PodSpec and RuntimeClass fields are immutable, including the `Overhead` field. For scheduling,
the pod `Overhead` is added to the container resource requests.

We don't currently enforce resource limits on the pod cgroup, but this becomes feasible once pod
overhead is accountable. If the pod specifies an overhead, and all containers in the pod specify a
limit, then the sum of those limits and overhead becomes a pod-level limit, enforced through the pod
cgroup.

Users are not expected to manually set `Overhead`; any prior values being set will result in the workload
being rejected. If runtimeClass is configured and selected in the PodSpec, `Overhead` will be set to the value
defined in the corresponding runtimeClass. This is described in detail in
[RuntimeClass admission controller](#runtimeclass-admission-controller).

Being able to specify resource requirements for a workload at the pod level instead of container
level has been discussed, but isn't proposed in this KEP.

In the event that pod-level requirements are introduced, pod overhead should be kept separate. This
simplifies several scenarios:
 - overhead, once added to the spec, stays with the workload, even if runtimeClass is redefined
 or removed.
 - the pod spec can be referenced directly from scheduler, resourceQuota controller and kubelet,
 instead of referencing a runtimeClass object which could have possibly been removed.

#### Container Runtime Interface (CRI)

The pod cgroup is managed by the Kubelet, so passing the pod-level resource to the CRI implementation
is not strictly necessary. However, some runtimes may wish to take advantage of this information, for
instance for sizing the Kata Container VM.

LinuxContainerResources is added to the LinuxPodSandboxConfig for both overhead and container
totals, as optional fields:

```
type LinuxPodSandboxConfig struct {
	Overhead *LinuxContainerResources
	ContainerResources *LinuxContainerResources
}
```

WindowsContainerResources is added to a newly created WindowsPodSandboxConfig for both overhead and container
totals, as optional fields:

```
type WindowsPodSandboxConfig struct {
	Overhead *WindowsContainerResources
	ContainerResources *WindowsContainerResources
}
```

ContainerResources field in the LinuxPodSandboxConfig and WindowsPodSandboxConfig matches the pod-level limits
(i.e. total of container limits). Overhead is tracked separately since the sandbox overhead won't necessarily
guide sandbox sizing, but instead used for better management of the resulting sandbox on the host.

### ResourceQuota changes

Pod overhead will be counted against an entity's ResourceQuota. The controller will be updated to
add the pod `Overhead` to the container resource request summation.

### RuntimeClass changes

Expand the runtimeClass type to include sandbox overhead, `Overhead *Overhead`.

Where Overhead is defined as follows:

```
type Overhead struct {
  PodFixed *ResourceList
}
```

In the future, the `Overhead` definition could be extended to include fields that describe a percentage
based overhead (scale the overhead based on the size of the pod), or container-level overheads. These are
left out of the scope of this proposal.

### RuntimeClass admission controller

The pod resource overhead must be defined prior to scheduling, and we shouldn't make the user
do it. To that end, we propose a mutating admission controller: RuntimeClass. This admission controller
is also proposed for the [native RuntimeClass scheduling KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/runtime-class-scheduling.md).

In the scope of this KEP, The RuntimeClass controller will have a single job: set the pod overhead field in the
workload's PodSpec according to the runtimeClass specified.

It is expected that only the RuntimeClass controller will set Pod.Spec.Overhead. If a value is provided, the pod will
be rejected.

Going forward, I foresee additional controller scope around runtimeClass:
 - validating the runtimeClass selection: This would require applying some kind of pod-characteristic labels
 (runtimeClass selectors?) which would then be consumed by an admission controller and checked against known
 capabilities on a per runtimeClass basis. This is is beyond the scope of this proposal.
 - Automatic runtimeClass selection: A controller could exist which would attempt to automatically select the
 most appropriate runtimeClass for the given pod. This, again, is beyond the scope of this proposal.

### Implementation Details

With this proposal, the following changes are required:
 - Add the new API to the pod spec and RuntimeClass
 - Update the RuntimeClass controller to merge the overhead into the pod spec
 - Update the ResourceQuota controller to account for overhead
 - Update the scheduler to account for overhead
 - Update the kubelet (admission, eviction, cgroup limits) to handle overhead

### Risks and Mitigations

This proposal introduces changes across several Kubernetes components and a change in behavior *if* Overhead fields
are utilized. To help mitigate this risk, I propose that this be treated as a new feature with an independent feature gate.

## Design Details

### Graduation Criteria

This KEP will be treated as a new feature, and will be introduced with a new feature gate,
PodOverhead. Plan to introduce this utilizing maturity levels: alpha, beta and stable. The
following criteria applies to `PodOverhead` feature gate:

Alpha
 - basic support added in node/core APIs, RuntimeClass admission controller, scheduler and kubelet.

Beta
- ensure proper node e2e test coverage is integrated verifying PodOverhead accounting, including e2e-node
and e2e-scheduling.
- add monitoring to allow monitor if the feature is used and is stable. See:
https://github.com/kubernetes/kubernetes/issues/87259

GA
- assuming no negative user feedback based on production experience, promote
  after 2 releases in beta.


### Test Plan

This feature is verified through a combination of unit and e2e tests.

E2E tests will be created to verify appropriate PodOverhead usage by the scheduler
and Kubelet. RuntimeClass admission controller functionality will be exercised
within the scheduler and kubelet e2e tests.

The Kubelet test, part of e2e-node, will verify appropriate pod cgroup sizing.

The scheduling test, part of e2e-scheduling, will verify predication accounts
for overhead when determining node fit.

### Upgrade / Downgrade Strategy

If a cluster is upgraded to enable this feature, the cluster administrator would experience pre-upgrade
behavior until RuntimeClasses are introduced which include a valid overhead field, and workloads are
created which make use of the new RuntimeClass(es).

If a cluster administrator does not want to utilize this feature's behavior after upgrading their cluster,
RuntimeClasses should be used which do not define an overhead field, and workloads should avoid specifying
RuntimeClasses which have an overhead defined.

### Version Skew Strategy

Set the overhead to the max of the two version until the rollout is complete.  This may be more problematic
if a new version increases (rather than decreases) the required resources.

## Drawbacks

This KEP introduces further complexity, and adds a field the PodSpec which users aren't expected to modify.

## Alternatives

In order to achieve proper handling of sandbox runtimes, the scheduler/resourceQuota handling needs to take
into account the overheads associated with running a particular runtimeClass.

### Introduce pod level resource requirements

Rather than just introduce overhead, add support for general pod-level resource requirements. Pod level
resource requirements are useful for shared resources (hugepages, memory when doing emptyDir volumes).

Even if this were to be introduced, there is a benefit in keeping the overhead separate.
 - post-pod creation handling of pod events: if runtimeClass definition is removed after a pod is created,
  it will be very complicated to calculate which part of the pod resource requirements were associated with
  the workloads versus the sandbox overhead.
 - a kubernetes service provider can subsidize the charge-back model potentially and eat the cost of the
 runtime choice, but charge the user for the cpu/memory consumed independent of runtime choice.


### Leaving the PodSpec unchanged

Instead of tracking the overhead associated with running a workload with a given runtimeClass in the PodSpec,
the Kubelet (for pod cgroup creation), the scheduler (for honoring reqests overhead for the pod) and the resource
quota handling (for optionally taking requests overhead of a workload into account) will need to be augmented
to add a sandbox overhead when applicable.

Pros:
 * no changes to the pod spec
 * user does not have the option of setting the overhead
 * no need for a mutating admission controller

Cons:
 * handling of the pod overhead is spread out across a few components
 * Not user perceptible from a workload perspective.
 * very complicated if the runtimeClass policy changes after workloads are running

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

Skipping this section as the feature was already rolled out to all supported k8s versions.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

Using metrics mentioned in documentation https://kubernetes.io/docs/concepts/scheduling-eviction/pod-overhead/#observability:

- `kube_pod_overhead_cpu_cores`
- `kube_pod_overhead_memory_bytes`

###### How can someone using this feature know that it is working for their instance?

Using metrics mentioned in documentation https://kubernetes.io/docs/concepts/scheduling-eviction/pod-overhead/#observability:

- `kube_pod_overhead_cpu_cores`
- `kube_pod_overhead_memory_bytes`

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The ultimate SLO is that Pod will not be evicted when it does not exceed set
limits because of the overhead introduced by the runtime. Due to the complex
nature of estimating resources Pod and runtime use, this is hard to measure.

Closest approximation to the intended SLO is that Pod's `Overhead` will be
updated on admission and cgroups will be adjusted as needed.

Since RuntimeClass Admission controller logic is straightforward and does not
introduce any new API calls, just one value assignment, Pod scheduling
latency is not affected by this feature.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Excessive Pod evictions on specific runtime that specifies an Overhead, may
indicate that feature is not working. However this is a proxy indication that
is very unreliable - there is a big chance that evictions are caused by Pod or
Runtime behavior.

Checking Pod object and cgroup settings as described in [Usage Example](https://kubernetes.io/docs/concepts/scheduling-eviction/pod-overhead/#usage-example)
section of the documentation may be used as a good proxy to check that the
feature is functional.

Finally, increased pod scheduling latency may indicate an issue with the
RuntimeClass admission controller.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

### Dependencies

No

###### Does this feature depend on any specific services running in the cluster?

The feature depends on RuntimeClass admission controller presence.

### Scalability


###### Will enabling / using this feature result in any new API calls?

No, RuntimeClass is already being checked for every pod in RuntimeClass
Admission Controller and PodOverhead assignment doesn't introduce any new API
calls. Same for the Kubelet.

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Every Pod that is scheduled for the RuntimeClass with the Overhead specified
will carry two additional values for the `Overhead` structure.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

N/A.

Note, specifying PodOverhead will increase the allocated resources for pods by design.

### Troubleshooting

Documentation has troubleshooting steps: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-overhead/

###### How does this feature react if the API server and/or etcd is unavailable?

No dependency on etcd availability.

###### What are other known failure modes?

No

###### What steps should be taken if SLOs are not being met to determine the problem?

- Validate the RuntimeClass Admission controller is functional
- Validate that Pod objects are updated correctly
- Validate that cgroups are updated correctly

## Implementation History

- 2019-04-04: Initial KEP published.

### Version 1.16

- Implemented as Alpha.

### Version 1.18

- Promoted to Beta.

### Version 1.24

1. Production usage: https://github.com/openshift/sandboxed-containers-operator/blob/0edbfbf353945dec4066a6d127bf9d88fbbc80a7/controllers/openshift_controller.go#L342
2. Documentation is in place: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-overhead/

- Promoted to stable
