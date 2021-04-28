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
  - [Startup within a phase](#startup-within-a-phase)
  - [&quot;Completion&quot; and container restart policy](#completion-and-container-restart-policy)
  - [Phase: Network](#phase-network)
  - [Phase: ClusterEnv](#phase-clusterenv)
  - [Phase: AppInit](#phase-appinit)
  - [Phase: AppRun](#phase-apprun)
  - [End-of-life](#end-of-life)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
    - [InitContainers](#initcontainers)
    - [RestartPolicy](#restartpolicy)
    - [Pods with RestartPolicy Never](#pods-with-restartpolicy-never)
    - [Modify killContainer() to enforece gracePeriodOverride on preStop hooks](#modify-killcontainer-to-enforece-graceperiodoverride-on-prestop-hooks)
    - [Time to kill a pod increased by 4 seconds in the worst case](#time-to-kill-a-pod-increased-by-4-seconds-in-the-worst-case)
    - [Enforce the startup/shutdown behavior only on startup/shutdown](#enforce-the-startupshutdown-behavior-only-on-startupshutdown)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Proof of concept implementations](#proof-of-concept-implementations)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Alternative designs considered](#alternative-designs-considered)
    - [Add a pod.spec.lifecycle.type=sidecar,  implementation and Demo](#add-a-podspeclifecycletypesidecar--implementation-and-demo)
    - [Using pod annotations](#using-pod-annotations)
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

From a high level, this proposal can be summarized as “add more phases and define container restart policies”.

We believe that a relatively small number of well-defined, linear phases, with clear statements of what parts of the pod are working or not working in each phase can address most user requirements without imposing the complexity of a full dependency graph on end users. As a pod moves through the phases, containers within each phase can depend on the results of containers executed in previous phases. When all containers in phase X are “complete” (see below for the definition), containers in phase Y may be started.

This proposal rejects the proposed requirement that users be able to express explicit container-to-container startup dependencies. We believe this proposal captures the majority of the intent in a simpler way:
* Sidecars must be guaranteed to start BEFORE and end AFTER application workloads.
* Service-mesh proxy sidecars must be able to capture traffic from initContainers.
* Sidecars must not artificially prevent the termination of a pod when app containers exit.
* Application authors must not need to be aware of sidecars when it comes to sequencing.
* If a pod doesn't have a sidecar container, the behaviour is completely unchanged.

## Prerequisites

On June 23 2020, during SIG-node meeting, it was decided that this KEP has a
prerequisite on the [node graceful shutdown KEP].

As of writing, this enhancement has reached beta (since 1.21) lifting any prerequisite on this KEP.

[node graceful shutdown KEP]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2000-graceful-node-shutdown

## Motivation

The concept of sidecar containers has been around since the early days of
Kubernetes. A clear example is [this Kubernetes blog post][sidecar-blog-post]
from 2015 mentioning the sidecar pattern.

Over the years the sidecar pattern has become more common in applications,
gained popularity and the uses cases are getting more diverse. The current
Kubernetes primitives handled that well, but they are starting to fall short for
several use cases and force weird work-arounds in the applications.

Pods today are split into 2 lifecycle “phases”. First, all init containers (the `initContainers` field) are run to completion, in serial. They look, schematically, like regular containers (the `containers` field) but many sub-fields are validated as “must not be specified for init containers” (blech!). Kubernetes does not make many statements about what does or does not work during the init container sequence, and indeed many init containers rely on things like volumes or networking to be up. Kubernetes does not offer any real way to sequence init containers (though they can be re-run) other than the order in which they are defined, and does not have any affordances or metadata about init container behaviors or needs, which might facilitate automatic sequencing. Perhaps most importantly, Kubernetes offers no way for init containers to remain active while app containers are running.

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
[alternatives](#alternatives) section. Nonetheless, here is a quick highlight
of some of the things some service mesh currently need to do:
 * Recommend users to delay starting their apps by using a script to wait for
   the service mesh to be ready. The goal of a service mesh to augment the app
   functionality without modifying it, that goal is lost in this case.
 * If they don't delay starting their application, the network connection they
   try to establish are blacklisted until the service mesh container is up.
 * Use preStop hooks with a "sleep infinity" to make sure the service mesh
   doesn't terminate before other containers that might be serving requests.

This KEP adds guarantees to startup/shutdown behavior, so _those_ problems will
be solved for service mesh. However, service mesh do have other problems that
are out of scope for this KEP, e.g. enable service mesh before initContainers
are started.

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

A simple approach to the API expression for this proposal is to add a single enumerated-string field to each container in `containers`, defining the phase. If this field is not specified, we can assume AppRun, for compatibility. This approach requires careful handling of version-skew and adding new phases, and must define what happens when a back-rev kubelet finds a phase it does not understand. This requires treating `initContainers` as a special-case (see the [note](#initcontainers) for more on this).

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
    phase: AppRun # default phase
    command: ['do something']
  - name: sidecar
    image: sidecar-image
    phase: AppInit
    restartPolicy: OnFailure
    command: ["do something to help my app"]

```

### Startup within a phase

Within a single phase, there are no guarantees made about container startup ordering.They may be started in serial, in parallel, or some combination thereof as determined by the system. This is intended to emphasize that phased-containers are just an extension of the "normal" container lifecycle. and that sequencing within a phase is a non-thing (lest users come to rely on *that* rather than the documented properties of each phase).

### "Completion" and container restart policy

Unlike init containers, the containers within a phase are NOT necessarily run to completion, but may elect to stay resident. To achieve this, we propose to add a `restartPolicy` field to each container, seeded with the same values as the pod `restartPolicy` field.

Containers which set this field to “Always” are assumed to be “sidecars” and they are considered “complete” when they become “ready” (including being subject to `startupProbe` and `readinessProbe`). If such a container terminates (or indicates failure through liveness probe), it will be restarted, the same as any other container. If multiple containers fail, they will be restarted in phase-order.

Containers which set this field to “OnFailure” are assumed to be roughly equivalent to init containers in that they must run to successful completion before their phase is complete. If such a container terminates without success, it will be restarted.

Containers which set this field to “Never” must run to completion before their phase is complete. If such a container terminates without success, the phase will be failed and the pod’s own `restartPolicy` will be invoked.

### Phase: Network

Containers in this phase are responsible for establishing the low-level environment for the pod, including things like network connectivity and, as such, may not rely on the network themselves.  Kubernetes projected volumes (e.g. Secrets) are available, and PersistentVolumes are mounted, so logging agents can run here as long as they can buffer until this phase completes.

Optional: If we link this to network plugins lifecycle (e.g. CNI), we could ensure that nobody can use the network during this phase.

Examples:
* Installing iptables rules to capture traffic
* Starting the service-mesh proxy


### Phase: ClusterEnv

This phase is earmarked for non-app-defined containers (e.g. injected sidecars) which want to envelope the user’s application.  Containers in this phase may rely on the network and can perform initialization of the pod environment.

Examples:
* Populating cloud credentials

### Phase: AppInit

This phase is roughly what the pod’s `initContainers` field represents today.  Containers in this phase may rely on the network and can perform initialization of the pod.  This phase is earmarked as “for the app”.  Injected sidecars should use the previous phases.

Examples:
* Populate data in an `emptyDir` volume

### Phase: AppRun

This phase is what the pod’s `containers` field represents today.  Containers in this phase are fully empowered.  This phase is special.  When this phase is complete, the pod’s end-of-life begins.

### End-of-life

A pod’s end-of-life can be triggered by one of several mechanisms:
* Successful completion of all AppRun containers
* Kubelet declares failure of any phase
* External termination (e.g. Pod was deleted, node is going down)

In all cases, any long-running containers will be terminated in reverse phase order.

### Notes/Constraints/Caveats

#### InitContainers

The existing `initContainers` field must be respected, including the fact that initContainers are started and completed in serial.

We could define that `initContainers` retains its current semantics and is executed as the first part of the AppRun phase. This seems to offer good compatibility, and users who need more robust semantics can move containers from `initContainers` into the AppInit phase. We could even deprecate (but not remove) `initContainers` in favor of AppInit and document it as such.

Similarly, we could define that `initContainers` retains its current semantics and is executed as the first part of the AppInit phase.

We could merge `initContainers` with the AppInit phase. Containers defined there have more limited semantics than containers defined in AppInit, but should be strictly compatible with this proposal.

Considered and rejected: If the API defines a list of containers for each phase, `initContainers` could simply become the AppInit phase. This would either change the semantics of `initContainers` (serial start, run-to-completion), or would make AppInit different from other phases with regards to sequencing.

#### RestartPolicy

Adding a container-scoped `restartPolicy` raises the question of what the pod-scoped `restartPolicy` field means. For pods that do not specify container-scoped policy, the pod-scoped policy can be used. This ensures that existing pods do not change semantics.

Beyond that, though, we can perhaps use the pod-scoped policy as an “outer loop”. If a phase fails, the pod-scoped policy can suggest how to respond. For example, if a container in the ClusterEnv phase fails (perhaps it was retried several times and some heuristic or config decides it is not going to be retried), we can choose to restart the entire pod from the beginning.


#### Pods with RestartPolicy Never

Pods with RestartPolicy Never will never even start non-sidecar containers if
sidecar containers crash. That is because the sidecar is not restarted due to
the pod RestartPolicy Never and non-sidecar containers will only start once
sidecar are up and in a ready state. As sidecars are not ready, non-sidecars are
not started.

The most common use case is for jobs. If the sidecar crashes and the policy is
to never restart, the pod will be "stalled" with no way to move forward.

This is quite similar to what happens today if a job (or a pod with
RestartPolicy Never) has more than one container and some crashes: those are not
restarted. Therefore is considered a known caveat.

#### Modify killContainer() to enforece gracePeriodOverride on preStop hooks

The behaviour of `killContainer()` ignoring `gracePeriodOverride` for preStop
hooks was discussed in [this issue][issue-gracePeriodOverride]. After more
investigation it seems that gracePeriodOverride is unused, so modifications to
enforce the time on preStop hooks too seems safe.

The modification can live under the feature gate and if concerns arise a new
parameter can be added to `killContainer()` to take note of the time left to
kill the containers and leave gracePeriodOverride untouched.

The param seems unused, as mentioned in the [issue
comment][issue-gracePeriodOverride-comment], because I can't find any call to
`killContainer()` that set it to a non-nil value (sometimes it is set to the
same value another parameter has, but that parameter ends up always being nil).
Additionally, looking at the [tests for the function][tests-for-func] those were
created in commit 25bc76dae4cf and always set DeletionGracePeriodSeconds and
TerminationGracePeriodSeconds to the same value used as gracePeriodOverride.
Therefore, it seems like a using smaller gracePeriodOverride than those values
(DeletionGracePeriodSeconds/TerminationGracePeriodSeconds) was overlooked rather
than an intended change.

Furthermore, changing `killContainer()` to enforce the gracePeriodOverride on
preStop hooks doesn't break the unit tests either.

[issue-gracePeriodOverride]: https://github.com/kubernetes/kubernetes/issues/92432
[issue-gracePeriodOverride-comment]: https://github.com/kubernetes/kubernetes/issues/92432#issuecomment-648259349
[tests-for-func]: https://github.com/kubernetes/kubernetes/blob/47c450776f2731955ee7a4e8cc7ec1b4b6f14851/pkg/kubelet/kuberuntime/kuberuntime_container_test.go#L310-L322

#### Time to kill a pod increased by 4 seconds in the worst case

Currently the shutdown sequence of a pod looks like this **if preStop hooks never
finish**:
 1. Execute prestop hooks until pod.TerminationGracePeriodSeconds
 1. Send SIGTERM, wait for 2 seconds. Send SIGKILL to containers that didn't
    exit.

When sidecar containers are used, the shutdown sequence looks like **if preStop
hooks never finish**:

1. non-sidecars containers: execute preStop hooks until pod.TerminationGracePeriodSeconds
1. non-sidecar containers: Send SIGTERM, wait for 2 seconds. Send SIGKILL to containers that didn't
    exit.
1. Sidecar containers: execute preStop hooks with 2 seconds grace period
1. sidecar containers: Send SIGTERM, wait for 2 seconds. Send SIGKILL to containers that didn't
    exit.

This means that in some cases where one step in the shutdown sequence is
stalled, the following steps are given 2 seconds to execute.  As there are 2
more steps in the shutdown sequence when using sidecars, it can take 4 more
seconds to finish.

#### Enforce the startup/shutdown behavior only on startup/shutdown

One of the goals of this KEP is to **only modify the startup/shutdown
sequence**. This makes the semantics clear and helps to have clean code, as only
those places will be changed.

However, one side effect is that, for example, after pod startup has been
completed and sidecar and non-sidecar started, if all containers happen to crash
at the same time, all _can_ be restarted at the same time (if the restart policy
allows), as it is not done during startup of the pod. This might be surprising,
as instances of all the containers being started at the same time can be seen by
the users.

We believe this is fine, though, as the goal is to not change the behaviour
other than startup/shutdown and this edge case should be handled by users, as
any other container crashes.

If this behaviour is not welcome, however, code probably can be adapted to
handle the case when all containers crashed differently.

### Risks and Mitigations

TODO

## Design Details

TODO

### Proof of concept implementations


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
- 5th October 2020: Use cases [collection][use cases] for sidecar lifecycles started.
- 6th November 2020: @thockin proposes a new approach leveraging [phases][tim proposal].
- 17th March 2021: @matthyx takes over the KEP on behalf of @rata using Tim's [proposal][tim proposal] as a starting point.

[stalled]: https://github.com/kubernetes/enhancements/issues/753#issuecomment-597372056
[use cases]: https://docs.google.com/document/d/1Drw9C_Ljpcr4X9UPLvms1fn8uMRnTfJLb-xipgX4C1M
[tim proposal]: https://docs.google.com/document/d/1Q3685Ic2WV7jPo9vpmirZL1zLVJU91zd3_p_aFDPcS0

## Alternatives

### Alternative designs considered

This section contains ideas that were originally discussed but then dismissed in favour of the current design.
It also includes some links to related discussion on each topic to give some extra context, however not all decisions are documented in Github prs and may have been discussed in sig-meetings or in slack etc.

#### Add a pod.spec.lifecycle.type=sidecar,  implementation and Demo

There is a [PR here](https://github.com/kubernetes/kubernetes/pull/75099) with a working Proof of concept for this, it's just a draft but should help illustrate what these changes would look like.

Please view this [video](https://youtu.be/4hC8t6_8bTs) if you want to see what the PoC looks like in action.

#### Using pod annotations

Another implementation using pod annotations on top of Kubernetes 1.17 is
available [here][kinvolk-sidecar-branch].

There are some example yamls in the [sidecar-tests
folder][kinvolk-poc-sidecar-test], also the yaml output was captured to easily
see the behavior. [See the commit that created
them][kinvolk-poc-sidecar-test-commit] for instruction on how to run it.

Some other things worth noting about the implementation:
 * It is done using pod annotations so it is easy to test for users (doesn't
   modify pod spec)
 * Wasn't updated to call preStop hooks one time only, as this KEP now proposes
 * There is some c&p code to avoid doing refactors and just have a patch that is
   simpler to cherry-pick on different Kubernetes versions.

[kinvolk-poc-sidecar-prestop]: https://github.com/kinvolk/kubernetes/blob/52a96112b3e7878740a0945ad3fc4c6d0a6c5227/pkg/kubelet/kuberuntime/kuberuntime_container.go#L851
[kinvolk-sidecar-branch]: https://github.com/kinvolk/kubernetes/tree/rata/sidecar-ordering-annotations-1.17
[kinvolk-poc-sidecar-test]: https://github.com/kinvolk/kubernetes/tree/52a96112b3e7878740a0945ad3fc4c6d0a6c5227/sidecar-tests
[kinvolk-poc-sidecar-test-commit]: https://github.com/kinvolk/kubernetes/commit/385a89d83df9c076963d2943507d1527ffa606f7

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

Service mesh, today, have to do the following workarounds due to lack of
startup/shutdown ordering:
 * Recommend users to delay starting their apps by using a script to wait for
   the service mesh to be ready.
 * Blackhole all the traffic until service mesh container is up (usually using
   an initContainer for this)
 * Some trickery (sleep preStop hooks or some alternative) to not be killed
   before other containers that need to use the network. Otherwise, traffic for
   those containers will be blackholed

This means that if the app container is started before the service mesh is
started and ready, all traffic will be blackholed and the app needs to retry.
Once the service mesh container is ready, traffic will be allowed. A similar
problem happens for shutdown: if the service mesh container is killed first, the
network is down for the rest of the containers in the pod.

This has another major disadvantage: several apps crash if traffic is blackholed
during startup (common in some rails middleware, for example) and have to resort
to some kind of workaround, like [this one][linkerd-wait] to wait. This makes
also service mesh miss their goal of augmenting containers functionality without
modifying the main application.

This KEP addresses these 3 problems just listed when initContainer are not used
by the application. If initContainers are used, the first and the last problem
are solved only. In other words, traffic might still be blackholed for
initContainers that run after the service mesh iptables rules are inserted.

Such rules are usually inserted as an initContainer (trying to run last, to
avoid blackholing traffic to other initContainers) or alternatively, in the case
of Istio, using a [CNI plugin][istio-cni-opt]. When using the CNI plugin all
traffic from initContainers will be blackholed.

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
sidecar that adds some metadata, for example, or just uses a simple sidecar
(like file uploading after containers finish in a Job, or create/update some
files in shared emptyDir volumes, etc.).

While this is an option, is not possible or extremely costly for several use
cases.
