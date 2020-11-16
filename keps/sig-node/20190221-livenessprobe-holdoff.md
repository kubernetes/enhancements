---
title: Add pod-startup liveness-probe holdoff for slow-starting pods
authors:
  - "@matthyx"
owning-sig: sig-node
participating-groups:
  - sig-apps
  - sig-architecture
reviewers:
  - "@thockin"
approvers:
  - "@derekwaynecarr"
  - "@thockin"
editor: TBD
creation-date: 2019-02-21
last-updated: 2020-09-16
status: implemented
see-also:
replaces:
superseded-by:
---

# Add pod-startup liveness-probe holdoff for slow-starting pods

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details](#implementation-details)
  - [Why a new probe instead of initializationFailureThreshold](#why-a-new-probe-instead-of-initializationfailurethreshold)
  - [Configuration example](#configuration-example)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Feature Gate](#feature-gate)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
  - [Version 1.16](#version-116)
  - [Version 1.17](#version-117)
  - [Version 1.18](#version-118)
  - [Version 1.19](#version-119)
  - [Version 1.20](#version-120)
<!-- /toc -->

## Release Signoff Checklist

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [X] KEP approvers have set the KEP status to `implementable`
- [X] Design details are appropriately documented
- [X] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] Graduation criteria is in place
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Slow starting containers are difficult to address with the current status of health probes: they are either killed before being up, or could be left deadlocked during a very long time before being killed.

This proposal adds a new probe called `startupProbe` that holds off all the other probes until the pod has finished its startup. In the case of a slow-starting pod, it could poll on a relatively short period with a high `failureThreshold`. Once it is satisfied, the other probes can start.

## Motivation

Slow starting containers here refer to containers that require a significant amount of time (one to several minutes) to start. There can be various reasons for this slow startup:

- long data initialization: only the first startup takes a lot of time
- heavy workload: every startups take a lot of time
- underpowered/overloaded node: startup times depend on external factors (however, solving node related issues is not a goal of this proposal)

The main problem with this kind containers is that they should be given enough time to start before having `livenessProbe` fail `failureThreshold` times, which triggers a kill by the `kubelet` before they have a chance to be up.

There are various strategies to handle this situation with the current API:

- Delay the initial `livenessProbe` sufficiently to permit the container to start up (set `initialDelaySeconds` greater than **startup time**). While this ensures no `livenessProbe` will run and fail during the startup period (triggering a kill), it also delays deadlock detection if the container starts faster than `initialDelaySeconds`. Also, since the `livenessProbe` isn't run at all during startup, there is no feedback loop on the actual startup time of the container.
- Increase the allowed number of `livenessProbe` failures until `kubelet` kills the container (set `failureThreshold` so that `failureThreshold` times `periodSeconds` is greater than **startup time**). While this gives enough time for the container to start up and allows a feedback loop, it prevents the container from being killed in a timely manner if it deadlocks or otherwise hangs after it has initially successfully come up.

However, none of these strategies provide an timely answer to slow starting containers stuck in a deadlock, which is the primary reason of setting up a `livenessProbe`.

### Goals

- Allow slow starting containers to run safely during startup with health probes enabled.
- Improve documentation of the `Probe` structure in core types' API.
- Improve `kubernetes.io/docs` section about Pod lifecycle:
  - Clearly state that PostStart handlers do not delay probe executions.
  - Introduce and explain this new probe.
  - Document appropriate use cases for this new probe.

### Non-Goals

- This proposal does not address the issue of pod load affecting startup (or any other probe that may be delayed due to load). It is acting strictly at the pod level, not the node level.
- This proposal will only update the official Kubernetes documentation, excluding [A Pod's Life] and other well referenced pages explaining probes.

[A Pod's Life]: https://blog.openshift.com/kubernetes-pods-life/

## Proposal

### Implementation Details

The proposed solution is to add a new probe named `startupProbe` in the container spec of a pod which will determine whether it has finished starting up.

It also requires keeping the state of the container (has the `startupProbe` ever succeeded?) using a boolean `Started` inside the ContainerStatus struct.

Depending on `Started` the probing mechanism in `worker.go` might be altered:

- `Started == true`: the kubelet worker works the same way as today
- `Started == false`: the kubelet worker only probes the `startupProbe`

If `startupProbe` fails more than `failureThreshold` times, the result is the same as today when `livenessProbe` fails: the container is killed and might be restarted depending on `restartPolicy`.

If no `startupProbe` is defined, `Started` is initialized with `true`.

### Why a new probe instead of initializationFailureThreshold

While trying to merge PR [#1014] in time for code-freeze, @thockin has make the following points which I agree with:

> I feel pretty strongly that something like a startupProbe would be net simpler to comprehend than a new field on liveness.
>
> In [issuecomment-437208330] we looked at a different take on this API - it is more precise in its meaning and rather than add yet another behavior modifier to probe, it can reuse the probe structure directly.

Here is the excerpt of [issuecomment-437208330] talking about the design:

> An idea that I toyed with but never pursued was a StartupProbe - all the other probes would wait on it at pod startup. It could poll on a relatively short period with a long FailureThreshold. Once it is satisfied, the other probes can start.

I also think the third probe gives more flexibility if we find other good reasons to inhibit `livenessProbe` or `readinessProbe` before something occurs during container startup.

[#1014]: https://github.com/kubernetes/enhancements/pull/1014
[issuecomment-437208330]: https://github.com/kubernetes/kubernetes/issues/27114#issuecomment-437208330

### Configuration example

This example shows how startupProbe can be used to emulate the functionality of `initializationFailureThreshold` as it was proposed before:

```yaml
ports:
- name: liveness-port
  containerPort: 8080
  hostPort: 8080

livenessProbe:
  httpGet:
    path: /healthz
    port: liveness-port
  failureThreshold: 1
  periodSeconds: 10

startupProbe:
  httpGet:
    path: /healthz
    port: liveness-port
  failureThreshold: 30 (=initializationFailureThreshold)
  periodSeconds: 10
```

## Design Details

### Test Plan

Unit tests will be implemented with `newTestWorker` and will check the following:

- proper initialization of `Started` to false
- `Started` becomes true as soon as `startupProbe` succeeds
- `livenessProbe` and `readinessProbe` are disabled until `Started` is true
- `startupProbe` is disabled after `Started` becomes true
- `failureThreshold` exceeded for `startupProbe` kills the container

E2e tests will also cover the main use-case for this probe:

- `startupProbe` disables `livenessProbe` long enough to simulate a slow starting container, using a high `failureThreshold`

### Feature Gate

- Expected feature gate key: `StartupProbeEnabled`
- Expected default value: `false`

### Graduation Criteria

- Alpha: Initial support for `startupProbe` added. Disabled by default.
- Beta: `startupProbe` enabled with no default configuration.
- Stable: `startupProbe` enabled with no default configuration.

## Implementation History

- 2018-11-27: prototype implemented in PR [#71449] under review
- 2019-03-05: present KEP to sig-node
- 2019-04-11: open issue in enhancements [#950]
- 2019-05-01: redesign to additional probe after @thockin [proposal]
- 2019-05-02: add test plan

### Version 1.16

- Implement `startupProbe` as Alpha [#77807]
- Cherry pick of #82747 [#83607]

### Version 1.17

- Fix `startup_probe_test.go` failing test [#82747]
- Add `startupProbe` result handling to kuberuntime [#84279]
- Clarify startupProbe e2e tests [#84291]

### Version 1.18

- Graduate `startupProbe` to Beta [#83437]
- Cherry pick of #92196 [#92477]

### Version 1.19

- Pods which have not "started" can not be "ready" [#92196]

### Version 1.20

- Graduate `startupProbe` to GA [#94160]

[#71449]: https://github.com/kubernetes/kubernetes/pull/71449
[#950]: https://github.com/kubernetes/enhancements/issues/950
[proposal]: https://github.com/kubernetes/kubernetes/issues/27114#issuecomment-437208330
[#77807]: https://github.com/kubernetes/kubernetes/pull/77807
[#82747]: https://github.com/kubernetes/kubernetes/issues/82747
[#83437]: https://github.com/kubernetes/kubernetes/pull/83437
[#83607]: https://github.com/kubernetes/kubernetes/pull/83607
[#84279]: https://github.com/kubernetes/kubernetes/pull/84279
[#84291]: https://github.com/kubernetes/kubernetes/pull/84291
[#92196]: https://github.com/kubernetes/kubernetes/pull/92196
[#92477]: https://github.com/kubernetes/kubernetes/pull/92477
[#94160]: https://github.com/kubernetes/kubernetes/pull/94160
