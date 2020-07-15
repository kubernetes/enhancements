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
  - [Jobs](#jobs)
  - [Startup](#startup)
  - [Shutdown](#shutdown)
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
    - [PoC and Demo](#poc-and-demo)
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
  - [Add a pod.spec.SidecarContainers array](#add-a-podspecsidecarcontainers-array)
  - [Mark one container as the Primary Container](#mark-one-container-as-the-primary-container)
  - [Boolean flag on container, Sidecar: true](#boolean-flag-on-container-sidecar-true)
  - [Mark containers whose termination kills the pod, terminationFatalToPod: true](#mark-containers-whose-termination-kills-the-pod-terminationfataltopod-true)
  - [Add &quot;Depends On&quot; semantics to containers](#add-depends-on-semantics-to-containers)
  - [Pre-defined phases for container startup/shutdown or arbitrary numbers for ordering](#pre-defined-phases-for-container-startupshutdown-or-arbitrary-numbers-for-ordering)
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

This KEP adds the notion of sidecar containers to Kubernetes. This KEP proses
to add a `type` field to the `containers` array in the `pod.spec` to define if a
container is a sidecar container. The only valid value for now is `sidecar`, but
other values can be added in the future if needed.

Pods with sidecar containers only change the behavior of the startup and
shutdown sequence of a pod: sidecar containers are started before non-sidecars
and stopped after non-sidecars.

A pod that has sidecar containers guarantees that non-sidecar containers are
started only after sidecar containers are started and are in a ready state.
Furthermore, we propose to treat sidecar containers as regular (non-sidecar)
containers as much as possible all over the code, except for the mentioned
special startup and shutdown behavior. The rest of the pod lifecycle (regarding
restarts, etc.) remains unchanged, this KEP aims to modify only the startup and
shutdown behavior.

If a pod doesn't have a sidecar container, the behavior is completely unchanged
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

The concept of sidecar containers has been around almost since Kubernetes early
days. A clear example is [this Kubernetes blog post][sidecar-blog-post] from
2015 mentioning the sidecar pattern.

Over the years the sidecar pattern has become more common in applications,
gained popularity and uses cases are getting more diverse. The current
Kubernetes primitives handled that well, but they are starting to fall short for
several use cases and force weird work-arounds in the applications. 

This proposal aims to remediate that by adding a simple set of guarantees for
sidecar containers, while trying to not do a complete re-implementation of an
init system to do so. Those alternatives are interesting and were considered,
but in the end the community decided to go for something simpler that will cover
most of the use cases. Those options are explored in the
[alternatives](#alternatives) section.

The next section expands on what the current problems are. But, to give more
context, it is important to highlight that some companies are already using a
fork of Kubernetes with this sidecar functionality added (not all
implementations are the same, but more than one company has a fork for this).

[sidecar-blog-post]: https://kubernetes.io/blog/2015/06/the-distributed-system-toolkit-patterns/#example-1-sidecar-containers

### Problems: jobs with sidecar containers

Imagine you have a Job with two containers: one which does the main processing
of the job and the other is just a sidecar facilitating it. This sidecar can be
a service mesh, a metrics gathering statsd server, etc.

When the main processing finishes, the pod won't terminate until the sidecar
container finishes too. This is problematic for sidecar containers that run
continously, like service-mesh or statsd metrics server or servers in general.

There is no simple way to handle this on Kubernetes today. There are
work-arounds for this problem, most of them consist of some form of coupling
between the containers to add some logic where a container that finishes
communicates it so other containers can react. But it gets tricky when you have
more than one sidecar container or want to auto-inject sidecars. Some
alternatives to achieve this currently and their pain points are discussed in
detail on the [alternatives](#alternatives) section.

### Problems: service mesh or metrics sidecars

While this problem is general to any sidecar container that might need to start
before others or stop after others with special handling on shutdown, we use
here a service mesh, metrics gathering and logging sidecar for the examples.

#### Logging/Metrics sidecar

A logging sidecar should start before several other containers, to not
lose logs from the startup of other applications. Let's call _main container_
the app that will log and _logging container_ the sidecar that will facilitate
it.

If the logging container starts after the main app, some logs can be lost.
Furthermore, if the logging container is not yet started and the main app
crashes on startup, those logs can be lost (depends if logs to a shared volume
or over the network on localhost, etc.). Startup is usually not that critical,
because if the case where the other container is not started (like a restart) is
important, is usually handled correctly.

On shutdown the ordering behavior is arguably more important: if the logging
container is stopped first, logs for other containers are lost. No matter if
those containers queue them and retry to send them to the logging container, or
if they are persisted to a shared volume. The logging container is already
killed and will not be started, as the pod is shutting down. In those cases,
logs are lost.

The same things regarding startup and shutdown apply for a metrics container.

Some work-arounds can be done, to alliviate the symptoms:
 * Ignore SIGTERM on the sidecar container, so it is alive until the pod is
   killed. This is not ideal and increases _a lot_ the time a pod will take to
terminate. For example, if the P99 response time is 2 minutes and therefore the
TerminationGracePeriodSeconds is set to that, the main container can finish in 2
seconds (that might be the average) but ignoring SIGTERM in the sidecar
container will force the pod to live for 2 minutes anyways.
 * Use preStop hooks that just runs a "sleep X" seconds. This is very similar to
   the previous item and has the same problems.

#### Service mesh

Service mesh present a similar problem: you want them to be set and ready before
other containers start, so any inbound/outbound connection that any container can
initiate goes through the service mesh.

A similar problem happens for shutdown: if the service mesh container is
terminated prior to other containers, outgoing traffic from other apps will be
blackholed or not use the service mesh.

However, as none of these are possible to guarantee most service mesh (like
Linkerd and Istio), for example, need to do several hacks to have the basic
functionality. These are explained in detail in the
[alternatives](#alternatives) section. Nonetheless, here is a  quick highlight
of some of the things some service mesh currently need to do:

 * Recommend users to delay starting their apps by using a script to wait for
   the service mesh to be ready. The goal of a service mesh to augment the app
   functionality without modifying it is lost in this case.
 * To guarantee that traffic goes via the services mesh, an initContainer is
   added to blackhole traffic until the service mesh containers is up. This way,
   other containers that might be started before the service mesh container can't
   use the network until the service mesh container is started and ready. A side
   effect is that traffic is blackholed until the service mesh is up and in a
   ready state.
 * Use preStop hooks with a "sleep infinity" to make sure the service mesh
   doesn't terminate before other containers that might be serving requests.

The auto-inject of initContainer [has caused bugs][linkerd-bug-init-last], as it
competes with other tools auto-injecting container to be run last too.

[linkerd-bug-init-last]: https://github.com/linkerd/linkerd2/issues/4758#issuecomment-658457737

### Problems: Coupling infrastructure with applications

The limitations to have some ordering guarantees on startup and shutdown,
sometimes forces to couple application code with infrastructure. This is not
nice, as the infrastructure is ideally handled independently, and forces more
coordination between multiple, possibly indepedant, teams.

An example in the open source world about this is the [Istio CNI
plugin][istio-cni-plugin]. This was created as an alternative for the
initContainer hack that service mesh need to do. The initContainer will
blackhole all traffic until the Istio container is up. But this alternative
requires that nodes have the CNI plugin installed, effectively coupling the
service mesh app with the infrastructure.

This KEP proposal removes the need for service mesh to use either an
initContainer or a CNI plugin: just guarantee that the sidecar container can be
started first.

While in this specific example the CNI plugin has some benefits too (removes the
need for some capabilities in the pod) and might be worth pursing, it is used
here just as an examples of possible coupling apps with infrastructure. Similar
examples also exist in-house for not open source applications too.

[istio-cni-plugin]: https://istio.io/latest/docs/setup/additional-setup/cni/

## Goals

This proposal aims to:
 * Allow Kubernetes Jobs to have sidecar containers that run continously
   (servers) without coupling between containers in the pod.
 * Allow pods to start a subset of containers first and, only when those are
   ready, start the remaining containers.
 * Allow pods to stop a subset of containers first and, only when those have
   been stopped, stop the rest.
 * Not change the semantics in any way for pods not using this feature.
 * Only change the semantics of startup/shutdown behavior for pods using this
   feature.

Solve issues so that they don't require application modification:
* [25908](https://github.com/kubernetes/kubernetes/issues/25908) - Job completion
* [65502](https://github.com/kubernetes/kubernetes/issues/65502) - Container startup dependencies. Istio bug report

## Non-Goals

This proposal doesn't aim to:
 * Allow a multi-level ordering of containers start/stop sequence. IOW, only two
   levels are deinfed: "sidecar" or "standard" containers. It is not a goal to
   have more than these two levels.
 * Model startup/shutdown dependencies between different sidecar containers
 * Change the pod phase in any way, with or without this feature enabled
 * Reimplement an init-system alike semantics for pod containers
   startup/shutdown
 * Allow sidecar containers to run concurrently with initContainers

Allowing multiple containers to run at once during the init phase - this could be solved using the same principal but can be implemented separately. //TODO write up how we could solve the init problem with this proposal

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
* Sidecars start
* Sidecars become ready
* Containers start

During pod termination sidecars will be terminated last:
* Containers sent SIGTERM
* Once all Containers have exited: Sidecars sent SIGTERM

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

#### PoC and Demo
There is a [PR here](https://github.com/kubernetes/kubernetes/pull/75099) with a working Proof of concept for this KEP, it's just a draft but should help illustrate what these changes would look like.

Please view this [video](https://youtu.be/4hC8t6_8bTs) if you want to see what the PoC looks like in action.

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
  reaching Beta stage and no concerns identified that may affect this KEP


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
This section contains ideas that were originally discussed but then dismissed in favour of the current design.
It also includes some links to related discussion on each topic to give some extra context, however not all decisions are documented in Github prs and may have been discussed in sig-meetings or in slack etc.
### Add a pod.spec.SidecarContainers array
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

### Mark one container as the Primary Container
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

### Boolean flag on container, Sidecar: true
```yaml
containers:
  - name: myApp
  - name: mySidecar
    sidecar: true
```
A boolean flag of `sidecar: true` could be used to indicate which pods are sidecars, this was dismissed as it was considered too specific and potentially other types of container lifecycle may want to be added in the future.

### Mark containers whose termination kills the pod, terminationFatalToPod: true
This suggestion was to have the ability to mark certain containers as critical to the pod, if they exited it would cause the other containers to exit. While this helped solve things like Jobs it didn't solve the wider issue of ordering startup and shutdown.

```yaml
containers:
  - name: myApp
    terminationFatalToPod: true
  - name: mySidecar
```
Discussion links:
https://github.com/kubernetes/community/pull/2148#issuecomment-414806613

### Add "Depends On" semantics to containers
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

### Pre-defined phases for container startup/shutdown or arbitrary numbers for ordering
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
