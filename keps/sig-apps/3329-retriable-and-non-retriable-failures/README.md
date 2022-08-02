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
# KEP-3329: Retriable and non-retriable Pod failures for Jobs

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
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Job-level vs. pod-level spec](#job-level-vs-pod-level-spec)
    - [Relationship with Pod.spec.restartPolicy](#relationship-with-podspecrestartpolicy)
    - [Current state review](#current-state-review)
      - [Preemption](#preemption)
      - [Taint-based eviction](#taint-based-eviction)
      - [Node drain](#node-drain)
      - [Node-pressure eviction](#node-pressure-eviction)
      - [OOM kill](#oom-kill)
      - [Disconnected node](#disconnected-node)
      - [Disconnected node when taint-manager is disabled](#disconnected-node-when-taint-manager-is-disabled)
      - [Direct container kill](#direct-container-kill)
    - [Termination initiated by Kubelet](#termination-initiated-by-kubelet)
    - [JobSpec API alternatives](#jobspec-api-alternatives)
    - [Failing delete after a condition is added](#failing-delete-after-a-condition-is-added)
    - [Marking pods as Failed](#marking-pods-as-failed)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Garbage collected pods](#garbage-collected-pods)
    - [Evolving condition types](#evolving-condition-types)
- [Design Details](#design-details)
  - [New PodConditions](#new-podconditions)
  - [Interim FailureTarget condition](#interim-failuretarget-condition)
  - [JobSpec API](#jobspec-api)
  - [Evaluation](#evaluation)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Upgrade](#upgrade)
    - [Downgrade](#downgrade)
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
  - [Only support for exit codes](#only-support-for-exit-codes)
  - [Using Pod status.reason field](#using-pod-statusreason-field)
  - [Using of various PodCondition types](#using-of-various-podcondition-types)
  - [More nodeAffinity-like JobSpec API](#more-nodeaffinity-like-jobspec-api)
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
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
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

This KEP extends Kubernetes to configure a job policy for handling pod failures.
In particular, the extension allows determining some of pod failures as caused
by infrastructure errors and to retry them without incrementing the counter
towards `backoffLimit`.

Additionally, the extension allows determining some pod failures as caused by
software bugs and to terminate the associated job early. This is needed to save
time and computational resources wasted due to unnecessary retries of containers
destined to fail due to software bugs.

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

## Motivation

Running a large computational workload, comprising thousands of pods on
thousands of nodes requires usage of pod restart policies in order
to account for infrastructure failures.

Currently, kubernetes Job API offers a way to account for infrastructure
failures by setting `.backoffLimit > 0`. However, this mechanism intructs the
job controller to restart all failed pods - regardless of the root cause
of the failures. Thus, in some scenarios this leads to unnecessary
restarts of many pods, resulting in a waste of time and computational
resources. What makes the restarts more expensive is the fact that the
failures may be encountered late in the execution time of a program.

Sometimes it can be determined from containers exit codes
that the root cause of a failure is in the executable and the
job is destined to fail regardless of the number of retries. However, since
the large workloads are often scheduled to run over night or over the
weekend, there is no human assistance to terminate such a job early.

The need for solving the problem has been emphasized by the kubernetes
community in the issues, see: [#17244](https://github.com/kubernetes/kubernetes/issues/17244) and [#31147](https://github.com/kubernetes/kubernetes/issues/31147).

Some third-party frameworks have implemented retry policies for their pods:
- [TensorFlow Training (TFJob)](https://www.kubeflow.org/docs/components/training/tftraining/)
- [Argo workflow](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-conditional.yaml) ([example](https://github.com/argoproj/argo-workflows/blob/master/examples/retry-conditional.yaml))

Additionally, some pod failures are not linked with the container execution,
but rather with the internal kubernetes cluster management (see:
[Scheduling, Preemption and Eviction](https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/)).
Such pod failures should be recognized as infrastructure failures and it
should be possible to ignore them from the counter towards `backoffLimit`.

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals

- Extension of Job API with user-friendly syntax to terminate jobs based on the
  end state of the failed pod.

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

### Non-Goals

- Implementation of other indicators of non-retriable jobs such as termination logs.
- Modification of the semantics for job termination. In particular, allowing for
  all indexes of an indexed-job to execute when only one or a few indexes fail
  [#109712](https://github.com/kubernetes/kubernetes/issues/109712).
- Similar termination policies for other workload controllers such as Deployments
  or StatefulSets.
- Handling of Pod configuration errors resulting in pods stuck in the `Pending`
  state (value of `status.phase`) rather than `Failed` (such as incorrect image
  name, non-matching configMap references, incorrect PVC references).

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

Extension of the Job API with a new field which allows to configure the set of
conditions and associated actions which determine how a pod failure is handled.
The extended Job API supports discrimination of pod failures based on the
container exit codes as well as based on the end state of a failed pod.

In order to support discrimination of pod failures based on their end state
we use the already existing `status.conditions` field to append a dedicated Pod
Condition indicating (by its type) that the pod is being terminated by an
internal kubernetes component. Moreover, we modify the internal kubernetes
components to send an API call to append the dedicated Pod condition along with
sending the associated Pod delete request. In particluar, the following
kubernetes components will be modified:
- kube-controller-manager (taint manager performing pod eviction)
- kube-scheduler (when performing `Preemption`)

We use the job controller's main loop to detect and categorize the pod failures
with respect to the configuration. For each failed pod, one of the following
actions is applied:
- terminate the job (non-retriable failure),
- ignore the failure (retriable failure) - restart the pod and do not increment
  the counter for `backoffLimit`,
- increment the `backoffLimit` counter and restart the pod if the limit is not
  reached (current behaviour).

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a machine learning researcher, I run jobs comprising thousands
of long-running pods on a cluster comprising thousands of nodes. The jobs often
run at night or over weekend without any human monitoring. In order to account
for random infrastructure failures we define `.backoffLimit: 6` for the job.
However, a signifficant portion of the failures happen due to bugs in code.
Moreover, the failures may happen late during the program execution time. In
such case, restarting such a pod results in wasting a lot of computational time.

We would like to be able to automatically detect and terminate the jobs which are
failing due to the bugs in code of the executable, so that the computation resources
can be saved.

Occasionally, our executable fails, but it can be safely restarted with a good
chance of succeeding the next time. In such known retriable situations our
executable exits with a dedicated exit code in the 40-42 range. All remaining
exit codes indicate a software bug and should result in an early job termination.

The following Job configuration could be a good starting point to satisfy
my needs:

```yaml
apiVersion: v1
kind: Job
spec:
  template:
    spec:
      containers:
      - name: job-container
        image: job-image
        command: ["./program"]
  backoffLimit: 6
  podFailurePolicy:
    rules:
    - action: Terminate
      onExitCodes:
        operator: NotIn
        values: [40,41,42]
```

Note that, when no rule specified in `podFailurePolicy` matches the pod failure
the default handling of pod failures applies - the counter of pod failures
is incremented and checked against the `backoffLimit`
(see: [JobSpec API](#jobspec-api)]).

#### Story 2

As a service provider that offers computational resources to researchers I would like to
have a mechanism which terminates jobs for which pods are failing due to user errors,
but allows infinite retries for pod failures caused by cluster-management
events (such as preemption). I do not have knowledge or influence over the executable that researchers run,
so I don't know beforehand which exit codes they might return.

The following Job configuration could be a good starting point to satisfy
my needs:

```yaml
apiVersion: v1
kind: Job
spec:
  template:
    spec:
      containers:
      - name: main-job-container
        image: job-image
        command: ["./program"]
      - name: monitoring-job-container
        image: job-monitoring
        command: ["./monitoring"]
  backoffLimit: 3
  podFailurePolicy:
    rules:
    - action: Ignore
      onPodConditions:
      - type: DisruptionTarget
```

Note that, in this case the user supplies a list of Pod condition type values.
This approach is likely to require an iterative process to review and extend of
the list.

### Notes/Constraints/Caveats (Optional)

#### Job-level vs. pod-level spec

We considered introduction of this feature in the pod spec, allowing to account for
container restarts within a running pod. However, we consider handling of
job-level failures or termination (for example, a Preemption) as an integral part of
this proposal. Also, when pod's `spec.restartPolicy` is specified as `Never`, then the
failures can't be handled by kubelet and need to be handled at job-level anyway.

Also, we consider this proposal as a natural extension of the already exiting
mechanism for job-level restarts of failed pods based on the Job's
`spec.backoffLimit` configuration. In particular, this proposal aims to fix the
issue, in this mechanism, of unnecessary restarts when a Job can be determined
to fail despite retries.

We believe this feature can co-exist with other pod-level features providing
container restart policies. However, if we establish there is a problematic
interaction of this feature with another such feature, then we will consider
additional validation of the JobSpec configuration to avoid the situation.

If, in the future, we introduce failure handling within the Pod spec, it would
be limited to restartPolicy=OnFailure. Only one of the Pod spec or Job spec APIs
will be allowed to be used at a time.

#### Relationship with Pod.spec.restartPolicy

For Alpha we may limit this feature by disallowing the use of `onExitCodes` when
`restartPolicy=OnFailure`. This is in order to avoid the problematic
race-conditions between Kubelet and Job controller. For example, Kubelet
could restart a failed container before the Job controller decides to terminate
the corresponding job due to a rule using `onExitCodes`. On the other hand,
the unnecessary container restart may not be too much of an issue. We are going
to re-evaluate if we want to support `onExitCodes` combined with
`restartPolicy=OnFailure` in Beta.

#### Current state review

Here we review the current state of kubernetes (version 1.24) regarding its
handling pod failures.

The list below contains scenarios which we have reproduced in order to
investigate which pod fields could be used as indicators if a pod failure
should or should not be retried.

The results demonstrate that there is no universal indicator (like a
pod or container field) currently that discriminates pod failures which should
be retried from those which should not be retried.

##### Preemption

- Reproduction: We run two long-running jobs. The second has higher priority
  pod which preempts the lower priority pod
- Comments: controlled by kube-scheduler in `scheduler/framework/preemption/preemption.go`
- Pod status:
  - status: Terminating
  - `phase=Failed`
  - `reason=`
  - `message=`
- Container status:
  - `state=Ternminated`
  - `exitCode=137`
  - `reason=Error`
- Retriable: Yes

##### Taint-based eviction

- Reproduction: We run a long-running job. Then, we taint the node with `NoExecute`
- Comments: controlled by kube-scheduler in `controller/nodelifecycle/scheduler/taint_manager.go`
- Pod status:
  - status: Terminating
  - `phase=Failed`
  - `reason=`
  - `message=`
- Container status:
  - `state=Ternminated`
  - `exitCode=137`
  - `reason=Error`
- Retriable: Yes

##### Node drain

- Reproduction: We run a job with a long-running pod, then drain the node
  with the `kubectl drain` command
- Comments: performed by Eviction API, controlled by kube-apiserver in `registry/core/pod/storage/eviction.go`
- Pod status:
  - status: Terminating
  - `phase=Failed`
  - `reason=`
  - `message=`
- Container status:
  - `state=Ternminated`
  - `exitCode=137`
  - `reason=Error`
- Retriable: Yes

##### Node-pressure eviction

Memory-pressure eviction:

- Reproduction: We run a job with a pod which attempts to allocate more
  memory than available on the node
- Comments: controlled by kubelet in `kubelet/eviction/eviction_manager.go`
- Pod status:
  - status: ContainerStatusUnknown
  - `phase=Failed`
  - `reason=Evicted`
  - `message=The node was low on resource: memory. (...)`
- Container status:
  - `state=Ternminated`
  - `exitCode=137`
  - `reason=ContainerStatusUnknown`
- Retriable: Unclear, excessive memory usage suggests a bug or misconfiguration.
  However, a restart on another node may succeed

Disk-pressure eviction:

- Reproduction: We run a job with a pod which attempts to write more
  data than the disk space available on the node
- Comments: controlled by kubelet in `kubelet/eviction/eviction_manager.go`
- Pod status:
  - status: Error
  - `phase=Failed`
  - `reason=Evicted`
  - `message=The node was low on resource: ephemeral-storage. (...)`
- Container status:
  - `state=Ternminated`
  - `exitCode=137`
  - `reason=Error`
- Retriable: Unclear, excessive disk usage suggests a bug or misconfiguration.
  However, a restart on another node may succeed

##### OOM kill

- Reproduction: We run a job with a pod which attempts to allocate more
  memory than constrained in the container spec by `resources.limits.memory`
- Comments: handled by kubelet
- Pod status:
  - status: OOMKilled
  - `phase=Failed`
  - `reason=`
  - `message=`
- Container status:
  - `state=Ternminated`
  - `exitCode=137`
  - `reason=OOMKilled`
- Retriable: Unclear, but if occurs when
  `resources.requests.memory=resources.limits.memory` it strongly suggests
  a software bug.

##### Disconnected node

- Reproduction: We run a job with a long-running pod, then disconnect the node
  and delete it by the `kubectl delete` command
- Comments: handled by Pod Garbage collector in: `controller/podgc/gc_controller.go`.
  However, the pod phase remains `Running`.
- Pod status:
  - status: Terminating
  - `phase=Running`
  - `reason=`
  - `message=`
- Container status:
  - `state=Running`
  - `exitCode=`
  - `reason=`
- Retriable: Yes

##### Disconnected node when taint-manager is disabled

- Reproduction: Run kube-controller-manager with disabled taint-manager (with the
  flag `--enable-taint-manager=false`). Then, run a job with a long-running pod and
  disconnect the node
- Comments: handled by node lifcycle controller in: `controller/nodelifecycle/node_lifecycle_controller.go`.
  However, the pod phase remains `Running`.
- Pod status:
  - status: Unknown
  - `phase=Running`
  - `reason=NodeLost`
  - `message=Node mycluster-worker which was running pod play-longrun-f28ls is unresponsive`
- Container status:
  - `state=Running`
  - `exitCode=`
  - `reason=`
- Retriable: Yes

##### Direct container kill

- Reproduction: We run a job with a long-running pod, then we kill the container
by the `crictl stop` command
- Comments: handled by Kubelet
- Pod status:
  - status: Error
  - `phase=Failed`
  - `reason=`
  - `message=`
- Container status:
  - `state=Ternminated`
  - `exitCode=137`
  - `reason=Error`
- Retriable: Yes

#### Termination initiated by Kubelet

For Alpha, we limit this feature by not recognizing Pod failures initiated
by Kubelet. This is because it is hard to determine in some scenarios of
Pod failures initiated by Kubelet if they should be retried or should not.
For example, when Kubelet evicts a pod due to node pressure it might mean either
software bug (then it might be better to terminate the entire job) or the node
being low on memory due to other processes (in which case it is sensible to retry).
We are going to re-evaluate for Beta if we want to add support for recognizing
pod terminations initiated by Kubelet.

#### JobSpec API alternatives

Alternative versions of the JobSpec API to define requirements on exit codes and
on pod end state have been proposed and discussed (see: [Alternatives](#alternatives)).
The outcome of the discussions as well as the experience gained during the Alpha
implementation may influence the final API.

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

#### Failing delete after a condition is added

Here we consider a scenario when a component fails (for example its container
dies) between appending a pod condition and deleting the pod.

In particular, scheduler can possibly decide to preempt
a different pod the next time (or none). This would leave a pod with a
condition that it was preempted, when it actually wasn't. This in turn
could lead to inproper handling of the pod by the job controller.

As a solution we are going to implement a worker, added to the disruption
controller which clears the pod condition added if `DeletionTimestamp` is
not added to the pod for a long enough time (for example 2 minutes).

#### Marking pods as Failed

As indicated by our experiments (see: [Disconnected node](#disconnected-node))
a failed pod may get stuck in the `Running` phase when there is no kubelet
working properly. In particular, this happens
in case of orphaned pods which are deleted by garbage-collector. Due to this
issue the logic of detecting pod failures in job controller is more complex than
would be needed otherwise.

Notably, this issue affects also an analogous scenario in which the taint-manager
is disabled [Disconnected node when taint-manager is disabled](#disconnected-node-when-taint-manager-is-disabled).
However, as disabling taint-manager is deprecated it is not a concern for this
KEP.

For Alpha, we implement a fix for this issue by setting the pod phase as `Failed`
in podgc. For Beta, we are going to simplify the code in job controller
responsible for detecting pod failures.

### Risks and Mitigations

#### Garbage collected pods

The Pod status (which includes the `conditions` field and the container exit
codes) could be lost if the failed pod is garbage collected.

Losing Pod's status before it is interpreted by Job Controller can be prevented
by using the feature of [job tracking with finalizers](https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/)
(see more about the design details section: [Interim FailureTarget condition](#interim-failuretarget-condition)).

#### Evolving condition types

The list of available PodCondition types field will be evolving with new
values being added and potentially some values becoming obsolete. This can make
it difficult to maintain a valid list of PodCondition types enumerated in
the Job configuration.

In order to mitigate this risk we are going to define (along with
documentation) the new condition types as constants in the already existing list
defined in the k8s.io/apis/core/v1 package. Thus, every addition of a new
condition type will require an API review. The constants will allow users of the
package to reduce the risk of typing mistakes.

Additionally, for Beta, we will re-evaluate an idea of a generic opinionated condition
type indicating that a pod can be retried, for example `DisruptionTarget`.

Finally, we are going to cover the handling of pod failures associated with the
new PodCondition types in integration tests.

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

As our [review](#current-state-review) shows there is currently no convenient
indicator, in the pod end state, if the pod should be retried or should not.
Thus, we introduce a set of dedicated Pod conditions which can be used for this
reason.

### New PodConditions

A new condition type, called `DisruptionTarget`, is introduced to indicate
a pod failure caused by a disruption. In order to account for different
reasons for pod termination we add the following reason types based on the
invocation context (we focus on covering these scenarios were the new
condition makes it easier to determine if a failed pod should be restarted):
- PreemptionByKubeScheduler (Pod preempted by kube-scheduler)
- DeletionByTaintManager (Pod evicted by kube-controller-manager due to taints)
- EvictionByEvictionAPI (Pod deleted by Eviction API)
- DeletionByPodGC (an orphaned Pod deleted by pod GC)

The already existing `status.conditions` field in Pod will be used by kubernetes
control plane components (kube-scheduler and kube-controller-manager) to append
a dedicated condition when they send the delete operation.

The API call to append the condition will be issued as a pod status update call
before the Pod delete request. This way the Job controller will already see the
condition when handling a failed pod.

During the implementation process we are going to review the places where the
pod delete requests are issued to modify the code to also append a meaningful
condition with dedicated `Type`, `Reason` and `Message` fields based on the
invocation context.

### Interim FailureTarget condition

There is a risk of losing the Pod status information due to PodGC, which could
prevent Job Controller to react to a pod failure with respect to the configured
pod failure policy rules (see also: [Garbage collected pods](#garbage-collected-pods)).

In order to make sure all pods are checked against the rules we require the
feature of [job tracking with finalizers](https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/)
to be enabled.

Additionally, before we actually remove the finalizers from the pods
(allowing them to be deleted by PodGC) we record the determined job failure
message (if any rule with `JobFail` matched) in an interim job condition, called
`FailureTarget`. Once the pod finalizers are removed we update the job status
with the final `Failed` job condition. This strategy eliminates a possible
race condition that we could lose the information about the job failure if
Job Controller crashed between removing the pod finalizers are updating the final
`Failed` condition in the job status.

### JobSpec API

We extend the Job API in order to allow to apply different actions depending
on the conditions associated with the pod failure.

```golang
// PodFailurePolicyAction specifies how a Pod failure is handled.
// +enum
type PodFailurePolicyAction string

const (
	// This is an action which might be taken on a pod failure - mark the
	// pod's job as Failed and terminate all running pods.
	PodFailurePolicyActionFailJob PodFailurePolicyAction = "FailJob"

	// This is an action which might be taken on a pod failure - the counter towards
	// .backoffLimit, represented by the job's .status.failed field, is not
	// incremented and a replacement pod is created.
	PodFailurePolicyActionIgnore PodFailurePolicyAction = "Ignore"

	// This is an action which might be taken on a pod failure - the pod failure
	// is handled in the default way - the counter towards .backoffLimit,
	// represented by the job's .status.failed field, is incremented.
	PodFailurePolicyActionCount PodFailurePolicyAction = "Count"
)

// +enum
type PodFailurePolicyOnExitCodesOperator string

const (
	PodFailurePolicyOnExitCodesOpIn    PodFailurePolicyOnExitCodesOperator = "In"
	PodFailurePolicyOnExitCodesOpNotIn PodFailurePolicyOnExitCodesOperator = "NotIn"
)

// PodFailurePolicyOnExitCodesRequirement describes the requirement for handling
// a failed pod based on its container exit codes. In particular, it lookups the
// .state.terminated.exitCode for each app container and init container status,
// represented by the .status.containerStatuses and .status.initContainerStatuses
// fields in the Pod status, respectively. Containers completed with success
// (exit code 0) are excluded from the requirement check.
type PodFailurePolicyOnExitCodesRequirement struct {
	// Restricts the check for exit codes to the container with the
	// specified name. When null, the rule applies to all containers.
	// When specified, it should match one the container or initContainer
	// names in the pod template.
	// +optional
	ContainerName *string

	// Represents the relationship between the container exit code(s) and the
	// specified values. Containers completed with success (exit code 0) are
	// excluded from the requirement check. Possible values are:
	// - In: the requirement is satisfied if at least one container exit code
	//   (might be multiple if there are multiple containers not restricted
	//   by the 'containerName' field) is in the set of specified values.
	// - NotIn: the requirement is satisfied if at least one container exit code
	//   (might be multiple if there are multiple containers not restricted
	//   by the 'containerName' field) is not in the set of specified values.
	// Additional values are considered to be added in the future. Clients should
	// react to an unknown operator by assuming the requirement is not satisfied.
	Operator PodFailurePolicyOnExitCodesOperator

	// Specifies the set of values. Each returned container exit code (might be
	// multiple in case of multiple containers) is checked against this set of
	// values with respect to the operator. The list of values must be ordered
	// and must not contain duplicates. Value '0' cannot be used for the In operator.
	// At least one element is required. At most 255 elements are allowed.
	// +listType=set
	Values []int32
}

// PodFailurePolicyOnPodConditionsPattern describes a pattern for matching
// an actual pod condition type.
type PodFailurePolicyOnPodConditionsPattern struct {
	// Specifies the required Pod condition type. To match a pod condition
	// it is required that specified type equals the pod condition type.
	Type api.PodConditionType
	// Specifies the required Pod condition status. To match a pod condition
	// it is required that the specified status equals the pod condition status.
	// Defaults to True.
	Status api.ConditionStatus
}

// PodFailurePolicyRule describes how a pod failure is handled when the requirements are met.
// One of OnExitCodes and onPodConditions, but not both, can be used in each rule.
type PodFailurePolicyRule struct {
	// Specifies the action taken on a pod failure when the requirements are satisfied.
	// Possible values are:
	// - FailJob: indicates that the pod's job is marked as Failed and all
	//   running pods are terminated.
	// - Ignore: indicates that the counter towards the .backoffLimit is not
	//   incremented and a replacement pod is created.
	// - Count: indicates that the pod is handled in the default way - the
	//   counter towards the .backoffLimit is incremented.
	// Additional values are considered to be added in the future. Clients should
	// react to an unknown action by skipping the rule.
	Action PodFailurePolicyAction

	// Represents the requirement on the container exit codes.
	// +optional
	OnExitCodes *PodFailurePolicyOnExitCodesRequirement

	// Represents the requirement on the pod conditions. The requirement is represented
	// as a list of pod condition patterns. The requirement is satisfied if at
	// least one pattern matches an actual pod condition. At most 20 elements are allowed.
	// +listType=atomic
	OnPodConditions []PodFailurePolicyOnPodConditionsPattern
}

// PodFailurePolicy describes how failed pods influence the backoffLimit.
type PodFailurePolicy struct {
	// A list of pod failure policy rules. The rules are evaluated in order.
	// Once a rule matches a Pod failure, the remaining of the rules are ignored.
	// When no rule matches the Pod failure, the default handling applies - the
	// counter of pod failures is incremented and it is checked against
	// the backoffLimit. At most 20 elements are allowed.
	// +listType=atomic
	Rules []PodFailurePolicyRule
}

// JobSpec describes how the job execution will look like.
type JobSpec struct {
  ...
	// Specifies the policy of handling failed pods. In particular, it allows to
	// specify the set of actions and conditions which need to be
	// satisfied to take the associated action.
	// If empty, the default behaviour applies - the counter of failed pods,
	// represented by the jobs's .status.failed field, is incremented and it is
	// checked against the backoffLimit. This field cannot be used in combination
	// with .spec.podTemplate.spec.restartPolicy=OnFailure.
	//
	// This field is alpha-level. To use this field, you must enable the
	// `JobPodFailurePolicy` feature gate (disabled by default).
	// +optional
	PodFailurePolicy *PodFailurePolicy
  ...
```

Note that, we do not introduce the `NotIn` operator in
`PodFailurePolicyOnPodConditionsOperator` as its usage could make job
configurations error-prone and hard to maintain.

Additionally, we validate the following constraints for each instance of
PodFailurePolicyRule:
- exactly one of the fields `onExitCodes` and `OnPodConditions` is specified
  for a requirement
- the specified `containerName` matches name of a configurated container

Here is an example Job configuration which uses this API:

```yaml
apiVersion: v1
kind: Job
spec:
  template:
    spec:
      containers:
      - name: main-job-container
        image: job-image
        command: ["./program"]
      - name: monitoring-job-container
        image: job-monitoring
        command: ["./monitoring"]
  backoffLimit: 3
  podFailurePolicy:
    rules:
    - action: Terminate
      onExitCodes:
        containerName: main-job-container
        operator: In
        values: [1,2,3]
    - action: Ignore
      onPodConditions:
      - type: DisruptionTarget
```

### Evaluation

We use the `syncJob` function of the Job controller to evaluate the specified
`podFailurePolicy` rules against the failed pods. It is only the first rule with
matching requirements which is applied as the rules are evaluated in order. If
the pod failure does not match any of the specified rules, then default
handling of failed pods applies.

If we limit this feature to use `onExitCodes` only when `restartPolicy=Never`
(see: [limitting this feature](#limitting-this-feature)), then the rules using
`onExitCodes` are evaluated only against the exit codes in the `state` field
(under `terminated.exitCode`) of `pod.status.containerStatuses` and
`pod.status.initContainerStatuses`. We may also need to check for the exit codes
in `lastTerminatedState` if we decide to support `onExitCodes` when
`restartPolicy=OnFailure`.

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

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

We assess that the Job controller (which is where the most complicated changes
will be done) has adequate test coverage for places which might be impacted by
this enhancement. Thus, no additional tests prior implementing this enhancement
are needed.

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

Unit tests will be added along with any new code introduced. In particular,
the following scenarios will be covered with unit tests:
- handling or ignoring of `spec.podFailurePolicy` by the Job controller when the
  feature gate is enabled or disabled, respectively,
- validation of a job configuration with respect to `spec.podFailurePolicy` by
  kube-apiserver
- handling of a pod failure, in accordance with the specified `spec.podFailurePolicy`,
  when the failure is associated with
  - a failed container with non-zero exit code,
  - a dedicated Pod condition indicating termmination originated by a kubernetes component

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
The core packages (with their unit test coverage) which are going to be modified during the implementation:
- `k8s.io/kubernetes/pkg/controller/job`: `13 June 2022` - `88%`  <!--(handling of failed pods with regards to the configured podFailurePolicy)-->
- `k8s.io/kubernetes/pkg/apis/batch/validation`: `13 June 2022` - `94.4%` <!--(validation of the job configuration with regards to the podFailurePolicy)-->
- `k8s.io/kubernetes/pkg/apis/batch/v1`: `13 June 2022` - `83.6%`  <!--(extension of JobSpec)-->

##### Integration tests

The following scenarios will be covered with integration tests:
- enabling, disabling and re-enabling of the feature gate
- pod failure is triggered by a delete API request along with appending a
  Pod condition indicating termination originated by a kubernetes component
  (we aim to cover all such scenarios)
- pod failure is caused by a failed container with a non-zero exit code

More integration tests might be added to ensure good code coverage based on the
actual implemention.

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

- <test>: <link to test coverage>
-->

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
- <test>: <link to test coverage

-->
The following scenario will be covered with e2e tests:
- early job termination when a container fails with a non-retriable exit code

More e2e test scenarios might be considered during implementation if practical.

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
-->
#### Alpha

- Implementation:
  - handling of failed pods with respect to `spec.podFailurePolicy` by Job controller
  - appending of a dedicated Pod condition (when the Pod termination is
    initiated by a kubernetes control plane component) to the list of Pod
    conditions along with sending the Pod delete request
  - define as a constant and document the new Pod condition Type
  - the feature is limited by disallowing of the use of `onExitCodes` when
    `restartPolicy=OnFailure`
- The feature flag disabled by default
- Tests: unit and integration

#### Beta

- Address reviews and bug reports from Alpha users
- E2e tests are in Testgrid and linked in KEP
- A scalability test to demonstrate the limited impact of the additional API call
  when terminating a Pod
- Re-evaluate modification to kubelet to send a dedicated condition when
  terminating a Pod, based on user feedback (see: [Termination initiated by Kubelet](#termination-initiated-by-kubelet))
- Re-evaluate supporting of `onExitCodes` when `restartPolicy=OnFailure` (see: [Relationship with Pod.spec.restartPolicy](#relationship-with-podspecrestartpolicy))
- Re-evaluate introduction of a generic opinionated condition type
  indicating that a pod should be retried (see: [Evolving condition types](#evolving-condition-types))
- Simplify the code in job controller responsible for detection of failed pods
  based on the fix for pods stuck in the running phase (see: [Marking pods as Failed](marking-pods-as-failed)).
  Also, introduce a worker in the disruption controller to mark pods
  stuck in the pending phase, with set deletionTimestamp, as failed.
- Commonize the code for appending pod conditions between components
- Do not update the pod disruption condition (with type=`DisruptionTarget`) if
  it is already present with `status=True`
- Review and implement if feasible adding of pod conditions with the use of
  [SSA](https://kubernetes.io/docs/reference/using-api/server-side-apply/) client.
- The feature flag enabled by default

#### GA

- Address reviews and bug reports from Beta users
- The feature is unconditionally enabled

<!--
**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

-->

#### Deprecation

N/A

### Upgrade / Downgrade Strategy

#### Upgrade

An upgrade to a version which supports this feature should not require any
additional configuration changes. In order to use this feature after an upgrade
users will need to configure their Jobs by specifying `spec.podFailurePolicy`. The
only noticeable difference in behaviour, without specifying `spec.podFailurePolicy`,
is that Pods terminated by kubernetes components will have an additional
condition appended to `status.conditions`.

#### Downgrade

A downgrade to a version which does not support this feature should not require
any additional configuration changes. Jobs which specified
`spec.podFailurePolicy` (to make use of this feature) will be handled in a
default way.

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

This feature uses an additional API call between kubernetes components to
append a Pod condition when terminating a pod. However, this API call uses
pre-existing API so the version skew does not introduce runtime compatibility
issues.

We use the feature gate strategy for coordination of the feature enablement
between components.

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
  - Feature gate name: PodDisruptionConditions
    - Components depending on the feature gate:
      - kube-apiserver
      - kube-controller-manager
      - kube-scheduler
  - Feature gate name: JobPodFailurePolicy
    - Components depending on the feature gate:
      - kube-apiserver
      - kube-controller-manager
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

Yes. The kubernetes components (kube-scheduler and kube-controller-manager) will
append a Pod Condition along with the request pod delete request.

However, the part of the feature responsible for handling of the failed pods
is opt-in with `.spec.podFailurePolicy`.
<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Using the feature gate is the recommended way. When the feature is disabled
the Job controller manager handles pod failures in the default way even if
`spec.podFailurePolicy` is specified. Additionally, the dedicated Pod Conditions
are no longer appended along with delete requests.

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

The Job controller starts to handle pod failures according to the specified
`spec.podFailurePolicy`. Additionally, again, along with the delete requests, the
dedicated Pod Conditions are appended to Pod's `status.condition`.

###### Are there any tests for feature enablement/disablement?

Yes, unit and integration test for the feature enabled, disabled and transitions.

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

We use the metrics-based approach based on the following metrics (exposed by
kube-controller-manager):
  - `job_finished_total` (existing, extended by a label): the new `reason`
label indicates the reason for the job termination. Possible values are
`PodFailurePolicyRule`, `BackoffLimitExceeded` and`DeadlineExceeded`.
It can be used to determine what is the relative frequency of job terminations
due to different reasons. For example, if jobs are terminated often due to
`BackoffLimitExceeded` it may suggest that the pod failure policy should be extended
with new rules to terminate jobs early more often
  - `job_pod_failure_total` (new): tracks the handling of failed pods. It will
have the `action` label indicating how a pod failure was handled. Possible
values are:`JobTerminated`, `Ignored` and `Counted`. This metric can be used to
assess the coverage of pod failure scenarios with `spec.podFailurePolicy` rules.

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

- [x] Metrics
  - Metric name:
    - `job_sync_duration_seconds` (existing): can be used to see how much the
feature enablement increases the time spent in the sync job
  - Components exposing the metric: kube-controller-manager

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

Yes. An API call to append a Pod condition when deleting the Pod.

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

No.
<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.
<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.
<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.
<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

The additional CPU and memory increase in kube-controller-manager related to
handling of failed pods is negligible and only limited to these jobs which
specify `spec.podFailurePolicy`.

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
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

- 2022-06-23: Initial KEP merged
- 2022-07-12: Preparatory PR "Refactor gc_controller to do not use the deletePod stub" merged
- 2022-07-14: Preparatory PR "efactor taint_manager to do not use getPod and getNode stubs" merged
- 2022-07-20: Preparatory PR "Add integration test for podgc" merged
- 2022-07-28: KEP updates merged
- 2022-08-01: Additional KEP updates merged
- 2022-08-02: PR "Append new pod conditions when deleting pods to indicate the reason for pod deletion" merged
- 2022-08-02: PR "Add worker to clean up stale DisruptionTarget condition" merged
- 2022-08-04: PR "Support handling of pod failures with respect to the configured rules" merged

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

### Only support for exit codes

We considered supporting just exit codes when defining the policy for handling
pod failures. However, this approach alone would not be sufficient to
distinguish pod failures caused by infrastructure issues. A special handling
of such failures is important in some use cases (see: [Story 2](#story-2)).

### Using Pod status.reason field

We considered using of the pod's `status.reason` field to determine
the reason for a pod failure. This field would be set based on the DeleteOptions
reason field associated with the delete API requests. However, this approach is
problematic as then the field would be used to set by multiple components
leading to race-conditions. Also reasons could be arbitrary strings, making it
hard for users to know which reasons to look for in each version.

### Using of various PodCondition types

We considered introducing a set of dedicated PodCondition types
corresponding to different components or reasons in which a pod deletion
is triggered. However, this could be problematic as the list of available
PodCondition types field would be evolving with new values being added and
potentially some values becoming obsolete. This could make it difficult to
maintain a valid list of PodCondition types enumerated in the Job configuration.

### More nodeAffinity-like JobSpec API

Along with introduction of the set of PodCondition types we also considered
a more nodeAffinity-like JobSpec API being able to match against multiple
condition types. It would also support the `key` field for constraining the
`status` field (and potentially other fields). This is an example Job
spec using such API:

```yaml
   podFailurePolicy:
     rules:
     - action: Ignore
      - onPodConditions:
        - key: Type
          operator: In
          values:
          - Evicted
          - Preempted
        - key: Status
          operator: In
          values:
          - True
```

Such API, while more flexible, might be harder to use in practice. Thus, in the
first iteration of the feature, we intend to provide a user-friendly API
targeting the known use-cases. A more flexible API can be considered as a future
improvement.

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
