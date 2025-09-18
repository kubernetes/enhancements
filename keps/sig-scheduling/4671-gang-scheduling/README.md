<!--
- [X] **Pick a hosting SIG.** 
- [X] **Create an issue in kubernetes/enhancements**
- [X] **Make a copy of this template directory.**
- [X] **Fill out as much of the kep.yaml file as you can.**
- [X] **Fill out this file as best you can.**
- [X] **Create a PR for this KEP.**
- [ ] **Merge early and iterate.**

```
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
# KEP-4671: Gang Scheduling using Workload Object


<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Naming](#naming)
  - [Associating Pod into PodGroups](#associating-pod-into-podgroups)
  - [API](#api)
  - [Scheduler Changes](#scheduler-changes)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

In this KEP, kube-scheduler is modified to support gang scheduling[^1]. To implement gang scheduling, kube-scheduler identifies pods that are in a group and waits until all pods reach the same stage of the scheduling/binding cycle before allowing any pods from the group to advance past that point.  If not all pods can reach that point before a timeout expires, then the scheduler stops trying to schedule that group, and all pods release all their resources.  This allows other workloads to try to allocate those resources.

 A new core type called `Workload` is introduced to tell the kube-scheduler that a group of pods should be scheduled together and any policy options related to gang scheduling. Pods have an object reference in their spec to their `Workload`, if any. The `Workload` object is intended to evolve[^2] via future KEPs to support additional kube-scheduler improvements, such as topology-aware scheduling,  

## Motivation

Parallel applications can require communication between every pod in order to begin execution, and then ongoing communication between all pods (such as barrier or all-reduce operations) in order to make progress.  Starting all pods at close to the same time is necessary to run these workloads.  Otherwise, either expensive compute resources are idle, or the application may fail due to an application-level communication timeout.

Gang scheduling has been implemented outside of kube-scheduler at least 4 times[^3].  Some controllers are starting to support multiple Gang Schedulers in order to be portable across different clusters.  Moving support into kube-scheduler makes gang scheduling support available in all Kubernetes distributions and eventually may allow workload controllers to reply on a standard interface to request gang scheduling from the standard or custom schedulers. A standard API may also allow other components to understand workload needs better (such as cluster autoscalers).

Workloads that require gang scheduling often also need all members of the gang to be as topologically "close" to one another as possible, in order to perform adequately. Existing Pod affinity rules influence pod placement, but they do not consider the gang as a unit of scheduling and they do not cause the scheduler to efficiently try multiple mutually exclusive placement options for a set of pods. The design of the Workload object introduced in this KEP anticipates how Gang Scheduling support can evolve over subsequent KEPs into full Topology-aware scheduling support in kube-scheduler, .  

The Workload object will allow kube-scheduler to be more aware that pods are part of workloads with complex internal structure.  Those workloads include builtins like `Job` and `StatefulSet`, and custom workloads, like `JobSet`, `LeaderWorkerSet` and `MPIJob`. All of these workload types are used for AI training and inference use cases.  


<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals
- Workloads requiring gang scheduling can be run on a stock, conformant Kubernetes cluster without any addons. 
  - It becomes easier to write fully-portable examples of sample AI training and inference applications.
  - Systems like Kueue and Volcano.sh can still offer much additional functionality, but the baseline is raised.
  - Controllers and software frameworks can hide some of the details of configuring gang scheduling from their users.
- When a workload requiring gang scheduling is submitted to Kubernetes, the workload does not remain in a "stuck" state.
  - In particular, where some pods are Running and others can never be satisfied (due to limited resources or misconfiguration of the workload).
- When two workloads of the same priority, and both requiring gang scheduling are submitted, a deadlock will not occur.
  - In particular, avoid the case where where each workload holds some resources, and is waiting for the other to release enough resources for it to complete.
- Developers working on kube-scheduler can access information about that behavior of a workload that a pod belongs to without:
  - needing to infer the behavior from what pods it has seen so far
  - validating all pods in a group have identical policy fields.
  - watching and interpreting a large number of different workload specs.
- Authors/users of custom workload controllers can use gang scheduling, and future scheduler improvements without needing to add code to a scheduler to support their workload.
  - Compare to the support for _specific_ workload controllers in Kueue and Volcano.sh.
-  Don't break any existing workloads.
  -  Kube-scheduler must behave the same when workloads don't use the gang-scheduling feature. 
  -  Kube-scheduler must continue to support running all workload types (including bare pods and unknown custom controllers) in the same cluster as workloads that use gang scheduling.

### Non-Goals

- It is not a goal to take away the responsibility from controllers to create pods.
- It is not a goal to bring fairness or multiple workload queues into kube-scheduler.  Kueue and Volcano.sh will continue to provide this.
- It is not a goal to be able to map all the declarative state and behaviors of all workloads into ths `Workload` object. It will focus on state that is relevant to kube-scheduler, and possibly to cluster autoscalers, reschedulers and closely related components.
- Introducing a resource reservation that can later hold pods.  This feature seems desirable, and will be informed by experience gained from _Gang Scheduling woth using Workload Object_. 
- Address resource contention between two separate schedulers. Resource reservations may address this.


## Proposal

The `spec.workload` field will be added to the Pod resource.  A sample pod with this new field looks like this:
```yaml
apiVersion: v1
kind: Pod
spec:
  ...
  workload:
    name: wkld-1
  ...
