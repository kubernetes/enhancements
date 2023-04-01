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
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Job-level vs. pod-level spec](#job-level-vs-pod-level-spec)
    - [Relationship with Pod.spec.restartPolicy](#relationship-with-podspecrestartpolicy)
    - [Current state review](#current-state-review)
      - [Preemption](#preemption)
      - [Taint-based eviction](#taint-based-eviction)
      - [Node drain](#node-drain)
      - [Node-pressure eviction](#node-pressure-eviction)
      - [Container memory limit exceeded](#container-memory-limit-exceeded)
      - [Container ephemeral-storage limit exceeded](#container-ephemeral-storage-limit-exceeded)
      - [Graceful node shutdown](#graceful-node-shutdown)
      - [Pod admission error](#pod-admission-error)
      - [Disconnected node](#disconnected-node)
      - [Disconnected node when taint-manager is disabled](#disconnected-node-when-taint-manager-is-disabled)
      - [Direct container kill](#direct-container-kill)
    - [Termination initiated by Kubelet](#termination-initiated-by-kubelet)
      - [Active deadline timeout exceeded](#active-deadline-timeout-exceeded)
      - [Admission failures](#admission-failures)
      - [Resource limits exceeded](#resource-limits-exceeded)
    - [JobSpec API alternatives](#jobspec-api-alternatives)
    - [Failing delete after a condition is added](#failing-delete-after-a-condition-is-added)
    - [Marking pods as Failed](#marking-pods-as-failed)
      - [Review of example steps for the 3rd scenario](#review-of-example-steps-for-the-3rd-scenario)
      - [Proposed solution for the 3rd scenario](#proposed-solution-for-the-3rd-scenario)
      - [Implementation progress and plan](#implementation-progress-and-plan)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Garbage collected pods](#garbage-collected-pods)
    - [Evolving condition types](#evolving-condition-types)
    - [Stale DisruptionTarget condition which is not cleaned up](#stale-disruptiontarget-condition-which-is-not-cleaned-up)
- [Design Details](#design-details)
  - [New PodConditions](#new-podconditions)
  - [Interim FailureTarget Job condition](#interim-failuretarget-job-condition)
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
  - [Possible future extensions](#possible-future-extensions)
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
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
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
failures by setting `.backoffLimit > 0`. However, this mechanism instructs the
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
- Adding pod conditions to indicate admission failures
  (see: [active deadline timeout exceeded](#active-deadline-timeout-exceeded))
  or exceeding of the active deadline timeout
  (see: [admission failures](#admission-failures).
  Also, adding pod conditions to
  indicate failures tue to exhausting the resource (memory or ephemeral storage)
  (see: [Resource limits exceeded](#resource-limits-exceeded)).
- Adding of the disruption condition for graceful node shutdown on Windows, as
  there is some ground work required first to support Pod eviction due to
  graceful node shutdown on Windows.

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
- kube-controller-manager (Pod deletion by Taint Manager or PodGC)
- kube-scheduler (Pod deletion due to `Preemption`)
- kube-apiserver (API-initiated eviction)
- kubelet (Pod eviction due to exceeded resource limits, node shutdown, node pressure etc.)

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
However, a significant portion of the failures happen due to bugs in code.
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
    - action: FailJob
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

Additionally, I would like to avoid unnecessary retries of Pods which exceed the
configured memory or ephemeral-storage limits.

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
        resources:
          requests:
            memory: "10Gi"
            ephemeral-storage: "10Gi"
          limits:
            memory: "10Gi"
            ephemeral-storage: "100Gi"
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

#### Story 3

As a service provider, similarly as in [Story 2](#story-2), I would like to
have a mechanism which terminates jobs for which pods are failing due to user
errors. However, I'm concerned that jobs running on a cluster with a broken
configuration might result in costly never-ending Pod retries.

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
        resources:
          requests:
            memory: "10Gi"
            ephemeral-storage: "10Gi"
          limits:
            memory: "10Gi"
            ephemeral-storage: "100Gi"
  backoffLimit: 3
  podFailurePolicy:
    rules:
    - action: Count
      onPodConditions:
      - type: DisruptionTarget
    - action: FailJob
      onExitCodes:
        operator: NotIn
        values: [0]
```

Here we count all disruptions in the counter towards `.spec.backoffLimit`. All
Pod failures caused by exceeding the configured limits or with non-zero exit code
are categorized as user errors and result in termination of the entire job.

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

We limit this feature by disallowing the use of `restartPolicy=OnFailure` when
the pod failure policy is specified by the `podFailurePolicy` field. Since the
Job controller only supports `restartPolicy=Never` and `restartPolicy=OnFailure`,
it effectively means that the use of job's pod failure policy requires
`restartPolicy=Never`.

This is in order to avoid the problematic race-conditions between Kubelet and
Job controller. For example, Kubelet could restart a failed container before the
Job controller decides to terminate the corresponding job due to a rule using
`onExitCodes`.

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
  - `state=Terminated`
  - `exitCode=137`
  - `reason=Error`

##### Taint-based eviction

- Reproduction: We run a long-running job. Then, we taint the node with `NoExecute`
- Comments: controlled by kube-scheduler in `controller/nodelifecycle/scheduler/taint_manager.go`
- Pod status:
  - status: Terminating
  - `phase=Failed`
  - `reason=`
  - `message=`
- Container status:
  - `state=Terminated`
  - `exitCode=137`
  - `reason=Error`

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
  - `state=Terminated`
  - `exitCode=137`
  - `reason=Error`

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
  - `state=Terminated`
  - `exitCode=137`
  - `reason=ContainerStatusUnknown`

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
  - `state=Terminated`
  - `exitCode=137`
  - `reason=Error`

##### Container memory limit exceeded

Linux:

- Reproduction: We run a job with a pod which attempts to allocate more
  memory than constrained in the container spec by `resources.limits.memory`
- Comments: handled by kubelet in `kubelet/kubelet_pods.go` which merges-in the status update (setting of the reason field) by the container runtime
- Pod status:
  - status: OOMKilled
  - `phase=Failed`
  - `reason=`
  - `message=`
- Container status:
  - `state=Terminated`
  - `exitCode=137`
  - `reason=OOMKilled`

Windows:

- Reproduction: We run a job with a pod which attempts to allocate more
  memory than constrained in the container spec by `resources.limits.memory`
- Comments: there is not clear indication that the container failed due to exceeding memory limit
- Pod status:
  - status: Error
  - `phase=Failed`
  - `reason=`
  - `message=`
- Container status:
  - `state=Terminated`
  - `exitCode=1`
  - `reason=Error`

##### Container ephemeral-storage limit exceeded

- Reproduction: We run a job with a pod which attempts to consume more disk space
  than constrained in the container spec by `resources.limits.ephemeral-storage`
- Comments: handled by kubelet in `kubelet/eviction/eviction_manager.go`
- Pod status:
  - status: Error
  - `phase=Failed`
  - `reason=Evicted`
  - `message=Pod ephemeral local storage usage exceeds the total limit of containers 1Gi.`
- Container status:
  - `state=Terminated`
  - `exitCode=137`
  - `reason=Error`

##### Graceful node shutdown

Container does not have a dedicated SIGTERM handling and exits with status 137:

- Reproduction: The node needs to be started with positive `shutdownGracePeriod`,
  for example `shutdownGracePeriod=30s`. We run a job with a long-running pod,
  then we prepare the node for shutdown by the command:
  `dbus-send --system --type=signal /org/freedesktop/login1 org.freedesktop.login1.Manager.PrepareForShutdown boolean:"true"`,
  which similulates graceful nodes shutdown for maintenance purposes.
- Comments: handled by kubelet in `kubelet/nodeshutdown/nodeshutdown_manager_linux.go`
- Pod status:
  - status: Error
  - `phase=Failed`
  - `reason=Terminated`
  - `message=Pod was terminated in response to imminent node shutdown.`
- Container status:
  - `state=Terminated`
  - `exitCode=137`
  - `reason=Error`

Container handles SIGTERM and exits with status 0:

- Reproduction: As above, but the container handles SIGTERM and exits with status 0.
- Comments: handled by kubelet in `kubelet/nodeshutdown/nodeshutdown_manager_linux.go`
- Pod status:
  - status: Completed
  - `phase=Succeeded`
  - `reason=`
  - `message=`
- Container status:
  - `state=Terminated`
  - `exitCode=0`

##### Pod admission error

Admission error due to disk pressure:

- Reproduction: We run a pod on a node which is under disk pressure
  (tainted with `node.kubernetes.io/disk-pressure:NoSchedule`). In order to
  schedule the pod we untaint the node by command line. The Pod is scheduled
  but fails admission by Kubelet as the taint is re-added shortly after its
  manual removal.
- Comments: controlled by kubelet in `kubelet/kubelet.go`
- Pod status:
  - status: Evicted
  - `phase=Failed`
  - `reason=Evicted`
  - `message=Pod The node had condition: [DiskPressure].`
- No containers created

Note that, admission errors may occur due to various other reasons, resulting
in different messages for pods.

##### Disconnected node

- Reproduction: We run a job with a long-running pod with finalizer (for example
  created by the Job controller), then disconnect the node
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

##### Disconnected node when taint-manager is disabled

- Reproduction: Run kube-controller-manager with disabled taint-manager (with the
  flag `--enable-taint-manager=false`). Then, run a job with a long-running pod and
  disconnect the node
- Comments: handled by node lifecycle controller in: `controller/nodelifecycle/node_lifecycle_controller.go`.
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
  - `state=Terminated`
  - `exitCode=137`
  - `reason=Error`

#### Termination initiated by Kubelet

In Alpha, there is no support for Pod conditions for failures or disruptions initiated by kubelet.

For Beta we introduce handling of Pod failures initiated by Kubelet by adding
the pod disruption condition (introduced in Alpha) in case of disruptions
initiated by Kubelet (see [Design details](#design-details)).

Kubelet can also evict a pod in some scenarios which are not covered with
adding a pod failure condition:
- [active deadline timeout exceeded](#active-deadline-timeout-exceeded)
- [admission failures](#admission-failures)
- [resource limits exceeded](#resource-limits-exceeded)

##### Active deadline timeout exceeded

Kubelet can also evict a pod due to exceeded active deadline timeout (configured
by pod's `.spec.activeDeadlineSeconds` field). On one hand, exceeding the timeout,
may suggest a software bug due to which the pod executes longer than expected.
On the other hand, it might be due node CPU pressure caused by other processes
on the node. Thus, in order to give users freedom of handling this situation we
should introduce a dedicated pod condition type, such as `ActiveDeadlineExceeded`.
However, as the feature focuses on scenarios which can be naturally interpreted
in terms of retriability and evolving Pod condition types
(see [evolving condition types](#evolving-condition-types)) are a concern we decide
to do not add any pod condition in this case. It should be re-considered in the
future if there is a good motivating use-case.

##### Admission failures

In some scenarios a pod admission failure could result in a successful pod restart on another
node (for example a pod scheduled to a node with resource pressure, see:
[Pod admission error](#pod-admission-error)). However, in other situations it won't be as clear,
since the failure can be caused by incompatible pod and node configurations. Node configurations
are often the same within a cluster, so it is likely that the pod would fail if restarted on any
other node in the cluster. In that case, adding `DisruptionTarget` condition could cause
a never-ending loop of retries, if the pod failure policy was configured to ignore such failures.
Given the above, we decide not to add any pod condition for such failures. If there is a sufficient
motivating use-case, a dedicated pod condition might be introduced to annotate some of the
admission failure scenarios.

##### Resource limits exceeded

A Pod failure initiated by Kubelet can be caused by exceeding pod's
(or container's) resource (memory or ephemeral-storage) limits. We have considered
(and prepared an initial implementation, see PR
[Add ResourceExhausted pod condition for oom killer and exceeding of local storage limits](https://github.com/kubernetes/kubernetes/pull/113436))
introduction of a dedicated Pod failure condition `ResourceExhausted` to
annotate pod failures due to the above scenarios.

However, it turned out, that there are complications with detection of exceeding
memory limits:
- the approach we considered is to rely on the Out-Of-Memory (OOM) killer.
In particular, we could detect that a pod was terminated due to OOM killer based
on the container's `reason` field being equal to `OOMKilled`. This value is set
on Linux by the leading container runtime implementations: containerd (see
[here](https://github.com/containerd/containerd/blob/23f66ece59654ea431700576b6020baffe1a4e49/pkg/cri/server/events.go#L344)
for event handling and
[here](https://github.com/containerd/containerd/blob/36d0cfd0fddb3f2ca4301533a7e7dcf6853dc92c/pkg/cri/server/helpers.go#L62)
for the constant definition)
and CRI-O (see
[here](https://github.com/cri-o/cri-o/blob/edf889bd277ae9a8aa699c354f12baaef3d9b71d/server/container_status.go#L88-L89)).
- setting the `reason` field to `OOMKilled` is not standardized, either. During
the Beta phase implementation we discussed the standardization issue within the
community (involving CNCF Technical Advisory Group for Runtime and SIG-node).
We have also started an effort to standardize the handling of OOM killed
containers (see:
[Documentation for the CRI API reason field to standardize the field for containers terminated by OOM killer](https://github.com/kubernetes/kubernetes/pull/112977)).
However, in the process it turned out that in some configurations
(for example the CRI-O with cgroupv2, see:
[Add e2e_node test for oom killed container reason](https://github.com/kubernetes/kubernetes/pull/113205)),
the container's `reason` field is not set to `OOMKilled`.
- OOM killer might get invoked not only when container's limits are exceeded,
but also when the system is running low on memory. In such scenario there
can be race conditions in which both the `DisruptionTarget` condition and the
`ResourceExhausted` could be added.

Thus, we've decided not to annotate the scenarios with the `ResourceExhausted`
condition in this KEP. Handling of exceeded limits might be done as a
follow up KEP once the ground work of introducing pod failure conditions is done.

While there are not known issues with detection of the exceeding of
Pod's ephemeral storage limits, we prefer to avoid future extension of the
semantics of the new condition type. Alternatively, we could introduce a pair of
dedicated pod condition types: `OOMKilled` and `EphemeralStorageLimitExceeded`.
This approach, however, could create an unnecessary proliferation of the pod
condition types.

Finally, we would like to first hear user feedback on the preferred approach
and also on how important it is to cover the resource limits exceeded scenarios.

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
could lead to improper handling of the pod by the job controller.

As a solution we implement a worker, part of the disruption
controller, which clears the pod condition added if `DeletionTimestamp` is
not added to the pod for a long enough time (for example 2 minutes).

#### Marking pods as Failed

When matching a failed pod against Job pod failure policy it is important that
the pod is actually in the terminal phase (`Failed`), to ensure their state is
not modified while Job controller matches them against the pod failure policy.

However, there are scenarios in which a pod gets stuck in a non-terminal phase,
but is doomed to be failed, as it is terminating (has `deletionTimestamp` set, also
known as the `DELETING` state, see:
[The API Object Lifecycle](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/object-lifecycle.md)).
In order to workaround this issue, Job controller, when pod failure policy is
disabled, considers any terminating pod that is in a non-terminal phase as failed.
Note that, it is important that when Job controller considers such pods as failed
so that it removes their finalizers and thus allows the API server to complete
their deletion.

In order to ensure consistency in behavior when handling pod failures when
pod failure policy is used or not, we need to make sure that all pods which
are terminating (were previously considered as failed by the Job controller),
are transitioned to the failed state eventually.

Thus, we review scenarios in which a pod is considered by Job controller as
failed, but may get stuck in a non-terminal. Note that, the following scenarios
are not only problematic to the Job controller (due to its use of finalizers),
but they can be considered as bugs in their own right, as every pod should end
up in a terminal phase, even if not started
(see [discussion](https://github.com/kubernetes/enhancements/pull/3757#discussion_r1095197982)).

1. **Orphan pods** (solved in Alpha)

This pod state is characterized by the following:
- scheduled (the Pod's `.spec.nodeName` is set), but the node no longer exists
- `Running` or `Pending` phase

Note that, as PodGC sends DELETE request for such pod (DELETE request was sent
prior to this KEP, but without setting the phase to `Failed`), it becomes also
terminating (has `deletionTimestamp` set).

Example steps leading to the Pod's state:
- pod scheduled to a node, might be `Running` or `Pending`
- node deleted (see: [Disconnected node](#disconnected-node))

2. **Pending, terminating and unscheduled** (solved in Alpha)

This pod state is characterized by the following:
- unscheduled (the Pod's `.spec.nodeName` is nil), so no Kubelet is assigned
- `Pending` phase
- the Pod's `deletionTimestamp` is set by a DELETE request (terminating)

Example steps leading to the Pod's state:
- unschedulable pod (for example due to unsatisfiable requests)
- DELETE request sent by a user or another k8s component

Note that, the point about the Pod being terminating is important here. As
long as the pod is not terminating it can be scheduled if resources are
increased to satisfy its requests.

3. <b id="3rd_scenario">Pending, terminating, and scheduled</b> (planned for second Beta)

This pod state is characterized by the following:
- the Pod's `.spec.nodeName` is set, but the node no longer exists
- `Pending` phase
- the Pod's `deletionTimestamp` is set by a DELETE request (terminating)

Example steps leading to the Pod's state:
- Pod scheduled to a node, but remains in `Pending` (e.g. due to an invalid
image reference, invalid config map, one of the containers failing to start)
- DELETE request sent by a user or another k8s component

Note that, the point about the Pod being terminating is important here. As long
as the pod is not terminating, it could be a transient issue and Kubelet can
make progress after retrying.

##### Review of example steps for the 3rd scenario

Below we present example steps leading to the [3rd scenario](#3rd_scenario) on k8s 1.26.

**Invalid image reference, pod deleted by a user**

1. create a pod with invalid image reference, example `yaml`:
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: invalid-image
spec:
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: invalid-image
        image: non_existing_image_name:102
        command: ["bash"]
        args: ["-c", 'echo "Hello world"']
  podFailurePolicy: # this is just to prevent considering the pod as failed until in terminal phase
    rules: []
  backoffLimit: 0
```
2. delete the pod with `kubectl delete pods -l job-name=invalid-image`

The relevant fields of the pod:

```
  metadata:
    deletionTimestamp: "2023-02-03T13:48:14Z"
    finalizers:
    - batch.kubernetes.io/job-tracking
  status:
    conditions:
    - status: "True"
      type: Initialized
    - status: "False"
      type: Ready
    - status: "False"
      type: ContainersReady
    - status: "True"
      type: PodScheduled
    containerStatuses:
    - image: non_existing_image_name:102
      state:
        waiting:
          message: Back-off pulling image "non_existing_image_name:102"
          reason: ImagePullBackOff
    phase: Pending
```

The pod is stuck in the Pending and Terminating state. The finalizer
is not deleted by Job controller as it requires the pod to be in terminal
phase when `podFailurePolicy` is defined. The pod remains in the state even
if the image reference is fixed manually.

**Invalid image reference, pod deleted by scheduler**

1. Create a pod similar to the one above, but extend it with requests to
ensure it will be preempted by another pod with a higher priority
2. Create another pod with the critical priority (e.g. `system-node-critical`)
on the same node so that Kube-scheduler preempts the first pod.

The status looks like in the previous example, but contains the
`DisruptionTarget=True` condition.

As above, the pod is stuck in the Pending and Terminating state.

**Invalid configMap reference, pod deleted by a user**

1. create a pod with invalid configMap reference, example `yaml`:
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: invalid-configmap-ref
spec:
  template:
    spec:
      restartPolicy: Never
      volumes:
      - name: volume-name
        configMap:
          name: invalid-config-name-name
      containers:
      - name: invalid-configmap-ref
        image: centos:7
        command: ["bash"]
        args: ["-c", 'echo "Hello world"']
        volumeMounts:
        - mountPath: /script_path
          name: volume-name
  podFailurePolicy:
    rules: []
  backoffLimit: 0
```
2. delete the pod with `kubectl delete pods -l job-name=invalid-configmap-ref`

The relevant fields of the pod:

```
  metadata:
    deletionTimestamp: "2023-02-03T13:48:14Z"
    finalizers:
    - batch.kubernetes.io/job-tracking
  status:
    conditions:
    - status: "True"
      type: Initialized
    - status: "False"
      type: Ready
    - status: "False"
      type: ContainersReady
    - status: "True"
      type: PodScheduled
    containerStatuses:
    - image: centos:7
      state:
        terminated:
          exitCode: 137
          message: The container could not be located when the pod was terminated
          reason: ContainerStatusUnknown
    phase: Pending
```

As above, the pod is stuck in the Pending and Terminating state.

**Correct config, but image takes long to download, pod deleted by a user in the meanwhile**

This scenario is a little bit different, the config is correct, but the
pod is deleted by a user while in the `Pending` phase. In that case, the
pods transition into the `Running` phase and fail soon after. With the proposed
change the transition will happen earlier, thus saving resources.

1. create a pod using a huge image to be in the `Pending` phase for long, example `yaml`:
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: huge-image
spec:
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: huge-image
        image: sagemathinc/cocalc # this is around 20GB
        command: ["bash"]
        args: ["-c", 'sleep 60 && echo "Hello world"']
  podFailurePolicy:
    rules: []
  backoffLimit: 0
```
2. delete the pod with `kubectl delete pods -l job-name=huge-image`

The relevant fields of the pod:

```
  status:
    conditions:
    - status: "True"
      type: Initialized
    - status: "False"
      type: Ready
    - reason: PodFailed
      status: "False"
      type: ContainersReady
    - status: "True"
      type: PodScheduled
    containerStatuses:
    - image: docker.io/sagemathinc/cocalc:latest
      state:
        terminated:
          exitCode: 137
          reason: Error
    phase: Failed
```

Here, the pod is not stuck, however it transitions to `Running` and fails
soon after, making the interim transition to `Running` unnecessary. Also, there
is a race condition, if the container succeeds before the graceful period for
pod termination (if not for the `sleep 60` in the example above) the running pod may complete with the
`Succeeded` status before its containers are killed (and it transitions in the
`Failed` phase). This is already problematic for the Job controller, which might
count the pod as failed, despite the pod eventually succeeding. With the proposed
change, in the scenario, the pod transitions directly from the `Pending` phase
to `Failed`.

##### Proposed solution for the 3rd scenario

We plan to fix the 3rd scenario by
modifying Kubelet to transition such pods (`Pending`, terminating, and scheduled)
into the `Failed` phase, allowing the Job
controller to count them, match against the (optional) pod failure policy and
remove their Job finalizers, allowing API server to complete deletion.

In the final update transitioning the pod to the `Failed` phase Kubelet will
also set the `reason` and the `message` field. We propose the values as
`Deleted` and `Deleted while pending`, respectively, in order to reflect the
direct reason for the pod to transitioned to the `Failed` phase.

As the reasons for deleting a Pod while it is in the `Pending` phase might be
various, it should be up to the controller or a user to add an adequate pod
condition to reflect the reason. An example scenario when Kube-scheduler adds
the `DisruptionTarget` is presented above.

Note that, the transitioning such pods to Failed phase requires an additional
PATCH request from Kubelet to the API server. However, the situation should be
rare as it requires DELETE which needs to be send by a user or another component
for the Pod to be terminating, so it should have a limited impact on performance.
An analogous decision has been made for PodGC.

A prototype implementation is prepared here:
[Mark Pending Terminating pods as Failed by Kubelet](https://github.com/kubernetes/kubernetes/pull/115331).

##### Implementation progress and plan

For Alpha, we implemented a fix for scenarios 1. and 2. by setting the pod phase
as `Failed` in PodGC. Note that, in both scenarios there is no Kubelet to
transition the pod, thus it is done by PodGC.

As the 3rd scenario wasn't fixed in the first iteration of Beta (1.26), we only
require pods to be in terminal phase when `podFailurePolicy` is specified (and
only in that case), but this creates an inconsistency with handling Jobs
with and without pod failure policies
(see Issue: [Job controller should wait for Pods to terminate to match the failure policy](https://github.com/kubernetes/kubernetes/issues/113855)).

For the second iteration of Beta (1.27), we plan to fix the 3rd scenario as
suggested above, see: [Proposed solution for the 3rd scenario](#proposed-solution-for-the-3rd-scenario).

Also, for the second iteration of Beta (1.27) we are going to expand the
feature documentation to explain this change. In particular, we are going to
provide a list of example scenarios impacted by this change, including:
invalid image reference, invalid config map reference.

One release after the `PodDisruptionCondition` feature gate graduates to GA
we plan to simplify Job controller to consider as failed (and count as such)
pods which are in terminal phase regardless of the fact if the `podFailurePolicy`
is specified (see [Deprecation](#deprecation)).

### Risks and Mitigations

#### Garbage collected pods

The Pod status (which includes the `conditions` field and the container exit
codes) could be lost if the failed pod is garbage collected.

Losing Pod's status before it is interpreted by Job Controller can be prevented
by using the feature of [job tracking with finalizers](https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/)
(see more about the design details section: [Interim FailureTarget condition](#interim-failuretarget-condition)).

#### Evolving condition types

The list of available pod condition types field will be evolving with new
values being added and potentially some values becoming obsolete. This can make
it difficult to maintain a valid list of pod condition types enumerated in
the Job configuration.

In order to mitigate this risk we are going to define (along with
documentation) the new condition types as constants in the already existing list
defined in the k8s.io/apis/core/v1 package. Thus, every addition of a new
condition type will require an API review. The constants will allow users of the
package to reduce the risk of typing mistakes.

Additionally, we introduce a generic condition type `DisruptionTarget` which
indicates pod disruption (due to e.g. preemption, API-initiated eviction or
taint-based eviction).

A more detailed information about the condition (containing the kubernetes
component name which initiated the disruption) is conveyed by the `reason` and
`message` fields of the added pod condition.

Finally, we are going to cover the handling of pod failures associated with the
new pod condition types in integration tests.

#### Stale DisruptionTarget condition which is not cleaned up

It is possible that a stale disruption condition
(see: [Failing delete after a condition is added](#failing-delete-after-a-condition-is-added))
is not clean up by the disruption controller before the Pod completes. The stale
Pod condition can misguide the Job controller.

First, the scenario is unlikely as it requires that a Pod deletion call fails
right after the Pod status update succeeded, and that conditions in the cluster
change (for example the `NoExecute` taint is removed or kube-scheduler decides
to preempt another Pod) so that the deletion is not re-attempted. Additionally,
the Pod needs to fail within 2min (before the disruption controller cleans it
up) for a reason which would not result in adding `DisruptionTarget`, so the
stale condition is inaccurate when inspected by the Job controller.

Second, the negative consequence of the scenario is limited. For jobs which
are configured to ignore the disruption errors it results in an unnecessary
Pod retry.

Given the factors above we assess this is an acceptable risk.

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
- TerminationByKubelet (Pod terminated due to graceful node shutdown or node resource pressure).

The already existing `status.conditions` field in Pod will be used by kubernetes
components to append a dedicated condition.

When the failure is initiated by a component which deletes the pod, then the API
call to append the condition will be issued as a pod status update call
before the Pod delete request (not necessarily as the last update request before
the actual delete). For Kubelet, which does not delete the pod itself, the pod
condition is added in the same API request as the phase change to failed.
This way the Job controller will be able to see the condition, and match it
against the pod failure policy, when handling a failed pod.

During the implementation process we are going to review the places where the
pod delete requests are issued to modify the code to also append a meaningful
condition with dedicated `type`, `reason` and `message` fields based on the
invocation context.

We list the reason constants above in order to gain the community consensus on
informative names, but they are purely informational. It is not supported to use the `reason` or `message` fields
when defining a pod failure policy (see `PodFailurePolicyOnPodConditionsPattern`
in [JobSpec API](#jobspec-api)). Supporting of matching by the `reason` field
is a possible extension of the feature which can be implemented once there is
use case to motivate it. However, it may create a risk of breaking compatibility
with evolving set of reasons in use, similar to the risk of
[evolving condition types](#evolving-condition-types).

### Interim FailureTarget Job condition

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
        resources:
          limits:
            memory: "128Mi"
            ephemeral-storage: "1Gi"
      - name: monitoring-job-container
        image: job-monitoring
        command: ["./monitoring"]
        resources:
          limits:
            memory: "128Mi"
            ephemeral-storage: "1Gi"
  backoffLimit: 3
  podFailurePolicy:
    rules:
    - action: FailJob
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
(see: [limiting this feature](#limitting-this-feature)), then the rules using
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
  - a dedicated Pod condition indicating termination originated by a kubernetes component
- adding of the `DisruptionTarget` by Kubelet in case of:
  - eviction due to graceful node shutdown
  - eviction due to node pressure
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

The kubelet packages (with their unit test coverage) which are going to be modified during implementation:
- `k8s.io/kubernetes/pkg/kubelet/nodeshutdown`: `13 Sep 2022` - `74.9%`  <!--(handling of nodeshutdown)-->
- `k8s.io/kubernetes/pkg/kubelet/eviction`: `13 Sep 2022` - `67.7%`  <!--(handling of node-pressure eviction)-->

##### Integration tests

The following scenarios will be covered with integration tests:
- enabling, disabling and re-enabling of the feature gate
- pod failure is triggered by a delete API request along with appending a
  Pod condition indicating termination originated by a kubernetes component
  (we aim to cover all such scenarios)
- pod failure is caused by a failed container with a non-zero exit code

More integration tests might be added to ensure good code coverage based on the
actual implementation.

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
The following scenario are covered with e2e tests:
- [sig-apps#gce](https://testgrid.k8s.io/sig-apps#gce):
  - Job Using a pod failure policy to not count some failures towards the backoffLimit Ignore DisruptionTarget condition
  - Job Using a pod failure policy to not count some failures towards the backoffLimit Ignore exit code 137
  - Job should allow to use the pod failure policy on exit code to fail the job early
  - Job should allow to use the pod failure policy to not count the failure towards the backoffLimit
- [sig-scheduling#gce-serial](https://testgrid.k8s.io/sig-scheduling#gce-serial):
  - SchedulerPreemption [Serial] validates pod disruption condition is added to the preempted pod
- [sig-release-master-informing#gce-cos-master-serial](https://testgrid.k8s.io/sig-release-master-informing#gce-cos-master-serial):
  - NoExecuteTaintManager Single Pod [Serial] pods evicted from tainted nodes have pod disruption condition

The following scenarios are covered with node e2e tests
([sig-node-presubmits#pr-kubelet-gce-e2e-pod-disruption-conditions](https://testgrid.k8s.io/sig-node-presubmits#pr-kubelet-gce-e2e-pod-disruption-conditions) and
[sig-node-presubmits#pr-node-kubelet-serial-containerd](https://testgrid.k8s.io/sig-node-presubmits#pr-node-kubelet-serial-containerd)):
  - GracefulNodeShutdown [Serial] [NodeFeature:GracefulNodeShutdown] [NodeFeature:GracefulNodeShutdownBasedOnPodPriority] graceful node shutdown when PodDisruptionConditions are enabled [NodeFeature:PodDisruptionConditions] should add the DisruptionTarget pod failure condition to the evicted pods
  - PriorityPidEvictionOrdering [Slow] [Serial] [Disruptive][NodeFeature:Eviction] when we run containers that should cause PIDPressure; PodDisruptionConditions enabled [NodeFeature:PodDisruptionConditions] should eventually evict all of the correct pods

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
- implementation of extending the existing Job controller's metrics:
  `job_finished_total` by the `reason` field; and introduction of the
  `pod_failures_handled_by_failure_policy_total` metric with the `action` label
  (see also [here](#how-can-an-operator-determine-if-the-feature-is-in-use-by-workloads))
- implementation of adding pod disruption conditions (`DisruptionTarget`)
  by Kubelet when terminating a Pod (see: [Termination initiated by Kubelet](#termination-initiated-by-kubelet))
- Refactor adding of pod conditions with the use of
  [SSA](https://kubernetes.io/docs/reference/using-api/server-side-apply/) client.
- The feature flag enabled by default

Second iteration:
 - Extend Kubelet to mark as failed pending terminating pods (see: [Marking pods as Failed](#marking-pods-as-failed)).
 - Extend the feature documentation to explain transitioning of pending and
   terminating pods into `Failed` phase.

#### GA

- Address reviews and bug reports from Beta users
- Write a blog post about the feature
- Graduate e2e tests as conformance tests
- Lock the `PodDisruptionConditions` and `JobPodFailurePolicy` feature-gates
- Declare deprecation of the `PodDisruptionConditions` and `JobPodFailurePolicy` feature-gates in documentation

<!--
**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

-->

#### Deprecation

In GA+1 release:
- Modify the code to ignore the `PodDisruptionConditions` and `JobPodFailurePolicy` feature gates
- Simplify the Job controller to wait for pods to terminate before counting them
  as failed or matching against the pod failure policy, regardless if the pod
  failure policy is specified (see: [Job controller should wait for Pods to terminate to match the failure policy](https://github.com/kubernetes/kubernetes/issues/113855)).

In GA+2 release:
- Remove the `PodDisruptionConditions` and `JobPodFailurePolicy` feature gates

### Upgrade / Downgrade Strategy

#### Upgrade

An upgrade to a version which supports this feature should not require any
additional configuration changes. In order to use this feature after an upgrade
users will need to configure their Jobs by specifying `spec.podFailurePolicy`. The
only noticeable difference in behavior, without specifying `spec.podFailurePolicy`,
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
      - kubelet
  - Feature gate name: JobPodFailurePolicy
    - Components depending on the feature gate:
      - kube-apiserver
      - kube-controller-manager
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

Yes. The kubernetes components (kubelet, kube-apiserver, kube-scheduler and
kube-controller-manager) will append a pod condition along with the request pod
delete request.

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

If any component has not yet rolled out, or fails to rollout, the existing
default behavior will continue to apply, but there is no downtime during partial
rollout or rollback.

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

A substantial increase in the `job_sync_duration_seconds` metric may suggest the
processing of the configured job pod failure policy rules consumes too much time.

An operator can also observe `job_pods_finished_total` to check if the reason count
of taken actions (`FailJob`, `Count` or `Ignore`) correlates with the expected
changes based on the Job workload specificity.

Additionally, an operator should check if the terminated pods (due to reasons
listed in [design details](#design-details)) have the appropriate pod condition
added. The addition of the pod conditions can be checked by standard tools such
as the `kubectl describe` command or a watch
`kubectl get pods -o yaml -w --output-watch-events`.

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Manual test performed to simulate the upgrade->downgrade->upgrade scenario:

1. Deploy k8s 1.25 with `PodDisruptionConditions` and `JobPodFailurePolicy` feature gates disabled
1. Enable the `PodDisruptionConditions` and `JobPodFailurePolicy` feature gates for the control plane components
1. Test the scenarios (described in [Handling retriable and non-retriable pod failures with Pod failure policy](https://kubernetes.io/docs/tasks/job/pod-failure-policy/)):
  - Scenario 1:
    - Create a job with container failing with `42` exit code. The job has `backoffLimit>0` and pod failure policy with a `FailJob` rule matching the exit codes.
    - Verify that the job fails fast without retries.
  - Scenario 2:
    - Create a job with a long running containers and `backoffLimit=0`.
    - Verify that the job continues after the node in uncordoned
1. Disable the feature gates. Verify that the above scenarios result in default behavior:
  - In scenario 1: the job restarts pods failed with exit code `42`
  - In scenario 2: the job is failed due to exceeding the `backoffLimit` as the failed pod failed during the draining
1. Re-enable the feature gates
1. Verify the above described scenarios work as after the first enablement

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

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
`PodFailurePolicy`, `BackoffLimitExceeded` and`DeadlineExceeded`.
It can be used to determine what is the relative frequency of job terminations
due to different reasons. For example, if jobs are terminated often due to
`BackoffLimitExceeded` it may suggest that the pod failure policy should be extended
with new rules to terminate jobs early more often
  - `pod_failures_handled_by_failure_policy_total` (new): the `action` label
tracks the number of failed pods that are handled by a specific failure policy
action. Possible values are: `FailJob`, `Ignore`
and `Count`. This metric can be used to assess the coverage of pod failure
scenarios with `spec.podFailurePolicy` rules.

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

- [x] Pod .status
  - Condition type: `DisruptionTarget` when a Pod is terminated due to a reason listed in [design details](#design-details).
- [x] Job .status
  - Condition reason: `PodFailurePolicy` for the job `Failed` condition if the job was terminated due to the matching `FailJob` rule.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- 99% percentile over day for Job syncs is <= 15s for a client-side 50 QPS limit.

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

No.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

No.

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

Yes. A PATCH API call to append a Pod condition when deleting the Pod. Also,
one PATCH API call to set Pod's phase as `Failed` in scenarios (2nd and 3rd)
described under [Marking pods as Failed](#marking-pods-as-failed). Note that,
in the 1st scenario the phase is set along with adding the Pod condition.

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

Yes.

When the feature is enabled, Pods will be added a new Pod condition on termination.
- API type: Pod
- Estimated increase in size: 100B
- No new Pod objects

The size of the new Pod condition we estimate by adding the estimated sizes of the fields (in bytes):
- `type`: 20
- `status`: 5
- `reason`: 30
- `message`: 50
- `lastProbeTime`: 8
- `LastTransitionTime`: 8.

When the feature is enabled users will be able to configure the Job's pod failure policy.
- API type: Job
- Estimated increase in size: 22KB
- No new Job objects

We estimate the size of the new `podFailurePolicy` field in Job as `max_number_of_rules * max(est_onExitCodes_size, est_onPodConditions_size)`, where (in bytes):
- `max_number_of_rules`: 20
- `est_onExitCodes_size`: 1120 (255*4 for exit code values + 100 for `containerName`)
- `est_onPodConditions_size`: 500 (max 20 patterns * (5 for `status` + 20 for `type`)).

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

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This feature does not introduce any resource exhaustive operations.

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

No change from existing behavior of the Job controller.

###### What are other known failure modes?

  - When `PodDisruptionConditions` enabled pods are not terminated based on the NoExecute taint
    - Known bug in 1.25.x (fixed in master)
    - Bugs: [Fix handling of NoExecute taint when PodDisruptionConditions is enabled](https://github.com/kubernetes/kubernetes/pull/112518)
    - Detection: Observe that the pods are not deleted when a node is tainted with `NoExecute`
    - Mitigations: disable `PodDisruptionConditions`
    - Testing: Discovered bugs are covered by unit and integration tests.

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

If pods terminate without the expected pod failure conditions (this part of the
feature does not depend on the Job controller, the standard troubleshooting
technics apply):
- Check reachability between kubernetes components.
- Consider increasing the logging level to trace when the issue occurs.

If a job failure policy isn't respected (this part of the feature depends
on the Job controller, the standard and Job controller-specific troubleshooting
technics apply):
- Inspect manually if the job's terminated pods have containers with exit codes matching the configured rules.
- Inspect manually if the job's terminated pods have conditions expected according to the [design details](#design-details).
- Inspect the Job controller's `job_sync_duration_seconds` metric to see if there
  is an increase of the Job controller processing time.
- Inspect the Job controller's `job_pods_finished_total` metric for the
  to check if the numbers of pod failures handled by specific actions (counted
  by the `failure_policy_action` label) agree with the expectations.
  For example, if a user configures job failure policy with `Ignore` action for
  the `DisruptionTarget` condition, then a node drain is expected to increase
  the metric for `failure_policy_action=Ignore`.
- Consider increasing the logging level of kube-controller-manager to trace the job_controller logs.

## Implementation History

- 2022-06-23: Initial KEP merged
- 2022-07-12: Preparatory PR "Refactor gc_controller to do not use the deletePod stub" merged
- 2022-07-14: Preparatory PR "Refactor taint_manager to do not use getPod and getNode stubs" merged
- 2022-07-20: Preparatory PR "Add integration test for podgc" merged
- 2022-07-28: KEP updates merged
- 2022-08-01: Additional KEP updates merged
- 2022-08-02: PR "Append new pod conditions when deleting pods to indicate the reason for pod deletion" merged
- 2022-08-02: PR "Add worker to clean up stale DisruptionTarget condition" merged
- 2022-08-04: PR "Support handling of pod failures with respect to the configured rules" merged
- 2022-09-09: Bugfix PR for test "Fix the TestRoundTripTypes by adding default to the fuzzer" merged
- 2022-09-26: Prepared PR for KEP Beta update. Summary of the changes:
  - proposal to extend kubelet to add the following pod conditions when evicting a pod (see [Design details](#design-details)):
    - DisruptionTarget for evictions due graceful node shutdown, admission errors, node pressure or Pod admission errors
    - ResourceExhausted for evictions due to OOM killer and exceeding Pod's ephemeral-storage limits
  - extended the review of pod eviction scenarios by kubelet-initiated pod evictions:
    - [Container memory limit exceeded](#container-memory-limit-exceeded)
    - [Container ephemeral-storage limit exceeded](#container-ephemeral-storage-limit-exceeded)
    - [Graceful node shutdown](#graceful-node-shutdown)
    - [Pod admission error](#pod-admission-error)
  - added a Risk and Mitigations sections:
    - [OOM killer invoked when memory limits are not exceeded](#oom-killer-invoked-when-memory-limits-are-not-exceeded)
    - [Stale DisruptionTarget condition which is not cleaned up](#stale-disruptiontarget-condition-which-is-not-cleaned-up)
  - updated names of the proposed metrics fields: `PodFailurePolicyRule` -> `PodFailurePolicy` and `JobTerminated` -> `JobFailed` (see [here](#how-can-an-operator-determine-if-the-feature-is-in-use-by-workloads))
  - added [Story 3](#story-3) to demonstrate how to use the API to ensure there are no infinite Pod retries
  - updated [Graduation Criteria for Beta](#beta)
  - updated of kep.yaml and [PRR questionnaire](#production-readiness-review-questionnaire) to prepare the KEP for Beta
- 2022-10-27: PR "Use SSA to add pod failure conditions" ([link](https://github.com/kubernetes/kubernetes/pull/113304))
- 2022-10-31: PR "Extend metrics with the new labels" ([link](https://github.com/kubernetes/kubernetes/pull/113324))
- 2022-11-03: PR "Fix disruption controller permissions to allow patching pod's status" ([link](https://github.com/kubernetes/kubernetes/pull/113580))
- 2022-11-08: KEP update for Beta ([link](https://github.com/kubernetes/enhancements/pull/3646)) with main changes:
  - do not introduce the `ResourceExhausted` condition (it was planned to be used for pods killed due to OOM killer or exceeding ephemeral storage limits)
  - do not add `DisruptionTarget` condition in case of admission failures
- 2022-11-11: PR "Fix match onExitCodes when Pod is not terminated" ([link](https://github.com/kubernetes/kubernetes/pull/113856))
- 2022-11-11: PR "Wait for Pods to finish before considering Failed in Job" ([link](https://github.com/kubernetes/kubernetes/pull/113860))
- 2022-11-15: PR "Add e2e test to ignore failures with 137 exit code" ([link](https://github.com/kubernetes/kubernetes/pull/113927))
- 2023-01-03: PR "Fix clearing of rate-limiter for the queue of checks for cleaning stale pod disruption conditions" ([link](https://github.com/kubernetes/kubernetes/pull/114770))
- 2023-01-09: PR "Adjust DisruptionTarget condition message to do not include preemptor pod metadata" ([link](https://github.com/kubernetes/kubernetes/pull/114914))
- 2023-01-13: PR "PodGC should not add DisruptionTarget condition for pods which are in terminal phase" ([link](https://github.com/kubernetes/kubernetes/pull/115056))

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

### Possible future extensions

As one possible direction of extending the feature is adding pod failure
conditions in the following scenarios (see links for discussions on the factors
that made us not to cover the scenarios in Beta):
- [active deadline timeout exceeded](#active-deadline-timeout-exceeded)
- [admission failures](#admission-failures)
- [resource limits exceeded](#resource-limits-exceeded).

We are going to re-evaluate the decisions based on the user feedback after users
start to use the feature - using job failure policies based on the `DisruptionTarget` condition
and container exit codes.

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
