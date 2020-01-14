---
title: Multi Scheduling Profiles
authors:
  - "@alculquicondor"
  - "@ahg-g"
owning-sig: sig-scheduling
reviewers:
  - "@Huang-Wei"
  - "@liggitt"
approvers:
  - "@Huang-Wei"
editor: TBD
creation-date: 2020-01-14
last-updated: 2020-01-14
status: provisional
see-also:
  - "/keps/sig-scheduling/20180409-scheduling-framework.md"
  - "/keps/sig-scheduling/20190226-default-even-pod-spreading.md"
---

# Multi Scheduling Profiles

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Component Config API](#component-config-api)
    - [Kube-Scheduler implementation](#kube-scheduler-implementation)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

As workloads in clusters become more heterogeneous, it is natural that they have
different scheduling needs.

We propose making the scheduler run different framework plugin configurations,
which we will call profiles and will be associated to a scheduler name.
Pods can choose to be scheduled under a particular configuration by setting the
scheduler name associated to it in its pod spec. They will continue to be
scheduled under the default configuration if they don't specify a scheduler
name. The scheduler will continue to schedule one pod at a time.

## Motivation

Clusters run a variety of workloads, which can be *broadly* classified as
services and batch jobs. Some users may chose to run only one class of workloads
in a cluster, so they can provide a reasonable configuration that suits their
scheduling needs.

However, users may choose to run more heterogeneous workloads in a single
cluster. They could have a multi-tenant cluster or they might want to take
advantage of under-utilized nodes. Or they could have a set of fixed nodes
and a set that auto-scales.

Pods can influence scheduling decisions with features such as node/pod affinity,
tolerations or (alpha) even pod spreading. But there are 2 problems:

- kube-scheduler also calculates a set of default scores that can compete with
  these requests.
- Authors of the workloads need to be aware of the cluster characteristics.

For this reason, some operators choose to run multiple schedulers, whether those
are different binaries or kube-scheduler with a different configuration. But
this setup might cause race conditions between the multiple schedulers, as they
might have a different view of the cluster resources at a given time.
Additionally, more binaries requires more management effort.

Instead, having a single kube-scheduler run multiple profiles can solve the
users' needs avoiding the problem of multiple schedulers.

### Goals

- Add support in the kube-scheduler's component config API for multiple
scheduling profiles.
- Make kube-scheduler schedule pods using different profiles given the scheduler
name specified in the pod spec.

### Non-Goals

- Introduce new default scheduling profiles.

## Proposal

### User Stories

#### Story 1

I have two types of workloads that I want to run in two different sets of nodes.
For one type of workload, I want them to spread in the topology. But for the
other type, I prefer them to get scheduled in a few nodes as possible.

### Implementation Details/Notes/Constraints

#### Component Config API

[`v1alpha1` component config](
https://github.com/kubernetes/kubernetes/blob/50deec2/pkg/scheduler/apis/config/types.go#L46)
looks like the following:

```go
type KubeSchedulerConfiguration struct {
   ...
   SchedulerName string
   HardPodAffinitySymmetricWeight
   Plugins *Plugins
   PluginConfig []PluginConfig
   ...
}
```

We will introduce `v1alpha2`, with the following structure:

```go
type KubeSchedulerConfiguration struct {
   ...
   Profiles []KubeSchedulerProfile
}

type KubeSchedulerProfile struct {
   SchedulerName string
   Plugins *Plugins
   PluginConfig []PluginConfig
}
```

`HardPodAffinitySymmetricWeight` would be moved to be a `PluginConfig.Arg` for
the plugin `InterPodAffinity` as `HardPodAffinityWeight`.

During conversion from `v1alpha1` to `v1alpha2`, we will copy all the necessary
parameters from KubeSchedulerConfiguration into one item in the `profiles` list.

#### Kube-Scheduler implementation

1. At startup, kube-scheduler will process all the different profiles,
initialize framework instances for them and store them in a registry.
If no profile is included in the configuration, one will be instantiated with
the name `default-scheduler` using the default plugins.

2. When getting notified about unscheduled pods, kube-scheduler will check the
scheduler name in the registry. If the name is present, they will be added to
the scheduler queue.

3. When a new pod is taken from the queue, it will get scheduled using the
framework instance from the registry corresponding to the specified scheduler
name.

### Risks and Mitigations

Operators could introduce profiles that disable scheduling features exposed in
the Pod Spec. Fortunately, the framework's plugins configuration makes it easy
to [create custom configurations from the default](
https://github.com/kubernetes/kubernetes/blob/50deec2/pkg/scheduler/apis/config/types.go#L156-L160)
through its `enabled` and `disabled` lists. 
However, we should discourage the use of `*` to disable all plugins in
the scheduler documentation.

## Design Details

### Test Plan

TODO

### Graduation Criteria

##### Alpha -> Beta Graduation

TODO

##### Beta -> GA Graduation

TODO

## Implementation History

- 2020-01-14: Initial KEP sent out for review, including Summary, Motivation
and Proposal