```

The above pod might be one of several pods created by a `Job` like this.
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: job-1
spec:
  completions: 100
  parallelism: 100
  template:
    spec:
      restartPolicy: OnFailure
      containers:
      - name: ml-worker
        image: awesome-training-program:v1 
        command: ["python", "train.py"]
        resources:
          limits:
            nvidia.com/gpu: 1
        env:
        - name: JOB_COMPLETION_INDEX
          valueFrom:
            fieldRef:
              fieldPath:
               "metadata.annotations['batch.kubernetes.io/job-completion-index']"
```

The `Workload` core resource will be introduced. A `Workload` does not create any pods. It just describes what pods the scheduler should expect to see, and how to treat them.   

 It does not affect pod creation by Job or any other controller.  It only  A sample resource looks like this:
```yaml
apiVersion: scheduling/v1alpha1   
kind: Workload
metadata:
  name: myjob
spec:
  podGroups:   # or gangGroups -- TBD
    - name: "pg1"
      gangMode: Single
      minCount: 100
      schedulingTimeoutSeconds: 60
```


### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

#### Story 2

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

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Naming

* `Workload` is the resource Kind.
* `scheduling` is the ApiGroup.
* `spec.workload` is the name of the new field in pod.
* Within a Workload there is a list of groups of pods. Each group represents a top-level division of pods within a Workload.  Each group can be independently gang scheduled (or not use gang scheduling). This group is named <<[UNRESOLVED community feedback requested]>> `PodGroup` or `GangGroup` for the top level. <<[/UNRESOLVED]>>.  
* In a future , we expect that this group can optionally specify further subdivision into sub groups.  Each sub-group can have an index.  The indexes go from 0 to N, without repeats or gaps. These subgroups are called <<[UNRESOLVED depending on previous unresolved item]>> `PodSubGroup` if `PodGroup` is chosen, or else `RankedGroup` if `GangGroup` is chosen<<[/UNRESOLVED]>>.
* In subsequent KEPs, we expect that a sub-group can optionally specify further subdivision into pod equivalence classes.  All pods in a pod equivalence class have the same values for all fields that affect scheduling feasibility.  These pod equivalence classes are called <<[UNRESOLVED depending on a previous unresolved item]>> `PodSet` if `PodGroup` is chosen, or else `EqGroup` if `GangGroup` is chosen<<[/UNRESOLVED]>>.

### Associating Pod into PodGroups

When a `Workload` consists of a single group of pods needing Gang Scheduling, it is clear which pods belong to the group from the `spec.workload.name` field of the pod.  However `Workload` supports listing multiple list items, and a list item can represent a single group, or a set of identical replica groups.
In these cases, there needs to be additional information to indicate which group a pod belongs to.

<<[UNRESOLVED @erictune @wojtek-t]>> 
Two options are being considered:
  * 

 <<[/UNRESOLVED]>>
The example below uses  `PodGroupSelector` on each `PodGroup` to identify pods.  For the Job example above, this looks like:

```yaml
apiVersion: scheduling/v1alpha1   
kind: Workload
metadata:
  name: myjobset
spec:
  podGroups:   # or gangGroups -- TBD
    - name: "job1"
      gangMode: Single
      minCount: 100
      schedulingTimeoutSeconds: 60
      podGroupSelector:
        jobset.sigs.k8s.io/replicatedjob-name: job1
     - name: "job2"
      gangMode: Single
      minCount: 100
      schedulingTimeoutSeconds: 60
      podGroupSelector:
        jobset.sigs.k8s.io/replicatedjob-name: job2

```
This works well with existing controllers without changes.

Another option being considered is to specify the name of the `PodGroup` in the Pod itself, like this:

```yaml
apiVersion: v1
kind: Pod
name:
  jobset-job1-abc123
spec:
  ...
  workload:
    name: wkld-1
    podGroup: job1
  ...

```
This is more succinct and makes the role of a pod clear just from inspecting the pod.

### API

