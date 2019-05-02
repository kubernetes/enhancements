---
title: Add pod-startup liveness-probe holdoff for slow-starting pods
authors:
  - "@matthyx"
owning-sig: sig-node
participating-sigs:
  - sig-apps
  - sig-architecture
reviewers:
  - @thockin
approvers:
  - @derekwaynecarr
  - @thockin
editor: TBD
creation-date: 2019-02-21
last-updated: 2019-05-02
status: implementable
see-also:
replaces:
superseded-by:
---

# Add pod-startup liveness-probe holdoff for slow-starting pods

## Table of Contents

- [Add pod-startup liveness-probe holdoff for slow-starting pods](#add-pod-startup-liveness-probe-holdoff-for-slow-starting-pods)
  - [Table of Contents](#table-of-contents)
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [Implementation Details](#implementation-details)
    - [Risks and Mitigations](#risks-and-mitigations)
      - [Stateless kubelet](#stateless-kubelet)
  - [Design Details](#design-details)
    - [Test Plan](#test-plan)
    - [Feature Gate](#feature-gate)
    - [Graduation Criteria](#graduation-criteria)
  - [Implementation History](#implementation-history)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Release Signoff Checklist

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [X] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

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
  - Document that `kubelet` does not save states, and what are the implications with this new probe (see Risks and Mitigations).
  - Document appropriate use cases for this new probe.

### Non-Goals

- This proposal does not address the issue of pod load affecting startup (or any other probe that may be delayed due to load). It is acting strictly at the pod level, not the node level.
- This proposal will only update the official Kubernetes documentation, excluding [A Pod's Life] and other well referenced pages explaining probes.
- This proposal does not propose to change the stateless nature of the kubelet.

[A Pod's Life]: https://blog.openshift.com/kubernetes-pods-life/

## Proposal

### Implementation Details

The proposed solution is to add a new probe named `startupProbe` in the container spec of a pod which will determine whether it has finished starting up.

It also requires keeping the state of the container (has the `startupProbe` ever succeeded?) using a boolean `hasStarted` inside the kubelet worker.

Depending on `hasStarted` the probing mechanism in `worker.go` might be altered:

- `hasStarted == true`: the kubelet worker works the same way as today
- `hasStarted == false`: the kubelet worker only probes the `startupProbe`

If `startupProbe` fails more than `failureThreshold` times, the result is the same as today when `livenessProbe` fails: the container is killed and might be restarted depending on `restartPolicy`.

If no `startupProbe` is defined, `hasStarted` is initialized with `true`.

### Risks and Mitigations

#### Stateless kubelet

`kubelet` handles container/pod lifecycle-related functions by relying on the underlying container runtime to persist the states. For probes and/or lifecycle hooks, kubelet rely on in-memory states only.

This means that the boolean `hasStarted` as well as all probe counters could be reset during the lifetime of a container. There are several cases to consider:

- The container is starting: `hasStarted` was already false, which means the `startupProbe` will continue to be checked and hold off other probes until it succeeds or fails completely.
- The container is running: `hasStarted` is reverted to false until the next `startupProbe` run which will succeed and reset it to true immediately after.
- The container is deadlocked: `hasStarted` is reverted to false, which means the container will have a maximum downtime of `periodSeconds` times `failureThreshold` (from the `startupProbe`). This is also the case today when you need to set a high `failureThreshold` for the `livenessProbe`.

The only way of killing the deadlocked container faster would be to have a stateful kubelet, which is not the purpose of this KEP.

## Design Details

### Test Plan

Unit tests will be implemented with `newTestWorker` and will check the following:

- proper initialization of `hasStarted` to false
- `hasStarted` becomes true as soon as `startupProbe` succeeds
- `livenessProbe` and `readinessProbe` are disabled until `hasStarted` is true
- `startupProbe` is disabled after `hasStarted` becomes true
- `failureThreshold` exceeded for `startupProbe` kills the container
- proper states are recovered after a kubelet restart

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

[#71449]: https://github.com/kubernetes/kubernetes/pull/71449
[#950]: https://github.com/kubernetes/enhancements/issues/950
[proposal]: https://github.com/kubernetes/kubernetes/issues/27114#issuecomment-437208330