---
title: Allow StatefulSets to surge if pods are terminating longer than DeletionGracePeriodSeconds.
authors:
  - "@yanchenko-igor"
owning-sig: sig-apps
participating-sigs:
  - sig-scheduling
  - sig-node
reviewers:
  - @mattfarina
  - @kow3ns
approvers:
  - @mattfarina
  - @kow3ns
editor: TBD
creation-date: 2020-06-16
last-updated: 2020-06-16
status: provisional
see-also:
replaces:
superseded-by:
---

# Allow StatefulSets to surge if pods are terminating longer than DeletionGracePeriodSeconds.

## Table of Contents

<!-- toc -->
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
  - [Proposal](#proposal)
    - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Workload Implications](#workload-implications)
    - [Risks and Mitigations](#risks-and-mitigations)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

StatefulSets need to make sure that the pod is completly shut down in order to start a new one with the same name, we need to allow StatefulSets to surge pods, in case we don't have confirmation that the pod is shutdown, but deletion grace perion is already expired, this will allow us to have the same amount of alive pods as requested without violating pod safety.

## Motivation

In case when a node fails StatefulSets need to receive a confirmation about pods being shut down, this is not possible until the node returns, the pods which were running on the failed node become terminating and stay in this state until the node is recovered. We would like to make sure we always have the enough pods running.

It should be possible to surge extra pods while we are waiting for the terminating pod(s) to return.

### Goals

- Add support for Surge to the StatefulSet if pods stuck in terminating state.

## Proposal

### Implementation Details/Notes/Constraints


To implement this we would a boolean property to StatefulSetSpec AllowSurgeOnTerminationTimeout, and if the property is set to true, we would check if there are pods stuck in terminating state and then increase StatefulSet replicas by amount of pods that are longer in terminating state then expected.

```
// A StatefulSetSpec is the specification of a StatefulSet
type StatefulSetSpec struct {
        //EXISTING CODE
        // Replicas is the desired number of replicas of the given Template.
        // These are replicas in the sense that they are instantiations of the
        // same Template, but individual replicas also have a consistent identity.
        // If unspecified, defaults to 1.
        // TODO: Consider a rename of this field.
        // +optional
        Replicas int32

        //NEW CODE
        AllowSurgeOnTerminationTimeout bool

        //REST OF THE CODE

```


### Workload Implications

The main workload for surging extra pods is to maintain the same amount of pods as was requested, this is especially critical when replicas is set to 1, the service provided by statefulset won't be available until the failed node is back.

AllowSurgeOnTerminationTimeout will help to minimize distruption on problems with the pod, the new pods will be created on available nodes.

### Risks and Mitigations

The primary risk that in some condition we will have more running pods then it's defined by replicas.

# Alternatives

The alternative would be to have a dedicated worker that counts pods stuck in terminating and automatically updates the replicas count, this option also requires `.spec.podManagementPolicy: Parallel` as we would be waiting for the pod to be in status Ready if `OrderedReady`(default) is set.