The `Workload` type will be defined with the following structure:
```go
type Workload struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Spec WorkloadSpec
	Status WorkloadStatus
}

type ReplicaMode string

const (
	ReplicaModeUnreplicated ReplicaMode = "Unreplicated"
)

// WorkloadSpec describes a workload in a portable way that scheduler and related
// tools can understand.  
type WorkloadSpec struct {
  // Optional but recommended to set.
  ControllerRef *v1.ObjectReference
  // PodGroups is a list of groups of pods.
  // Each group may request gang scheduling.
  PodGroups []PodGroup 
}

type GangMode string
const (
	// GangModeGang means that all pods in this PodGroup need to be scheduled as one gang.
	GangModeSingle GangMode = "Single"

	// GangModeOff means that all pods in this PodGroup do not need to be scheduled as a gang.
	GangModeOff GangMode = "Off"

	// GangModeReplicatedGang means that there is a variable number of identical copies of this PodGroup,
       //  as specified in Replicas, and each copy needs to be independently gang scheduled.
	GangModeReplicated GangMode = "Replicated"
)

// GangSchedulingPolicy holds options that affect how gang scheduling of one PodGroup is handled by the scheduler.
type GangSchedulingPolicy struct {
	ShedulingTimeoutSeconds *int
	MinCount *int
}

// PodGroupSelector holds a label selector that identifies pods of one PodGroup.
// Immutable. Only MatchLabels is valid.  Only one key in MatchLabels can have a different value across
// all list items.  Every list item must have a distintict value for this key.  This ensures
// pod groups have disjoint sets of pods in them.
type PodGroupSelector struct {
       // Selector that identifies pods in this gang.
	Selector *metav1.LabelSelector
}


type GangGroup struct {
      Name *string
      GangMode *GangMode // default is GangModeGang 
      RankedGroups *[]RankedGroup

// Topology request applying to all pods in this GangGroup.
TopologyRequest *TopologyRequest
// GangScheduling options applying to all pods in this gang.
// Only specify if GangMode == GangModeGang
GangSchedulingPolicy
}

// PodGroup is a group of pods that may contain multiple shapes (EqGroups) and may contain
// multiple dense indexes (RankedGroups) and which can optionally be replicated in a variable
// number of identical copies.
type PodGroup struct {
  Name *string
  GangMode *GangMode // default is "Single"

	// ReplicatedGangKey must be set when GangMode = "Replicated"
	// It must be one of:
  //   - `metadata.labels['<KEY>']`
	//   Note: Annotations are not supported because they are not indexed easily, and may not
  //         have a form that can be printed without escaping.
	// Each value of that key maps to a different identical gang, which will be anonymous in status,
  // and referred to by the key's value in events and errors.
	ReplicatedGangKey *ObjectFieldSelector
  // Optional when GangMode = "ReplicatedGang".
  // Forbidden otherwise.
  Replicas int

	// PodGroupSelector must be set when GangMode = "Single" or "Off".
	// Must be one of:
  //   - `metadata.labels['<KEY>']`
	//   Note: Annotation not supported because it is not indexed easily, and it may not
  //         have a form that can be printed without escaping.
	// Each value of that key maps to a different gang, which will be anonymous in status,
  // and referred to by the key's value in events and errors.
	PodGroupSelector PodGroupSelector

// GangScheduling options applying to all pods in this PodGroup.
// Only specify if GangMode != GangModeOff
GangSchedulingPolicy
}


type WorkloadStatus struct {
  // Necessary status fields TBD.
}
```

### Scheduler Changes

The Gang Scheduling functionality will be implemented using a combination of kube-scheduler plugins and scheduling framework changes.
The plugin will use  PreEnqueue and WaitOnPermit framework hooks to control the scheduling process of a gang. The scheduling framework changes will watch Workloads, map pods to and from their Workloads and Gang(s) within the workload. 

More detail about scheduled changes is described in [this document](https://docs.google.com/document/d/1lMYkDuGqEoZWfE2b8vjQx0vHieOMyfmi6VHUef5-5is/edit?tab=t.0#heading=h.1p88ilpefnb).


### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
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

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

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
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

<!--
- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

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

### Version Skew Strategy

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

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

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

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

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

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

[^1]: The Kubernetes community uses the term "gang scheduling" to mean "all-or-nothing scheduling of a set of pods" [1,2,3,4,5,6,7,8,9,10,11,12,13]. In the Kubernetes context, it does not imply time-multiplexing (in contrast to prior academic work such as [Feitelson and Rudolph](https://doi.org/10.1016/0743-7315(92)90014-E), and in contrast to [Slurm Gang Scheduling](https://slurm.schedmd.com/gang_scheduling.html)).  

[^2]: [API Design for Gang and Workload-Aware Scheduling](https://docs.google.com/document/d/1ulO5eUnAsBWzqJdk_o5L-qdq5DIVwGcE7gWzCQ80SCM/edit?pli=1&tab=t.0)
[^3]: Volcano.sh, Co-scheduling plugin, Preferred Networks Plugin, and Kueue all implement gang scheduling outside of kube-scheduler.  Additionally, two previous proposals have been made on this KEP's issue.  These alternatives are compared in detail in the [Background tab of the API Design for Gang Scheduling](https://docs.google.com/document/d/1ulO5eUnAsBWzqJdk_o5L-qdq5DIVwGcE7gWzCQ80SCM/edit?pli=1&tab=t.3zjbiyx2yldg).

