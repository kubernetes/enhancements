---
title: Sidecar Containers
authors:
  - "@joseph-irving"
  - "@rata"
owning-sig: sig-apps
participating-sigs:
  - sig-apps
  - sig-node
reviewers:
  - "@fejta"
  - "@sjenning"
  - "@SergeyKanzhelev"
approvers:
  - "@enisoc"
  - "@kow3ns"
  - "@derekwaynecarr"
  - "@dchen1107"
editor: TBD
creation-date: 2018-05-14
last-updated: 2020-06-24
status: provisional
---

# Sidecar Containers

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Prerequisites](#prerequisites)
- [Motivation](#motivation)
  - [Problems: jobs with sidecar containers](#problems-jobs-with-sidecar-containers)
  - [Problems: service mesh, metrics and logging sidecars](#problems-service-mesh-metrics-and-logging-sidecars)
    - [Logging/Metrics sidecar](#loggingmetrics-sidecar)
    - [Service mesh](#service-mesh)
  - [Problems: Coupling infrastructure with applications](#problems-coupling-infrastructure-with-applications)
- [Goals](#goals)
- [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [API Changes:](#api-changes)
    - [Kubelet Changes:](#kubelet-changes)
      - [Shutdown triggering](#shutdown-triggering)
      - [Sidecars terminated last](#sidecars-terminated-last)
      - [Sidecars started first](#sidecars-started-first)
      - [PreStop hooks sent to Sidecars first](#prestop-hooks-sent-to-sidecars-first)
  - [Proposal decisions to discuss](#proposal-decisions-to-discuss)
    - [preStop hooks delivery guarantees are changed](#prestop-hooks-delivery-guarantees-are-changed)
      - [Alternative 1: add a TerminationHook](#alternative-1-add-a-terminationhook)
      - [Alternative 2: Do nothing](#alternative-2-do-nothing)
      - [Suggestion](#suggestion)
    - [Killing pods take 3x the time](#killing-pods-take-3x-the-time)
      - [Why is it important to discuss this?](#why-is-it-important-to-discuss-this)
      - [Alternatives to kill the pod in the expected time](#alternatives-to-kill-the-pod-in-the-expected-time)
    - [How to split the shutdown time to kill different types of containers?](#how-to-split-the-shutdown-time-to-kill-different-types-of-containers)
      - [Alternative 1: Allow any step to consume all and be over <code>GraceTime</code> by 8s](#alternative-1-allow-any-step-to-consume-all-and-be-over--by-8s)
      - [Alternative 2: Allow any step to consume all and skip preStop hooks](#alternative-2-allow-any-step-to-consume-all-and-skip-prestop-hooks)
      - [Alternative 3: Allow any step to consume all and be over <code>GraceTime</code> by 0s](#alternative-3-allow-any-step-to-consume-all-and-be-over--by-0s)
      - [Alternative 4: Allow any step to consume all and use 6s as minimum <code>GraceTime</code>](#alternative-4-allow-any-step-to-consume-all-and-use-6s-as-minimum-)
      - [Alternative 5: Use <em>per container</em> terminations](#alternative-5-use-per-container-terminations)
      - [Suggestion](#suggestion-1)
    - [Currently not handling the case of pod=nil](#currently-not-handling-the-case-of-podnil)
    - [Pods with RestartPolicy Never](#pods-with-restartpolicy-never)
      - [Alternative 1: Add a per container fatalToPod field](#alternative-1-add-a-per-container-fataltopod-field)
      - [Alternative 2: Do nothing](#alternative-2-do-nothing-1)
      - [Alternative 2: Always restart sidecar containers](#alternative-2-always-restart-sidecar-containers)
      - [Suggestion](#suggestion-2)
    - [Enforce the startup/shutdown behavior only on startup/shutdown](#enforce-the-startupshutdown-behavior-only-on-startupshutdown)
    - [Sidecar containers won't be available during initContainers phase](#sidecar-containers-wont-be-available-during-initcontainers-phase)
      - [Suggestion](#suggestion-3)
    - [Revisit if we want to modify the podPhase](#revisit-if-we-want-to-modify-the-podphase)
      - [Alternative 1: don't change the pod phase](#alternative-1-dont-change-the-pod-phase)
      - [Alternative 2: change the pod phase](#alternative-2-change-the-pod-phase)
      - [Suggestion](#suggestion-4)
    - [No container type standard](#no-container-type-standard)
    - [Is this change really worth doing?](#is-this-change-really-worth-doing)
      - [Kubernetes jobs with sidecars](#kubernetes-jobs-with-sidecars)
      - [Service mesh](#service-mesh-1)
      - [Kubernetes pods that run indefinitely with sidecars](#kubernetes-pods-that-run-indefinitely-with-sidecars)
      - [Summary](#summary-1)
    - [Why this design seems extensible?](#why-this-design-seems-extensible)
      - [What if we add pre-defined phases for container startup/shutdown?](#what-if-we-add-pre-defined-phases-for-container-startupshutdown)
      - [What if we add &quot;Depends On&quot; semantics to containers?](#what-if-we-add-depends-on-semantics-to-containers)
  - [Proof of concept implementations](#proof-of-concept-implementations)
    - [KEP implementation and Demo](#kep-implementation-and-demo)
    - [Another implementation using pod annotations](#another-implementation-using-pod-annotations)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Alternative designs considered](#alternative-designs-considered)
    - [Add a pod.spec.SidecarContainers array](#add-a-podspecsidecarcontainers-array)
    - [Mark one container as the Primary Container](#mark-one-container-as-the-primary-container)
    - [Boolean flag on container, Sidecar: true](#boolean-flag-on-container-sidecar-true)
    - [Mark containers whose termination kills the pod, terminationFatalToPod: true](#mark-containers-whose-termination-kills-the-pod-terminationfataltopod-true)
    - [Add &quot;Depends On&quot; semantics to containers](#add-depends-on-semantics-to-containers)
    - [Pre-defined phases for container startup/shutdown or arbitrary numbers for ordering](#pre-defined-phases-for-container-startupshutdown-or-arbitrary-numbers-for-ordering)
  - [Workarounds sidecars need to do today](#workarounds-sidecars-need-to-do-today)
    - [Jobs with sidecar containers](#jobs-with-sidecar-containers)
    - [Service mesh or metrics sidecars](#service-mesh-or-metrics-sidecars)
      - [Istio bug report](#istio-bug-report)
    - [Move containers out of the pod](#move-containers-out-of-the-pod)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

This KEP adds the concept of sidecar containers to Kubernetes. This KEP proposes
to add a `lifecycle.type` field to the `container` object in the `pod.spec` to
define if a container is a sidecar container. The only valid value for now is
`sidecar`, but other values can be added in the future if needed.

Pods with sidecar containers only change the behaviour of the startup and
shutdown sequence of a pod: sidecar containers are started before non-sidecars
and stopped after non-sidecars.

A pod that has sidecar containers guarantees that non-sidecar containers are
started only after all sidecar containers are started and are in a ready state.
Furthermore, we propose to treat sidecar containers as regular (non-sidecar)
containers as much as possible all over the code, except for the mentioned
special startup and shutdown behaviour. The rest of the pod lifecycle (regarding
restarts, etc.) remains unchanged, this KEP aims to modify only the startup and
shutdown behaviour.

If a pod doesn't have a sidecar container, the behaviour is completely unchanged
by this proposal.

## Prerequisites

On June 23 2020, during SIG-node meeting, it was decided that this KEP has a
prerequisite on the (not yet submitted KEP) kubelet node graceful shutdown.

As of writing, when a node is shutdown the kubelet doesn't gracefully shutdown
all of the containers (running preStop hooks and other guarantees users would
expect). It was then decided that adding more guarantees to pod shutdown
behavior (as this KEP proposes) depends on having the kubelet gracefully
shutdown first. The main reason for this is to avoid users relying on something
we can't guarantee (like the pod shutdown sequence in case the node shutdowns).

Also, authors of this KEP and the (yet not submitted) KEP for node graceful
shutdown have met several times and are in sync regarding these features
interoperability.

The details about this dependency is explained in the [graduation criteria
section](#graduation-criteria).

## Motivation

The concept of sidecar containers has been around since the early days of
Kubernetes. A clear example is [this Kubernetes blog post][sidecar-blog-post]
from 2015 mentioning the sidecar pattern.

Over the years the sidecar pattern has become more common in applications,
gained popularity and the uses cases are getting more diverse. The current
Kubernetes primitives handled that well, but they are starting to fall short for
several use cases and force weird work-arounds in the applications.

This proposal aims to remediate this by adding a simple set of guarantees for
sidecar containers, while trying to avoid doing a complete re-implementation of an
init system. These alternatives are interesting and were considered,
but in the end the community decided to go for something simpler that will cover
most of the use cases. These options are explored in the
[alternatives](#alternatives) section.

The next section expands on what the current problems are. But, to give more
context, it is important to highlight that some companies are already using a
fork of Kubernetes with this sidecar functionality added (not all
implementations are the same, but more than one company has a fork for this).

[sidecar-blog-post]: https://kubernetes.io/blog/2015/06/the-distributed-system-toolkit-patterns/#example-1-sidecar-containers

### Problems: jobs with sidecar containers

Imagine you have a Job with two containers: one which does the main processing
of the job and the other is just a sidecar facilitating it. This sidecar could be
a service mesh, a metrics gathering statsd server, etc.

When the main processing finishes, the pod won't terminate until the sidecar
container finishes too. This is problematic for sidecar containers that run
continuously.

There is no simple way to handle this on Kubernetes today. There are
work-arounds for this problem, most of them consist of some form of coupling
between the containers to add some logic where a container that finishes
communicates it so other containers can react. But it gets tricky when you have
more than one sidecar container or want to auto-inject sidecars. Some
alternatives to achieve this currently and their pain points are discussed in
detail on the [alternatives](#alternatives) section.

### Problems: service mesh, metrics and logging sidecars

While this problem is generic to any sidecar container that might need to start
before others or stop after others, in these examples we will use a service
mesh, metrics gathering and logging sidecar.

#### Logging/Metrics sidecar

A logging sidecar should start before several other containers, to not lose logs
from the startup of other applications. Let's call _main container_ the app that
will log and _logging container_ the sidecar that will facilitate it.

If the logging container starts after the main app, some logs can be lost.
Furthermore, if the logging container is not yet started and the main app
crashes on startup, those logs can be lost (depends if logs go to a shared volume
or over the network on localhost, etc.). While you can modify your application
to handle this scenario during startup (as it is probably the change you need to
do to handle sidecar crashes), for shutdown this approach won't work.

On shutdown the ordering behaviour is arguably more important: if the logging
container is stopped first, logs for other containers are lost. No matter if
those containers queue them and retry to send them to the logging container, or
if they are persisted to a shared volume. The logging container is already
killed and will not be restarted, as the pod is shutting down. In these cases,
logs are lost.

The same things regarding startup and shutdown apply for a metrics container.

Some work-arounds can be done, to alleviate the symptoms:
 * Ignore SIGTERM on the sidecar container, so it is alive until the pod is
   killed. This is not ideal and _greatly_ increases the time a pod will take to
terminate. For example, if the P99 response time is 2 minutes and therefore the
TerminationGracePeriodSeconds is set to that, the main container can finish in 2
seconds (that might be the average) but ignoring SIGTERM in the sidecar
container will force the pod to live for 2 minutes anyways.
 * Use preStop hooks that just runs a "sleep X" seconds. This is very similar to
   the previous item and has the same problems.

#### Service mesh

Service mesh presents a similar problem: you want the service mesh container to
be running and ready before other containers start, so that any inbound/outbound
connections that a container can initiate goes through the service mesh.

A similar problem happens for shutdown: if the service mesh container is
terminated prior to the other containers, outgoing traffic from other apps will be
blackholed or not use the service mesh.

However, as none of these are possible to guarantee, most service meshes (like
Linkerd and Istio), need to do several hacks to have the basic
functionality. These are explained in detail in the
[alternatives](#alternatives) section. Nonetheless, here is a  quick highlight
of some of the things some service mesh currently need to do:

 * Recommend users to delay starting their apps by using a script to wait for
   the service mesh to be ready. The goal of a service mesh to augment the app
   functionality without modifying it, that goal is lost in this case.
 * To guarantee that traffic goes via the services mesh, an initContainer is
   added to blackhole traffic until the service mesh containers are up. This way,
   other containers that might be started before the service mesh container can't
   use the network until the service mesh container is started and ready. A side
   effect is that traffic is blackholed until the service mesh is up and in a
   ready state.
 * Use preStop hooks with a "sleep infinity" to make sure the service mesh
   doesn't terminate before other containers that might be serving requests.

The auto-inject of initContainer [has caused bugs][linkerd-bug-init-last], as it
competes with other tools auto-injecting a container to be run last too.

[linkerd-bug-init-last]: https://github.com/linkerd/linkerd2/issues/4758#issuecomment-658457737

### Problems: Coupling infrastructure with applications

The limitations due to the lack of ordering guarantees on startup and shutdown,
can sometimes force the coupling of application code with infrastructure. This
causes a dependency between applications and infrastructure, and forces more
coordination between multiple, possibly independent, teams.

An example in the open source world about this is the [Istio CNI
plugin][istio-cni-plugin]. This was created as an alternative for the
initContainer hack that the service mesh needs to do. The initContainer will
blackhole all traffic until the Istio container is up. The CNI plugin can be
used to completely remove the need for such an initContainer. But this
alternative requires that nodes have the CNI plugin installed, effectively
coupling the service mesh app with the infrastructure.

This KEP removes the need for a service mesh to use either an initContainer or a
CNI plugin: just guarantee that the sidecar container can be started first.

While in this specific example the CNI plugin has some benefits too (removes the
need for some capabilities in the pod) and might be worth pursuing, it is used
here as an example to show thee possibility of coupling apps with
infrastructure. Similar examples also exist in-house for non-open source
applications too.

[istio-cni-plugin]: https://istio.io/latest/docs/setup/additional-setup/cni/

## Goals

This proposal aims to:
 * Allow Kubernetes Jobs to have sidecar containers that run continuously
   without any coupling between containers in the pod.
 * Allow pods to start a subset of containers first and, only when those are
   ready, start the remaining containers.
 * Allow pods to stop a subset of containers first and, only when those have
   been stopped, stop the rest.
 * Not change the semantics in any way for pods not using this feature.
 * Only change the semantics of startup/shutdown behaviour for pods using this
   feature.

Solve issues so that they don't require application modification:
* [25908](https://github.com/kubernetes/kubernetes/issues/25908) - Job completion
* [65502](https://github.com/kubernetes/kubernetes/issues/65502) - Container startup dependencies. Istio bug report

## Non-Goals

This proposal doesn't aim to:
 * Allow a multi-level ordering of containers start/stop sequence. Only two
   types are defined: "sidecar" or "standard" containers. It is not a goal to
   have more than these two types.
 * Model startup/shutdown dependencies between different sidecar containers
 * Change the pod phase in any way, with or without this feature enabled
 * Reimplement an init-system alike semantics for pod containers
   startup/shutdown
 * Allow sidecar containers to run concurrently with initContainers

## Proposal

Create a way to define containers as sidecars, this will be an additional field to the `container.lifecycle` spec: `Type` which can be either `Standard` (default) or `Sidecar`.

e.g:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp-pod
  labels:
    app: myapp
spec:
  containers:
  - name: myapp
    image: myapp
    command: ['do something']
  - name: sidecar
    image: sidecar-image
    lifecycle:
      type: Sidecar
    command: ["do something to help my app"]

```
Sidecars will be started before normal containers but after init, so that they are ready before your main processes start.

This will change the Pod startup to look like this:
* Init containers start
* Init containers finish
* Sidecars start (all in parallel)
* Sidecars become ready
* Non-sidecar containers start (all in parallel)

During pod termination sidecars will be terminated last:
* Non-sidecar containers sent SIGTERM
* Once all non-sidecar containers have exited: Sidecar container are sent a SIGTERM

Containers and Sidecar will share the TerminationGracePeriod. If Containers don't exit before the end of the TerminationGracePeriod then they will be sent a SIGKIll as normal, Sidecars will then be sent a SIGTERM with a short grace period of 2 Seconds to give them a chance to cleanly exit.

PreStop Hooks will be sent to sidecars before containers are terminated.
This will be useful in scenarios such as when your sidecar is a proxy so that it knows to no longer accept inbound requests but can continue to allow outbound ones until the the primary containers have shut down.

To solve the problem of Jobs that don't complete: When RestartPolicy!=Always if all normal containers have reached a terminal state (Succeeded for restartPolicy=OnFailure, or Succeeded/Failed for restartPolicy=Never), then all sidecar containers will be sent a SIGTERM.

PodPhase will be modified to not include Sidecars in its calculations, this is so that if a sidecar exits in failure it does not mark the pod as `Failed`. It also avoids the scenario in which a Pod has RestartPolicy `OnFailure`, if the containers all successfully complete, when the sidecar gets sent the shut down signal if it exits with a non-zero code the Pod phase would be calculated as `Running` despite all containers having exited permanently.

Sidecars are just normal containers in almost all respects, they have all the same attributes, they are included in pod state, obey pod restart policy etc. The only differences are lifecycle related.

### Implementation Details/Notes/Constraints

The proposal can broken down into four key pieces of implementation that all relatively separate from one another:

* Shutdown triggering for sidecars when RestartPolicy!=Always
* Pre-stop hooks sent to sidecars before non sidecar containers
* Sidecars are terminated after normal containers
* Sidecars start before normal containers

#### API Changes:
As this is a change to the Container spec we will be using feature gating, you will be required to explicitly enable this feature on the api server as recommended [here](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#adding-unstable-features-to-stable-versions).

New field `Type` will be added to the lifecycle struct:

```go
type Lifecycle struct {
  // Type
  // One of Standard, Sidecar.
  // Defaults to Standard
  // +optional
  Type LifecycleType `json:"type,omitempty" protobuf:"bytes,3,opt,name=type,casttype=LifecycleType"`
}
```
New type `LifecycleType` will be added with two constants:
```go
// LifecycleType describes the lifecycle behaviour of the container
type LifecycleType string

const (
  // LifecycleTypeStandard is the default container lifecycle behaviour
  LifecycleTypeStandard LifecycleType = "Standard"
  // LifecycleTypeSidecar means that the container will start up before standard containers and be terminated after
  LifecycleTypeSidecar LifecycleType = "Sidecar"
)
```
Note that currently the `lifecycle` struct is only used for `preStop` and `postStop` so we will need to change its description to reflect the expansion of its uses.

#### Kubelet Changes:
Broad outline of what places could be modified to implement desired behaviour:

##### Shutdown triggering
Package `kuberuntime`

Modify `kuberuntime_manager.go`, function `computePodActions`. Have a check in this function that will see if all the non-sidecars had permanently exited, if true: return all the running sidecars in `ContainersToKill`. These containers will then be killed via the `killContainer` function which sends preStop hooks, sig-terms and obeys grace period, thus giving the sidecars a chance to gracefully terminate.

##### Sidecars terminated last
Package `kuberuntime`

Modify `kuberuntime_container.go`, function `killContainersWithSyncResult`. Break up the looping over containers so that it goes through killing the non-sidecars before terminating the sidecars.
Note that the containers in this function are `kubecontainer.Container` instead of `v1.Container` so we would need to cross reference with the `v1.Pod` to check if they are sidecars or not. This Pod can be `nil` but only if it's not running, in which case we're not worried about ordering.

##### Sidecars started first
Package `kuberuntime`

Modify `kuberuntime_manager.go`, function `computePodActions`. If pods has sidecars it will return these first in `ContainersToStart`, until they are all ready it will not return the non-sidecars. Readiness changes do not normally trigger a pod sync, so to avoid waiting for the Kubelet's `SyncFrequency` (default 1 minute) we can modify `HandlePodReconcile` in the `kubelet.go` to trigger a sync when the sidecars first become ready (ie only during startup).

##### PreStop hooks sent to Sidecars first
Package `kuberuntime`

Modify `kuberuntime_container.go`, function `killContainersWithSyncResult`. Loop over sidecars and execute `executePreStopHook` on them before moving on to terminating the containers. This approach would assume that PreStop Hooks are idempotent as the sidecars would get sent the PreStop hook again when they are terminated.

### Proposal decisions to discuss

This section expands on some decisions or side effects of this proposal that
were identified by Rodrigo Campos (@rata) and discussed during the SIG-node
meeting on June 23. The conclusion was to open a PR with those things, so this
section is about them, with some alternatives and suggestions to open up the
discussion.

This doesn't mean in any way that discussion about other things is not welcome.
They are indeed very much welcome.

Bear in mind that these edge cases discussed here are either present in the KEP
design or the KEP design was not clear enough and they are present in the
[current open PR](https://github.com/kubernetes/kubernetes/pull/80744).

#### preStop hooks delivery guarantees are changed

Kubernetes currently [tries to deliver preStop hooks only once][hook-delivery],
although in some cases they can be called more than once. [The current
proposal](#prestop-hooks-sent-to-sidecars-first), however, guarantees that when
sidecar containers are used preStop hooks will be delivered _at least twice_ for
sidecar containers.

As explained [here](#proposal) the reason this is done is because (the following
is just c&p from the relevant paragraphs in the proposal):

> PreStop Hooks will be sent to sidecars before containers are terminated. This will be useful in scenarios such as when your sidecar is a proxy so that it knows to no longer accept inbound requests but can continue to allow outbound ones until the the primary containers have shut down.

In other words, the shutdown sequence proposed is:

1. Run preStop hooks for sidecars
1. Run preStop hooks for non-sidecars
1. Send SIGTERM for non-sidecars
1. Run preStop hooks for sidecars
1. Send SIGTERM for sidecars

For brevity, what happens when some step timeouts and needs to send a SIGKILL is
omitted in the above sequence, as it is not relevant for this point.

The concerns we see with this are that this changes the [current delivery
guarantees][hook-delivery] from _at least once_ to _at least twice_ for _some_
containers (for sidecar containers). Furthermore, before this patch preStop
hooks were delivered only once most of the time. After this patch is delivered
twice for most cases (using sidecars). The problem is not if this is idempotent
or not (as it should be in any case), but chainging the most common case from
delivering preStop hook once to twice.

Another concern is that in the future a different container type might be added,
and these semantics seems difficult to extend. For example, let's say a
container type "OrderedPhases" with the semantics of the [explained
alternative][phase-startup] is added in the future. When would the preStop hooks
be executed now that there are several phases? At the beginning? If there are 5
phases: 0, 1, 2, 3 and 4, how should the shutdown behaviour be? Run preStop
hooks for containers marked in phase 0, then kill containers in phase 4, then
preStop hooks for containers in phase 1, then kill containers in phase 3, etc.?
Or only the preStop hooks for container 0 are called? Why?

It seems confusing, there doesn't seem to be a clear answer on those cases nor
what the use case would be for such semantics.

[hook-delivery]: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#hook-delivery-guarantees
[phase-startup]: #pre-defined-phases-for-container-startupshutdown-or-arbitrary-numbers-for-ordering

##### Alternative 1: add a TerminationHook

Add to containers a `TerminationHook` field. It will accept the same values as
preStop hooks and will be called for any container that defines it when the pod
is switching to a terminating state. These clear semantics seem easy to extend in
the future.

Then, instead of running preStop hooks twice for sidecar containers as it is now
proposed, preStop hooks are run only once: just before stopping the
corresponding containers.

The shutdown sequence steps in this case would be:
1. Run TerminationHook for _any_ container that defines it (sidecar or not)
1. Run preStop hooks for non-sidecars
1. Send SIGTERM for non-sidecars
1. Run preStop hooks for sidecars
1. Send SIGTERM for sidecars

If containers want to take action when a pod is switching to terminating state,
they should use the TerminationHook.

The motivation for running the preStop hooks two times, then, can be implemented
by adding the TerminationHook field to the service mesh container to drain
connections.

Furthermore, if a container type "OrderedPhases" with the semantics of the [explained
alternative][phase-startup] is added in the future, the semantics are still
clear: the TerminationHook will be called for _any_ container (irrespective of
their phase) when the pod switches to a terminating state.

##### Alternative 2: Do nothing

Do not call preStop hooks for sidecars twice, just remove the first call this
section discusses.

Then, suggest users that need to know when the pod changes to a terminating state to
integrate with the Kubernetes API and add a watcher for that.

This might be inconvenient for several users and further conversation is needed
to see if this is feasible. We are aware of some users that this will cause
issues and will probably need to patch Kubernetes because of this. This could
also result in a large increase in calls to the Kubernetes API server causing
scalability issues.

##### Suggestion

Aim to use alternative 1, while in parallel we collect more feedback from the
community.

It will be nice to do nothing (alternative 2), but probably the first step
calling preStop hooks was added for a reason that couldn't be solved by
integrating with the Kubernetes API. Therefore, while we double check this,
seems safe to go with alternative 1.

#### Killing pods take 3x the time

In the design, sidecars and non-sidecars containers are supposed to share the
termination grace period. However, in [the implementation
PR](https://github.com/kubernetes/kubernetes/pull/80744) it takes up to ~3x the
length. We think it is nice to discuss it here as the solution might need some
agreements on how to improve this.

Let's see why this happens first, and then discuss the possible fixes.

The KEP currently proposes the following shutdown sequence steps (same as explained
before, with a little more detail on what is blocking or not now):

1. All sidecar containers preStop hook are executed (blocking) until they finish
1. All non-sidecar containers preStop hook are executed (blocking) until they finish
1. All non-sidecar containers are sent SIGTERM. If they don’t finish during the
   time remaining for this step (more information below), SIGKILL is sent
   (Idem step 1).
1. All sidecar containers preStop hook are executed (blocking) until
   they finished or are interrupted if exceed pod.TerminationGracePeriodSeconds
1. All sidecar containers are sent SIGTERM. If they don’t finish during the time
   remaining for this step (more information below), SIGKILL is sent

In the current implementation PR, steps 1, 2 and 4 should complete within
`pod.TerminationGracePeriodSeconds` each.  If they don't, the shutdown sequence
continues with the next step. Please note that each of the mentioned steps has
this maximum time to execute the hooks: `pod.TerminationGracePeriodSeconds`. In
other words, elapsed time is ignored during preStop hook execution.

Steps 3 and 5, on the other hand, do take into account the elapsed time and
should complete within:
* The remaining time: `pod.TerminationGracePeriodSeconds - <elapsed time in all
previous steps>` seconds if this value is >= 2
([`minimumGracePeriodInSeconds`][min-gracePeriodInSeconds])
* 2 seconds if the previous value is < 2

A side effect of this behaviour is that killing a pod with sidecar containers
can take in the worst case, approximately: `pod.TerminationGracePeriodSeconds *
3 + 2 * 2` seconds, compared to `pod.TerminationGracePeriodSeconds + 2` seconds
without sidecar containers.

This behaviour is because in the PR implementation the `gracePeriodOverride` is
being used when [calling `killContainer()`][kill-container-param] to specify the
time left to kill the containers. However, in `KillContainer()` the preStop
hooks are run [here][kill-container-preStop] using the `gracePeriod` variable
defined [here][kill-container-gracePeriod], which ignores the
`gracePeriodOverride` param for the `killContainer()` function.  That parameter
is used later, as the time to wait since SIGTERM is sent until a SIGKILL will be
sent.

##### Why is it important to discuss this?

The most important reason is that this KEP needs to interoperate well with the
kubelet shutdown KEP (KEP not yet created). We have been working together with
the team working on that KEP and one of the scenarios to handle is shutting down
a node in time constrained environments (preempt VMs in GCE, spot instances in
AWS, etc.).

As far as we coordinated, the Kubelet will kill the pods and the field
`pod.DeletionGracePeriodSeconds` will be set for the time each pod has to
be properly killed. If we add sidecars and we can't kill pods in a way that respect
that time, this feature will not work on node shutdown. The whole point of this
KEP having a [prerequisite](#prerequisites) on that KEP is to avoid that from
happening.

##### Alternatives to kill the pod in the expected time

The behaviour of `killContainer()` ignoring `gracePeriodOverride` for preStop
hooks was discussed in [this issue][issue-gracePeriodOverride].

There, [we posted some observations][issue-gracePeriodOverride-comment] in the
issue:

1. `gracePeriodOverride` variable is not set when running command like `kubectl delete --grace-period 1 pod <pod-name>`.
   Intuitively, we thought that was the case when the variable was set.
   However, when running such a command, `pod.DeletionGracePeriodSeconds` is set instead.
   More details in the link to the issue with this.

1. Changing `killContainer()` to respect `gracePeriodOverride` parameter, as the
   issue suggest, might be backwards incompatible. At least, if there are still
   users that call the function with that param set (is a pointer, but it seems to be nil
   in most cases).

1. If we can't change `killContainer()` to respect `gracePeriodOverride` for
   preStop hooks too, we should consider another way. One option is to update
   `pod.DeletionGracePeriodSeconds` with the elapsed time, as that is used by
   `killContainer()` for the preStop hooks.

The documentation for
[`pod.DeletionGracePeriodSeconds`][doc-DeletionGracePeriodSeconds] says this
field can only be shortened. So, while I'm not really sure if this is okay to
change that field, one option is to use that field for the elapsed time instead
of the `gracePeriodOverride` parameter. As mentioned, this field is already
taken into account within `killContainer()` to run preStop hooks, etc.

But even if using `pod.DeletionGracePeriodSeconds` is possible, the current code
assumes that it is possible that this field is not always set on deletion (for example
[here][DeletionGracePeriodNotSet]). So, if indeed is not always set, we would need
to set it (in conjunction to `pod.DeletionTimestamp` as the doc says) and I'd
like some confirmation from the community to see if this makes sense.

I'd also like to open the discussion about other options that might be better to
handle this. Please let us know what you think :)

[kill-container-param]: https://github.com/kubernetes/kubernetes/pull/80744/files#diff-44f60dd6d99cb695e5f333647ebd0703R727
[kill-container-preStop]: https://github.com/kubernetes/kubernetes/blob/75b555241578ac60cbdef21e01604f2bba8d040d/pkg/kubelet/kuberuntime/kuberuntime_container.go#L624
[kill-container-gracePeriod]: https://github.com/kubernetes/kubernetes/blob/75b555241578ac60cbdef21e01604f2bba8d040d/pkg/kubelet/kuberuntime/kuberuntime_container.go#L604-L610
[min-gracePeriodInSeconds]: https://github.com/kubernetes/kubernetes/blob/e2d8f6c278011b2eabf5754c3274bc406731933c/pkg/kubelet/kuberuntime/kuberuntime_manager.go#L61-L62
[issue-gracePeriodOverride]: https://github.com/kubernetes/kubernetes/issues/92432
[issue-gracePeriodOverride-comment]: https://github.com/kubernetes/kubernetes/issues/92432#issuecomment-648259349
[doc-DeletionGracePeriodSeconds]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#objectmeta-v1-meta
[DeletionGracePeriodNotSet]: https://github.com/kubernetes/kubernetes/blob/e63fb9a597bfbf6f3d454489e4fb49b40ad8c48f/pkg/kubelet/kuberuntime/kuberuntime_container.go#L604-L610

#### How to split the shutdown time to kill different types of containers?

The previous section is about how to, at the implementation level, respect a
given time when killing a pod. This section, in contrast, is about how to split
the available time between the different steps in the shutdown sequence.

Something worth mentioning is that the time to terminate a pod is defined _per
pod_ and not _per container_. Therefore, any termination sequence will have to
deal with how to split that time in the different steps, now that all containers
are not stopped in parallel.

To avoid confusion, the steps of the shutdown sequence are:
1. Run TerminationHook for _any_ container that defines it (sidecar or not)
1. Run preStop hooks for non-sidecars
1. Send SIGTERM for non-sidecars
1. Run preStop hooks for sidecars
1. Send SIGTERM for sidecars

It's important to note that:
 * Pods without sidecar containers terminate in about
   `TerminationGracePeriodSeconds + 2` seconds in the worst case

This is because if the preStop hooks use all the time available, 2 seconds is
used as a [minimum grace period][min-gracePeriodInSeconds] to kill the container
by default (to avoid SIGKILLs).

If the node is running on a preempt VM or spot instance, the time to terminate
a pod might be shorter than `TerminationGracePeriodSeconds`. In that case, when
the kubelet graceful shutdown (not yet submitted) KEP is implemented, the
kubelet will set that time as pod.DeletionGracePeriodSeconds and that will be
used here. For simplicity, instead of naming the two options, we will only use
`GraceTime` from now on to mean: if pod.DeletionGracePeriodSeconds is set, then
`GraceTime` is that and if not set it is `TerminationGracePeriodSeconds`. This
logic is what is [already used in Kubernetes today][k8s-grace-time-precedence].

We can see the following alternatives to handles this, while trying to respect
the time to kill a pod: `GraceTime + c` (c being a constant, sometimes 0). As
pods without sidecars use `GraceTime + 2`, aiming to something close to that (or
lower) seems good.

Bear in mind that to interoperate correctly with the kubelet graceful shutdown
(not yet submitted) KEP we should respect the time given to kill the pod.

Also, take into account that this deadline currently takes into account only the
time the kubelet will wait from sending SIGTERM till it needs to send a SIGKILL,
or the time it will wait for prestop hooks execution. But it completely ignores
the time spent in go code that needs to be executed before/after each step.

[k8s-grace-time-precedence]: https://github.com/kubernetes/kubernetes/blob/5a50c5c95fa2e16e9655ab3db2a3e4255aa87e25/pkg/kubelet/kuberuntime/kuberuntime_container.go#L604-L610

##### Alternative 1: Allow any step to consume all and be over `GraceTime` by 8s

This alternative allows any step in the shutdown sequence to consume all the
`GraceTime`. In general, any step will have to execute within:
 * Remaining time: `GraceTime - <elapsed time in the previous steps>` if this
   value is >= 2.
 * Minimum time: 2 seconds if the previous value is <2.

This option can will take, in the worst case, `GraceTime + 8` seconds. The
reason is simple: there are 5 steps, the first can consume `GraceTime` and the
rest can consume 2 seconds each (`2*4`).

##### Alternative 2: Allow any step to consume all and skip preStop hooks

This alternative allows any step in the shutdown sequence to consume all the
`GraceTime`, but preStop hooks are skipped if we run out of time. In general,
any step will have to execute within:
 * Remaining time: `GraceTime - <elapsed time in the previous steps>` if this
   value is >= 2.
 * preStop hooks will be skipped if the remaining time is <2
 * 2 seconds for containers (SIGTERM) if the remaining time <2

This option can will take, in the worst case, `GraceTime + 4` seconds. The
explanation is similar to the previous case: there are 5 steps, the first can
consume `GraceTime`, preStop hooks are skipped and 2 seconds is used in each
step that sends SIGTERM (`2*2`).

Skipping the preStop hooks is weird, but this option is listed as this is what
[currently happens][k8s-prestop-skipped] if you use something like `kubectl delete pod --grace-time 0
<pod>`. In other words, today when running that command the preStop hooks are
ignored and 2s is used to wait for the container to finish when sending SIGTERM.

[k8s-prestop-skipped]: https://github.com/kubernetes/kubernetes/blob/5a50c5c95fa2e16e9655ab3db2a3e4255aa87e25/pkg/kubelet/kuberuntime/kuberuntime_container.go#L622-L623

##### Alternative 3: Allow any step to consume all and be over `GraceTime` by 0s

This alternative allows any step in the shutdown sequence to consume all the
`GraceTime`, but if we run out of time, the next steps have 0s to execute. In
general, any step will have to execute within:
 * Remaining time: `GraceTime - <elapsed time in the previous steps>`
 * 0s if the remaining time is 0.

Running a step with 0s means: preStop hooks will be skipped and containers will
be just sent SIGKILL. Therefore, this alternative will take, in the worst case,
`GraceTime` to execute.

Skipping the preStop hooks when the gracePeriod is 0, as mentioned in the
previous alternative, is what is currently done.

This alternative is also weird in the following way: a non-sidecar container can
make sidecars never execute. The goal of using sidecar containers to have some
time to execute after other containers are killed is not achieved, in the end.

##### Alternative 4: Allow any step to consume all and use 6s as minimum `GraceTime`

A different alternative is to force `GraceTime` to be more than 6s to allow all
steps to have a 2s minimum and still not take more than `GraceTime + 2` overall.
This is the same time that can take to kill a pod without sidecar containers.

Each step in the shutdown sequence has to execute within:
 * Remaining time: `GraceTime - 6 - <elapsed time in previous steps>` if this
   value is >2.
 * 2 seconds if the remaining time is <2.

This alternative can take `GraceTime + 2` in the worst case to kill a pod. This
is because the first step can take `GraceTime - 6` and the following 4 steps can
take 2s each.

In this case, it can be validated that TerminationGracePeriodSeconds is >=6 for
pods with sidecar containers and be properly documented.

The minimum of 6s can be used when the pod is killed, but handling correctly
when the user wants to override this (being documented that might cause SIGKILL,
etc.) on commands like: `kubectl delete pod --grace-period 0`.

We will need to gather feedback from users, though, to know if this restriction
is enough for all use cases.

##### Alternative 5: Use _per container_ terminations

Adding _per container_ terminations _can_ create tricky semantics when combined
with the _per pod_ termination.

The first thing to note is that any defined _per container_ termination time will
not be guaranteed in case of spot/preempt VMs, as the time to kill the pod might
be shorter than the one configured. In that case, what is the correct way to
handle it is unclear. One option is to calculate the weight or proportional time
each container has and use that for each.

The second thing to note about creating _per container_ termination time is that
it can be redundant with the _per pod_ setting. For example, if a _per
container_ TerminationGracePeriod is set for _all_ containers in a pod, the
pod.TerminationGracePeriod setting doesn't add any information: it can be
calculated by those _per container_ settings.

Alternatives to combine a _per container_ termination with _per pod_ are not
very nice either. Some options to combine them are: allow only the _per pod_ or
_per container_ setting to be set but never both at the same time; allow
pod.TerminationGracePeriod to be set if some container doesn't have the
equivalent _per container_ and calculate how much time is left to run those.

In the latter case, it can be very confusing as some combinations of
`pod.TerminationGracePeriod` and the _per container_ settings are not compatible
as will create negative timeouts for some steps in the shutdown sequences.
This needs to be validated at the admission time and calculations are not that
obvious for users to be doing (I wouldn't like to do that, at least).

If a _per container_ setting is needed, we consider the first option the
simplest way forward.

##### Suggestion

Use alternative 4. Guarantees to terminate in the same time that pods without
sidecar containers terminates and adds a simple validation on pod admission for
pods with sidecar containers.

The rest of the alternatives either take more time, require SIGKILL (that
defeats the purpose of sidecar containers finishing after) or are way more
complicated (like alternative 5).

Having a minimum time of 6s for pods with sidecar containers, as alternative 4
proposes, is a guarantee that might be difficult to change in the future (after
GA). However, it seems the cleanest and simplest way to move forward now.

Alternative 4 seems fine for the Alpha stage and we can gather feedback from
users. Ideas and opinions are _very_ welcome, though.

#### Currently not handling the case of pod=nil

This issue is not a design issue only, but the design doesn't mention what to
modify for this and [the implementation
PR](https://github.com/kubernetes/kubernetes/pull/80744) doesn't handle this
case.

The kubelet tries hard to handle the case where the pod being deleted is nil.
This can happen in some race scenarios (and it seems now is less likely to
happen, maybe with static pod) where the kubelet is restarted while the pod is
being terminated and when the kubelet starts again, the pod is deleted from the
API server. In that case the kubelet can't get the pod from the API server (so
pod is nil) but it needs to kill the running containers.

The current open PR doesn't handle this case. This means the sidecar shutdown
behaviour is not used when the pod is nil in the current implementation PR.
However, there is another PoC implementation using pod annotations, linked in
this doc, that handles this case by just adding the labels to the container
runtime.

From Rodrigo's experience (@rata) writing the PoC implementation, this seems
easy to fix, so we would suggest to fix it for alpha stage. This will be added
in detail to the proposal if agreed.

#### Pods with RestartPolicy Never

This case also derives from a _per pod_ setting that for some use cases having
it _per container_ can be desired.

The case is simple: pods with RestartPolicy Never will never even start
non-sidecar containers if sidecar containers crash. That is because the sidecar
is not restarted due to the pod RestartPolicy and non-sidecar containers will
only start once sidecar are up and in a ready state. As sidecars are not ready,
non-sidecars are not started.

The most common use case is for jobs. If the sidecar crashes and the policy is
to never restart, the pod will be "stalled" with no way to move forward.

Furthermore, if the podPhase is not modified to have a special behavior for
sidecar containers, the [phase will be pending][pod-phase-pending] (as some
containers were not started).

[pod-phase-pending]: https://github.com/kubernetes/kubernetes/blob/8398bc3b53cb51b341e14ae2a2cea01cedbf7904/pkg/kubelet/kubelet_pods.go#L1447-L1453

##### Alternative 1: Add a per container fatalToPod field

One option is to add a `fatalToPod` bool field _per container_. This will mean
that if the given container crashes, that is fatal to the pod so the pod is
killed.

This will give a clear way to have jobs with restartPolicy never and not leaving
stalled pods: a sidecar container can define this field and if it crash, it will
kill the pod.

For different (unclear) reasons, this functionality [was requested on the
mailing list][ml-kill-pod] in the past.

Another use of this functionality is that in some specific use cases, if a
container crashes, you just want to kill the entire pod and start again. This
was requested by some clients, at least.

I'd like to know what the rest think about this.

[ml-kill-pod]: https://discuss.kubernetes.io/t/how-can-i-have-a-pod-delete-itself-on-failure/8699

##### Alternative 2: Do nothing

Another option is to leave the pod stalled. A pod with containers that crash and
are not restarted, will have a similar behaviour. The main and non-trivial
difference, though, is that in this case some container will not be even
started (non-sidecar containers won't be started if sidecars crash).


##### Alternative 2: Always restart sidecar containers

Another option is to always restart sidecar containers, under the assumptions
that they are always needed.

This seems confusing, as doing this when the pod restartPolicy is Never doesn't
seem right. In other words, having a container not respect the pod restartPolicy
with no explicit mentions seems asking for trouble.

However, this is what the fork of at least one company is doing. Their use case
is using Jobs to run some workloads where the main (non-sidecar) container has TB
of memory in RAM, with no way to persist calculated data and may take weeks to run.
In those cases losing all of that because a sidecar crashes is a too expensive
price to pay and they resorted to always restart them.

We think that in those cases using an OnFailure RestartPolicy for the job might
seem more appropriate and, in any case, not having any kind of persistence for
graceful shutdown doesn't seem like a good reason to have this behaviour in
Kubernetes.

In any case, it seems like a real problem and worth exploring ways to handle
this. As always, ideas are very welcome :).

##### Suggestion

Go for Alternative 1. Seems simple and leaves the cluster in a clean state.
However, I really would like to know what others think.


#### Enforce the startup/shutdown behavior only on startup/shutdown

One of the goals of this KEP is to **only modify the startup/shutdown
sequence**. This makes the semantics clear and helps to have clean code, as only
those places will be changed.

Therefore, once non-sidecar containers have been started, kubernetes treats
containers in the pod indistinguishable from containers in a pod without
sidecar containers. That is, once sidecars and non-sidecars containers have
been started once, the regular kubernetes reconciliation loop is used for all
the containers in the pod during the pod lifecycle until the shutdown sequence
is fired. This guarantees that only the startup and shutdown behaviour of
sidecar containers is affected and not the rest of the pod lifecycle.

However, one side effect is that if, for example, all containers happen to
crash at the same time, all _can_ be restarted at the same time (if the restart
policy allows). This might be surprising, as instances of all the containers
being started at the same time can be seen by the users.

We believe this is fine, though, as the goal is to not change the behaviour
other than startup/shutdown and this edge case should be handled by users, as
any other container crashes.

If this behaviour is not welcome, however, code probably can be adapted to
handle the case when all containers crashed differently.

#### Sidecar containers won't be available during initContainers phase

The current proposal adds sidecar containers that will be started after all
initContainer executed. In other words, it just gives an order to containers on
the `containers` array.

For most users we talked about (some big companies and linkerd, will try to
contact istio soon) this doesn't seem like a problem. But wanted to make this
visible, just in case.

Furthermore, service meshes don't provide easy ways to use them with
initContainers (probably due to Kubernetes limitations). This KEP won't change
that.

Another thing to take into account is that Istio has an alternative to the
initContainer approach. Istio [has an option][istio-cni-opt] to integrate with
CNI and inject the blackhole from there instead of using the initContainer. In
that case, it will do (just c&p from the link, in case it breaks in the
future):

> By default Istio injects an initContainer, istio-init, in pods deployed in the mesh. The istio-init container sets up the pod network traffic redirection to/from the Istio sidecar proxy. This requires the user or service-account deploying pods to the mesh to have sufficient Kubernetes RBAC permissions to deploy containers with the NET_ADMIN and NET_RAW capabilities. Requiring Istio users to have elevated Kubernetes RBAC permissions is problematic for some organizations’ security compliance
> ...
> The Istio CNI plugin performs the Istio mesh pod traffic redirection in the Kubernetes pod lifecycle’s network setup phase, thereby removing the requirement for the NET_ADMIN and NET_RAW capabilities for users deploying pods into the Istio mesh. The Istio CNI plugin replaces the functionality provided by the istio-init container.

In other words, when using the CNI plugin it seems that InitContainer don't use
the service mesh either. Rodrigo will double check this, just in case.

[linkerd-last-container]: https://github.com/linkerd/linkerd2/issues/4758#issuecomment-658457737
[istio-cni-opt]: https://istio.io/latest/docs/setup/additional-setup/cni/

##### Suggestion

Confirm with users that is okay to not have sidecars during the initContainers
phase and they don't foresee any reason to add them in the near future.

It seems likely that is not a problem and this is a win for them, as they can
remove most of the hacks. However, it seems worth investigating if the CNI
plugin is a viable alternative for most service mesh and, in that case, how much
they will benefit from this sidecar KEP.

It seems likely that they will benefit for two reason: (a) this might
be simpler or model better what service mesh need during startup, (b) they still
need to solve the problem on shutdown, where the service mesh needs to drain
connections first and be running until others terminate. These are needed for
graceful shutdown and allowing other containers to use the network on shutdown,
respectively.

Rodrigo will reach out to users to verify, though.

#### Revisit if we want to modify the podPhase

The current proposal modifies the `podPhase`. The reasoning is this (c&p from
the proposal):

> PodPhase will be modified to not include Sidecars in its calculations, this is so that if a sidecar exits in failure it does not mark the pod as `Failed`. It also avoids the scenario in which a Pod has RestartPolicy `OnFailure`, if the containers all successfully complete, when the sidecar gets sent the shut down signal if it exits with a non-zero code the Pod phase would be calculated as `Running` despite all containers having exited permanently.

As noted by @zhan849 in [this review comment][pod-phase-review-comment], those
changes to the pod phase is a behavioural change regarding [current
documentation about the pod phase][pod-phase-doc].

[pod-phase-review-comment]: https://github.com/kubernetes/kubernetes/pull/80744/files?file-filters%5B%5D=.go#r379928630
[pod-phase-doc]: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-phase

##### Alternative 1: don't change the pod phase

We propose to not change the pod phase due to sidecar containers status. This
simplifies the code (no special behaviour is needed besides the startup or
shutdown sequences) and properly reflect the current documented phases.

Documentation says that during pending phase, not all containers have been
started. This is true for pods with sidecar containers as well, even when
sidecars are running and non-sidecar not yet.

The running state will also have the same meaning as currently documented: all
containers in the pod are running.

Furthermore, it seems correct to mark the pod as failed if a sidecar container
failed. For example, if a job has all containers exited successfully except for
a sidecar container that failed, marking it as failed seems the right call. That
sidecar container can be a container uploading files when the main container
finished. Just making sure it finishes to run seems the right thing to do.

##### Alternative 2: change the pod phase

Keep the current proposal to modify it, making sure everyone on the community is
on board with this modification.

##### Suggestion

Go for alternative 1, seems to lead to simpler code, pod phase seems to be
correct and no behavioural changes are done regarding the current documentation
that might need to have a special case when sidecars are running.

#### No container type standard

This proposal adds a field `type` to the pod spec containers array, but the only
value is `sidecar`. This seems weird as there would be containers with type
`sidecar` and containers with no type that will have different behavior.

This was already discussed [in the past][pr-kep-type-standard], with several
issues that bitten us when default values were added to pod spec (see the link
for examples). It was decided to not have a `standard` container type back then,
to avoid such issues.

We consider this weird but, as we can't do anything about it and we were okay in
the past, we propose to keep this as it is. It seems we were okay, judging from [that same
review comment][pr-kep-type-standard] and [this][pr-kep-api-ok].

[pr-kep-type-standard]: https://github.com/kubernetes/kubernetes/pull/79649#discussion_r362658887
[pr-kep-api-ok]: https://github.com/kubernetes/kubernetes/pull/79649#issuecomment-589861200

#### Is this change really worth doing?

We think this is a very important question to make and discuss. We think there is
probably agreement on this already, but we prefer to discuss it now and raise
any concerns (if any) to minimize chances of a last minute show stopper again.

This KEP has a clear motivation and explains the pain some users currently
experience. However, it is worth discussing openly if we want to fix this or if
this way is the right path to pursue. To avoid this discussion to happen on tons
of different places across the doc, we think adding a section just for this can
help concentrate the discussion about it. Hopefully this section will mention
most of the concerns reviewers might have.

This KEP proposes basically to fix two different problems:
 * Allow Kubernetes Jobs to have sidecar containers that run continuously
   without coupling with other containers in the pod.
 * Allow pods to start a subset of containers first and, only when those are
   ready, start the remaining containers. And similar guarantees during
   shutdown.

Let's analyse both.

##### Kubernetes jobs with sidecars

As explained in the motivation section, this is a problem today. Also, the
workaround needed from users to do this today seem horrible: coupling between
containers to signal termination are invasive and require modifications to all
containers in the pod. Additionally, they usually are prone and gets in the way
of auto-injecting containers for jobs.

In addition, Kubernetes seems the best layer to do a fix: it can have all the
needed information to know which containers should be killed when some others
are finished and avoid any kind of coupling in other layers.

##### Service mesh

For Istio service mesh, part of the problem were solved by Istio using a CNI
plugin: the CNI plugin created the iptables redirect it is needed to proxy the
traffic. This removed the need for an initContainer. However, some problems
persist:
1. If the service mesh is not started before other containers, containers will
   start without network connectivity
1. If the service mesh is killed before other containers, those won't be able to
   use the network during shutdown (and possible finish processing inflight
   requests)

It is worth noting, however, that the first problem is something that *should*
be handled in any case: if the service mesh container crashes, others container
won't be able to use the network. And if they happen to crash and be restarted
while the service mesh container is down, it is the same situation that can
happen today at startup. So, this _really_ should be handled at the application
layer. However, as mentioned in the motivation section, there are some
apps/middleware that have this problem on startup only.

The other problem seems to be very directed to the way Kubernetes starts and
stops containers: if that isn't changed in Kubernetes, it seems difficult to
solve by any other piece in the stack other than Kubernetes.

##### Kubernetes pods that run indefinitely with sidecars

One concern might be that this KEP helps to hide some errors from users. For
example: let's say a we have a pod with two container: A and B. And let's say
container B doesn't handle correctly the case when container A crashes.

If a user is unaware of B not handling correctly when A crashes and makes
container A a sidecar, this may indeed hide most occurrences of B not handling
the cases where A crashes. Because B will be started only when A is up and in a
ready state.

This is a valid concern. We think this can be mitigated by good documentation,
clarifying that this is only at startup and container crashes need to still be
handled correctly. I'd love to hear other opinions, though.

Furthermore, adding the sidecar functionality can improve the experience of
several users (properly drain connections, reduce errors on shutdown or even not
lose events during shutdown as shown in the motivation section) and other nice
side effects. Those are mentioned in detail in the motivation section.

Another side effect _can_ be reduced startup time (it might not be the case for
all users, though). Even when using applications that use a backoff to retry
when things fails, if containers are started in a "random" order it can increase
the startup time considerably. If some containers are started only after some
others are ready, the startup time can be reduced, especially as the number of
sidecar containers increases (imagine a pod with 6 sidecar containers). The
startup time can have an interesting effect when it is combined with
autoscaling, for example.

##### Summary

We think both cases are worth solving and this way seems like a good way to
solve them: start/stop containers is something kubernetes does and therefore
seems the right layer to control that.

We also think that there is some risk of users using `type: sidecar` to any
sidecar container that doesn't need any startup/shutdown special handling. But
this is something we can gather feedback during alpha stage and properly
document. We think that will be enough to alleviate that concern (if anyone
actually has it) while at the same time achieve the goals this KEP tries to
achieve.

All opinions are very welcome on this. Please comment.

#### Why this design seems extensible?

This is a difficult section to add, as different people probably have different
concerns. So, I'd list the ones that we can think of to open this discussion, and
encourage everyone to review this critically and see if there is something
missing to take into account.

Let's try to look into the extensibility of this under different scenarios, to
have a better understanding, but is of course impossible to give guarantees.

##### What if we add pre-defined phases for container startup/shutdown?

This is an invented example. Let's suppose we add a `type: OrderedPhase` in the
future, that also includes another `Order` field that is an int, starting from
0.

In this case, containers are started from lower to higher. And stopped from
higher to lower. For example, first containers with `Order: 0` will be started,
then container with `Order: 1`, etc. And in the reverse order for shutdown.

Let's call standard container type to containers without the `type` field set.

It seems the current containers type `standard` and `sidecar` can be easily
extended for this: standard is mapped to one particular int (let's say 1) and
sidecar is mapped into the previous one (0 in this example).

Then, all container type can coexist just fine and the startup and shutdown
sequence is clearly defined.

If the "NotificationHook" suggested is also implemented, then the shutdown
sequence will run that first, no matter the container _type_, as the first step in the
shutdown sequence.

##### What if we add "Depends On" semantics to containers?

Let's call standard container type to containers without the `type` field set.

This is very similar to the previous:
 * TerminationHook doesn't need any adaptation
 * Containers type standard will be treated as having a "Depends On" on all
   sidecar type containers
 * Type sidecar containers won't depend on other

This semantic will just be a 1-1 translation from the proposed KEP to this
"Depends On" semantic, so it is easy for them to coexist.

### Proof of concept implementations

#### KEP implementation and Demo

There is a [PR here](https://github.com/kubernetes/kubernetes/pull/75099) with a working Proof of concept for this KEP, it's just a draft but should help illustrate what these changes would look like.

Please view this [video](https://youtu.be/4hC8t6_8bTs) if you want to see what the PoC looks like in action.

#### Another implementation using pod annotations

Another implementation using pod annotations on top of Kubernetes 1.17 is
available [here][kinvolk-sidecar-branch].

There are some example yamls in the [sidecar-tests
folder][kinvolk-poc-sidecar-test], also the yaml output was captured to easily
see the behavior. [See the commit that created
them][kinvolk-poc-sidecar-test-commit] for instruction on how to run it.

Some other things worth noting about the implementation:
 * It is done using pod annotations so it is easy to test for users (doesn't
   modify pod spec)
 * It implements the KEP proposal and not the suggested modifications, with one
   exception: the podPhase is not modified.
 * [This line][kinvolk-poc-sidecar-prestop] should be modified to use the
   `TerminationHook` instead, if such alternative is chosen
 * There is some c&p code to avoid doing refactors and just have a patch that is
   simpler to cherry-pick on different Kubernetes versions.

[kinvolk-poc-sidecar-prestop]: https://github.com/kinvolk/kubernetes/blob/52a96112b3e7878740a0945ad3fc4c6d0a6c5227/pkg/kubelet/kuberuntime/kuberuntime_container.go#L851
[kinvolk-sidecar-branch]: https://github.com/kinvolk/kubernetes/tree/rata/sidecar-ordering-annotations-1.17
[kinvolk-poc-sidecar-test]: https://github.com/kinvolk/kubernetes/tree/52a96112b3e7878740a0945ad3fc4c6d0a6c5227/sidecar-tests
[kinvolk-poc-sidecar-test-commit]: https://github.com/kinvolk/kubernetes/commit/385a89d83df9c076963d2943507d1527ffa606f7

### Risks and Mitigations

You could set all containers to have `lifecycle.type: Sidecar`, this would cause strange behaviour in regards to shutting down the sidecars when all the non-sidecars have exited. To solve this the api could do a validation check that at least one container is not a sidecar.

Init containers would be able to have `lifecycle.type: Sidecar` applied to them as it's an additional field to the container spec, this doesn't currently make sense as init containers are ran sequentially. We could get around this by having the api throw a validation error if you try to use this field on an init container or just ignore the field.

Older Kubelets that don't implement the sidecar logic could have a pod scheduled on them that has the sidecar field. As this field is just an addition to the Container Spec the Kubelet would still be able to schedule the pod, treating the sidecars as if they were just a normal container. This could potentially cause confusion to a user as their pod would not behave in the way they expect, but would avoid pods being unable to schedule.

Shutdown ordering of Containers in a Pod can not be guaranteed when a node is being shutdown, this is due to the fact that the Kubelet is not responsible for stopping containers when the node shuts down, it is instead handed off to systemd (when on Linux) which would not be aware of the ordering requirements. Daemonset and static Pods would be the most effected as they are typically not drained from a node before it is shutdown. This could be seen as a larger issue with node shutdown (also effects things like termination grace period) and does not necessarily need to be addressed in this KEP , however it should be clear in the documentation what guarantees we can provide in regards to the ordering.

## Design Details

### Test Plan
* Units test in kubelet package `kuberuntime` primarily in the same style as `TestComputePodActions` to test a variety of scenarios.
* New E2E Tests to validate that pods with sidecars behave as expected e.g:
 * Pod with sidecars starts sidecars containers before non-sidecars
 * Pod with sidecars terminates non-sidecar containers before sidecars
 * Pod with init containers and sidecars starts sidecars after init phase, before non-sidecars
 * Termination grace period is still respected when terminating a Pod with sidecars
 * Pod with sidecars terminates sidecars once non-sidecars have completed when `restartPolicy` != `Always`
 * Pod phase should be `Failed` if any sidecar exits in failure when `restartPolicy` != `Always`
 * Pod phase should be `Succeeded` if all containers, including sidecars, exit with success when `restartPolicy` != `Always`


### Graduation Criteria
#### Alpha -> Beta Graduation
* Addressed feedback from Alpha testers
* Thorough E2E and Unit testing in place
* The beta API either supports the important use cases discovered during alpha testing, or has room for further enhancements that would support them
* Graduation depends on the (yet not submitted) kubelet graceful shutdown KEP
  reaching Beta stage. It is okay to for both features to reach Beta in the same
  release, but this KEP should not reach beta before kubelet graceful shutdown KEP


#### Beta -> GA Graduation
* Sufficient number of end users are using the feature
* We're confident that no further API changes will be needed to achieve the goals of the KEP
* All known blocking bugs have been fixed
* Graduation depends on the (not yet submitted) kubelet graceful shutdown KEP
  being in GA for at least 1 release and no concerns identified that may affect
  this KEP

### Upgrade / Downgrade Strategy
When upgrading no changes should be needed to maintain existing behaviour as all of this behaviour is optional and disabled by default.
To activate the feature they will need to enable the feature gate and mark their containers as sidecars in the container spec.

When downgrading `kubectl`, users will need to remove the sidecar field from any of their Kubernetes manifest files as `kubectl` will refuse to apply manifests with an unknown field (unless you use `--validate=false`).

### Version Skew Strategy
Older Kubelets should still be able to schedule Pods that have sidecar containers however they will behave just like a normal container.

## Implementation History

- 14th May 2018: Proposal Submitted
- 26th June 2019: KEP Marked as implementable
- 24th June 2020: KEP Marked as provisional. Got stalled on [March 10][stalled]
  with a clear explanation. The topic has been discussed in SIG-node and this
  KEP will be evolved with, at least, some already discussed changes.

[stalled]: https://github.com/kubernetes/enhancements/issues/753#issuecomment-597372056

## Alternatives

### Alternative designs considered

This section contains ideas that were originally discussed but then dismissed in favour of the current design.
It also includes some links to related discussion on each topic to give some extra context, however not all decisions are documented in Github prs and may have been discussed in sig-meetings or in slack etc.

#### Add a pod.spec.SidecarContainers array
An early idea was to have a separate list of containers in a similar style to init containers, they would have behaved in the same way that the current KEP details. The reason this was dismissed was due to it being considered too large a change to the API that would require a lot of updates to tooling, for a feature that in most respects would act the same as a normal container.

```yaml
initContainers:
  - name: myInit
containers:
  - name: myApp
sidecarContainers:
  - name: mySidecar
```
Discussion links:
https://github.com/kubernetes/community/pull/2148#issuecomment-388813902
https://github.com/kubernetes/community/pull/2148#discussion_r221103216

#### Mark one container as the Primary Container
The primary container idea was specific to solving the issue of Jobs that don't complete with a sidecar, the suggestion was to have one container marked as the primary so that the Job would get completed when that container has finished. This was dismissed as it was too specific to Jobs whereas the more generic issues of sidecars could be useful in other places.
```yaml
kind: Job
spec:
  template:
    spec:
      containers:
      - name: worker
        command: ["do a job"]
      - name: "sidecar"
        command: ["help"]
  backoffLimit: 4
  primaryContainer: worker
```
Discussion links:
https://github.com/kubernetes/community/pull/2148#discussion_r192846570

#### Boolean flag on container, Sidecar: true
```yaml
containers:
  - name: myApp
  - name: mySidecar
    sidecar: true
```
A boolean flag of `sidecar: true` could be used to indicate which pods are sidecars, this was dismissed as it was considered too specific and potentially other types of container lifecycle may want to be added in the future.

#### Mark containers whose termination kills the pod, terminationFatalToPod: true
This suggestion was to have the ability to mark certain containers as critical to the pod, if they exited it would cause the other containers to exit. While this helped solve things like Jobs it didn't solve the wider issue of ordering startup and shutdown.

```yaml
containers:
  - name: myApp
    terminationFatalToPod: true
  - name: mySidecar
```
Discussion links:
https://github.com/kubernetes/community/pull/2148#issuecomment-414806613

#### Add "Depends On" semantics to containers
Similar to [systemd](https://www.freedesktop.org/wiki/Software/systemd/) this would allow you to specify that a container depends on another container, preventing that container from starting until the container it depends on has also started. This could also be used in shutdown to ensure that the containers which have dependent containers are only terminated after their dependents have all safely shut down.
```yaml
containers:
  - name: myApp
    dependsOn: mySidecar
  - name: mySidecar
```
This was rejected as the UX was considered to be overly complicated for the use cases that we were trying to solve. It also doesn't solve the problem of Job termination where you want the sidecars to be terminated once the main containers have exited, to do that you would require some additional fields that would further complicate the UX.
Discussion links:
https://github.com/kubernetes/community/pull/2148#discussion_r203071377

#### Pre-defined phases for container startup/shutdown or arbitrary numbers for ordering
There were a few variations of this but they all had a similar idea which was the ability to order both the shutdown and startup of containers using phases or numbers to determine the ordering.
examples:
```yaml
containers:
  - name: myApp
    StartupPhase: default
  - name: mySidecar
    StartupPhase: post-init
```

```yaml
containers:
  - name: myApp
    startupOrder: 2
    shutdownOrder: 1
  - name: mySidecar
    startupOrder: 2
    shutdownOrder: 1
  ```
This was dismissed as the UX was considered overly complex for the use cases we were trying to enable and also lacked the semantics around container shutdown triggering for things like Jobs.
Discussion links:
https://github.com/kubernetes/community/pull/2148#issuecomment-424494976
https://github.com/kubernetes/community/pull/2148#discussion_r221094552
https://github.com/kubernetes/enhancements/pull/841#discussion_r257906512

### Workarounds sidecars need to do today

This section show the alternatives and workaround app developers need to do
today.

#### Jobs with sidecar containers

The problem is described in the [Motivation
section](#problems-jobs-with-sidecar-containers). Here, we present some
alternatives and the pain points that affect users today.

Most known work-arounds for this are achieved by building an ad-hoc signalling
mechanism to communicate completion status between containers. Common
implementations use a shared scratch volume mounted into all pods, where
lifecycle status can be communicated by creating and watching for the presence
of files. With the disadvantages of:

 * Repetitive lifecycle logic must be incorporated into each sidecar (code might
   be shared, but is usually language dependant)
 * Wrappers can alleviate that, but it is still quite complex when there are
   several sidecar to wait for. When more than one sidecar is used, some
   question arise: how many sidecars a wrapper has to wait for? How can that be
   configured in a non-error prone way? How can I use wrappers while still inject
   sidecars automatically and reliably in a mutating webhook or programmatically?

Other possible work-arounds can be using a shared PID namespace and checking for
other containers running or not. Also, it comes with several disadvantages, like:

 * Security concerns around sharing PID namespace (might be able to see other
   containers env vars via /proc, or even the root filesystem, depends on
   permissions used)
 * Restricts the possibility of changing the container runtime, until all
   runtimes support a shared PID namespace
 * Several applications might need re-work, as PID 1 is not the container
   entrypoint anymore.

Using a wrapper with this approach might sound viable for _some_ use cases, but
when you add to the mix that containers can have more than one sidecar, then
each container has to know which other containers are sidecar to know if it is
safe to proceed. This becomes specially tricky when combined with auto-injection
of sidecars, and even more complicated if auto-inject is done by a third party
or independent team.

Furthermore, wrappers have several pain points if you want to use them for
startup, as explained in the next section.

#### Service mesh or metrics sidecars

Let app container be the main app that just has the service mesh extra container
in the pod.

Service mesh, today, have to do the following workarounds due to lack of startup
ordering:
 * Blackhole all the traffic until service mesh container is up (usually using
   an initContainer for this)
 * Some trickery (sleep preStop hooks or some alternative) to not be killed
   before other containers that need to use the network. Otherwise, traffic for
   those containers will be blackholed

This means that if the app container is started before the service mesh is
started and ready, all traffic will be blackholed and the app needs to retry.
Once the service mesh container is ready, traffic will be allowed.

This has another major disadvantage: several apps crash if traffic is blackholed
during startup (common in some rails middleware, for example) and have to resort
to some kind of workaround, like [this one][linkerd-wait] to wait. This makes
also service mesh miss their goal of augmenting containers functionality without
modifying the main application.

Istio has an alternative to the initContainer hack. Istio [has an
option][istio-cni-opt] to integrate with CNI and inject the blackhole from there
instead of using the initContainer. In that case, it will do (just c&p from the
link, in case it breaks in the future):

> By default Istio injects an initContainer, istio-init, in pods deployed in the mesh. The istio-init container sets up the pod network traffic redirection to/from the Istio sidecar proxy. This requires the user or service-account deploying pods to the mesh to have sufficient Kubernetes RBAC permissions to deploy containers with the NET_ADMIN and NET_RAW capabilities. Requiring Istio users to have elevated Kubernetes RBAC permissions is problematic for some organizations’ security compliance
> ...
> The Istio CNI plugin performs the Istio mesh pod traffic redirection in the Kubernetes pod lifecycle’s network setup phase, thereby removing the requirement for the NET_ADMIN and NET_RAW capabilities for users deploying pods into the Istio mesh. The Istio CNI plugin replaces the functionality provided by the istio-init container.

In other words, Istio has an alternative to configure the traffic blockhole
without an initContainer. But the other problems and hacks mentioned remain,
though.

[linkerd-last-container]: https://github.com/linkerd/linkerd2/issues/4758#issuecomment-658457737
[istio-cni-opt]: https://istio.io/latest/docs/setup/additional-setup/cni/
[linkerd-wait]: https://github.com/olix0r/linkerd-await

##### Istio bug report

There is also a [2 years old bug][istio-bug-report] from Istio devs that this
KEP will fix. In addition, similar benefit is expected for Linkerd, as we talked
with Linkerd devs.

One of the things mentioned there is that, at least in 2018, a workaround used
was to tell the user to run a script to wait for the service mesh to start on
their containers.

Rodrigo will reach out to Istio devs to see if the situation changed since 2018.

[istio-bug-report]: https://github.com/kubernetes/kubernetes/issues/65502

#### Move containers out of the pod

Due to the ordering problems of having the container in the same pod, another
option is to move it out of the pod. This will, for example, remove the problem
of shutdown order. Furthermore, Rodrigo will argue than in many cases this is
better or more elegant in Kubernetes.

While some things might be possible to move to daemonset or others, it is not
possible for all applications. For example some apps are not multi-tenant and
this can not be an option security-wise. Or some apps would still like to have a
sidecar that adds some metadata, for example.

While this is an option, is not possible or extremely costly for several use
cases.
