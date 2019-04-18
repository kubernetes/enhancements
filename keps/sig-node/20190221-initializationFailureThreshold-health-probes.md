---
title: Add initializationFailureThreshold to health probes
authors:
  - "@matthyx"
owning-sig: sig-node
participating-sigs:
  - sig-apps
  - sig-architecture
reviewers:
  - @RobertKrawitz
approvers:
  - @RobertKrawitz
editor: TBD
creation-date: 2019-02-21
last-updated: 2019-04-12
status: provisional
see-also:
replaces:
superseded-by:
---

# Add initializationFailureThreshold to health probes

## Table of Contents

- [Add initializationFailureThreshold to health probes](#add-initializationFailurethreshold-to-health-probes)
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
  - [Implementation History](#implementation-history)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Release Signoff Checklist

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
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

This proposal adds a numerical option `initializationFailureThreshold` to probes allowing a greater number of failures during the initial start of the container before taking action, while keeping `failureThreshold` at a minimum to restart deadlocked containers after an acceptable delay.

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
  - Introduce and explain this new option.
  - Document that `kubelet` does not save states, and what are the implications with this new option (see Risks and Mitigations).
  - Document appropriate use cases for this new option.

### Non-Goals

- This proposal does not address the issue of pod load affecting startup (or any other probe that may be delayed due to load). It is acting strictly at the pod level, not the node level.
- This proposal will only update the official Kubernetes documentation, excluding [A Pod's Life] and other well referenced pages explaining probes.

[A Pod's Life]: https://blog.openshift.com/kubernetes-pods-life/

## Proposal

### Implementation Details

The proposed solution is to add a new `int32` field named `InitializationFailureThreshold` to the type `Probe` of the core API, which is mapped to the numerical option `initializationFailureThreshold`.

It also requires keeping the state of the container (has the probe ever succeeded?) using a boolean `hasInitialized` inside the kubelet worker.

The combination of `hasInitialized` false and `resultRun` count lower than `initializationFailureThreshold` becomes another condition to return `true` to the probe state in `worker.go`.

For example, if `periodSeconds` is 10, `initializationFailureThreshold` is 20, and `failureThreshold` is 3, it means that:

- The kubelet will allow the container 200 seconds to start (20 probes, spaced 10 seconds apart).
- If a probe succeeds at any time during that interval, the container is considered to have started, and `failureThreshold` is used thereafter.

This means that all these cases will lead to a container being terminated:

- The container fails 20 probes at startup. It is considered to have failed, and is terminated after 200 seconds of downtime.
- The container fails 10 probes at startup, starts successfully, and after a long time fails 3 probes. The container is considered to have failed and is terminated after 30 seconds of downtime.
- The container fails 10 probes at startup, succeeds once, and fails 3 more probes. The container is considered to have started at 100 seconds, and even though it is still within the first 200 seconds of its lifetime covered by `initializationFailureThreshold`, it is considered to have failed because of `failureThreshold` and the fact that it had initially started successfully, therefore it is terminated after 30 seconds of downtime.

If `initializationFailureThreshold` is set smaller than `failureThreshold`, it's value is overridden to `failureThreshold` to avoid having the container being killed faster during startup than it would be in case of a deadlock, rendering is permanently unavailable.

This is being implemented and reviewed in PR [#71449].

[#71449]: https://github.com/kubernetes/kubernetes/pull/71449

### Risks and Mitigations

#### Stateless kubelet

`kubelet` handles container/pod lifecycle-related functions by relying on the underlying container runtime to persist the states. For probes and/or lifecycle hooks, kubelet rely on in-memory states only.

This means that the boolean `hasInitialized` as well as all probe counters could be reset during the lifetime of a container. There are several cases to consider:

- The container is starting: `hasInitialized` was still false, which means the container will have more probe attempts to successfully start. This is also the case today with a long `initialDelaySeconds`, except that with the new feature `failureThreshold` is taken into account after the first success of the probe.
- The container is running: `hasInitialized` is reverted to false until the next probe which will succeed and reset it to true immediately after.
- The container is deadlocked: `hasInitialized` is reverted to false, which means the container will have a maximum downtime of `periodSeconds` times `initializationFailureThreshold`. This is also the case today with a high `failureThreshold`.

## Design Details

### Test Plan

The following test cases can be covered by calling the `fakeExecProber` a number of times to verify:

- the container is killed after `initializationFailureThreshold` if it has never initialized (emulated by always calling `fakeExecProber{probe.Success, nil}`)
- the container is killed after `failureThreshold` once it has initialized (emulated by calling `fakeExecProber{probe.Success, nil}` once) with total probes > `initializationFailureThreshold`
- the container is killed after `failureThreshold` once it has initialized (emulated by calling `fakeExecProber{probe.Success, nil}` once) with total probes < `initializationFailureThreshold`

## Implementation History

- 2018-11-27: prototype implemented in PR [#71449] under review
- 2019-03-05: present KEP to sig-node
- 2019-04-11: open issue in enhancements [#950]

[#950]: https://github.com/kubernetes/enhancements/issues/950