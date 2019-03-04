---
title: KEP Template
authors:
  - "@egernst"
owning-sig: sig-node
participating-sigs:
reviewers:
  - "@tallclair"
  - "@derekwaynecarr"
  - "@dchen1107"
approvers:
  - TBD
editor: TBD
creation-date: 2019-02-26
last-updated: 2019-04-02
status: provisional
---

# pod overhead

This includes the Summary and Motivation sections.

## Table of Contents

Tools for generating: https://github.com/ekalinin/github-markdown-toc

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

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

* Making runtimeClass selections

## Proposal

Augment the RuntimeClass definition and the `PodSpec` to introduce
the field `Overhead *ResourceRequirements`. This field represents the overhead associated
with running a pod for a given runtimeClass.  A mutating admission controller is
introduced which will update the `Overhead` field in the workload's `PodSpec` to match
what is provided for the selected RuntimeClass, if one is specified.

Kubelet's creation of the pod cgroup will be calculated as the sum of container
`ResourceRequirements` fields, plus the Overhead associated with the pod.

The scheduler, resource quota handling, and Kubelet's pod cgroup creation and eviction handling
will take Overhead into account, as well as the sum of the pod's container requests.

Horizontal and Veritical autoscaling are calculated based on container level statistics,
so should not be impacted by pod Overhead.

### API Design

#### Pod overhead
Introduce a Pod.Spec.Resources field on the pod to specify the pods overhead.

```
Pod {
  Spec PodSpec {
    // Overhead is the resource overhead consumed by the Pod, not including
    // container resource usage. Users should leave this field unset.
    // +optional
    Overhead *ResourceRequirements
  }
}
```

For scheduling, the pod resource requests are added to the container resource requests.

We don't currently enforce resource limits on the pod cgroup, but this becomes feasible once
pod overhead is accountable. If the pod specifies a resource limit, and all containers in the
pod specify a limit, then the sum of those limits becomes a pod-level limit, enforced through the
pod cgroup.

Users are not expected to manually set the pod resources; if a runtimeClass is being utilized,
the manual value will be discarded. See RuntimeController for the proposal for setting these
resources.

Being able to specify resource requirements for a workload at the pod level instead of container level
has been discussed, but isn't proposed in this KEP.

In the event that pod-level requirements are introduced, pod overhead should be kept separate. This simplifies
several scenarios:
 - overhead, once added to the spec, stays with the workload, even if runtimeClass is redefined or removed.
 - the pod spec can be referenced directly from scheduler, resourceQuote controller and kubelet, instead of referencing
 a runtimeClass object which could have possibly been removed.


### RuntimeClass changes

Expand the runtimeClass type to include sandbox overheads:

```
openAPIV3Schema:
     properties:
       spec:
         properties:
           runtimeHandler:
             type: string
             Pattern: '^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*)?$'
+           runtimeCpuReqOverhead:
+             type: string
+             pattern: '^([0-9]+([.][0-9])?)|[0-9]+(m)$'
+           runtimeCpuLimitOverhead:
+             type: string
+             pattern: '^([0-9]+([.][0-9])?)|[0-9]+(m)$'
+           runtimeMemoryReqOverhead:
+             type: string
+             pattern: '^[0-9]+([.][0-9]+)+(Mi|Gi|M|G)$'
+           runtimeMemoryLimitOverhead:
+             type: string
+             pattern: '^[0-9]+([.][0-9]+)+(Mi|Gi|M|G)$'
```

### RuntimeClass admission controller

The pod resource overhead must be defined prior to scheduling, and we shouldn't make the user
do it. To that end, we propose a new mutating admission controller: RuntimeClass. This admission controller
is also proposed for the [native RuntimeClass scheduling KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/runtime-class-scheduling.md).

In the scope of this KEP, The RuntimeClass controller will have a single job: set the pod overhead field in the
workload's PodSpec according to the runtimeClass specified.

It is expected that only the RuntimeClass controller will set Pod.Spec.Overhead. If a value is provided
which does not match what is defined in the runtimeClass, the pod will be rejected.

Going forward, I foresee additional controller scope around runtimeClass:
 -validating the runtimeClass selection: This would require applying some kind of pod-characteristic labels
 (runtimeClass selectors?) which would then be consumed by an admission controller and checked against known
 capabilities on a per runtimeClass basis. This is is beyond the scope of this proposal.
 -Automatic runtimeClass selection: A controller could exist which would attempt to automatically select the
 most appropriate runtimeClass for the given pod. This, again, is beyond the scope of this proposal.

### User Stories [optional]

Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of the system.
The goal here is to make this feel real for users without getting bogged down.

#### Story 2

### Implementation Details/Notes/Constraints [optional]

What are the caveats to the implementation?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they releate.

### Risks and Mitigations

What are the risks of this proposal and how do we mitigate.
Think broadly.
For example, consider both security and how this will impact the larger kubernetes ecosystem.

How will security be reviewed and by whom?
How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.

## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.
Anything that would count as tricky in the implementation and anything particularly challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage expectations).
Please adhere to the [Kubernetes testing guidelines][testing-guidelines] when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. Initial KEP should keep
this high-level with a focus on what signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

#### Examples

These are generalized examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

##### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

##### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and GA/stable, since there's no opportunity for user feedback, or even bug reports, in back-to-back releases.

##### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/conformance-tests.md

### Upgrade / Downgrade Strategy

If applicable, how will the component be upgraded and downgraded? Make sure this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to make use of the enhancement?

### Version Skew Strategy

Set the overhead to the max of the two version until the rollout is complete.  This may be more problematic
if a new version increases (rather than decreases) the required resources.

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded

## Drawbacks [optional]

This KEP introduceds further complexity, and adds a field the PodSpec which users aren't expected to modify.

## Alternatives [optional]

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


### Leaving the PodSpec unchaged

Instead of tracking the overhead associated with running a workload with a given runtimeClass in the PodSpec,
the Kubelet (for pod cgroup creation), the scheduler (for honoring reqests overhead for the pod) and the resource
quota handling (for optionally taking requests overhead of a workload into account) will need to be augmented
to add a sandbox overhead when applicable.

Pros:
 * no changes to the pod spec
 * no need for a mutating admission controller

Cons:
 * handling of the pod overhead is spread out across a few components
 * Not user perceptible from a workload perspective.
 * very complicated if the runtimeClass policy changes after workloads are running

