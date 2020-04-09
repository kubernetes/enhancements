---
title: pdb-support-for-custom-resources-with-scale-subresource
authors:
  - "@mortent"
owning-sig: sig-apps
participating-sigs:
  - sig-scheduling
  - sig-autoscaling
reviewers:
  - "@kow3ns"
  - "@janetkuo"
approvers:
  - "@kow3ns"
  - "@janetkuo"
editor: TBD
creation-date: 2019-04-12
last-updated: 2019-04-12
status: implementable
see-also:
replaces:
superseded-by:
---

# PDB support for custom resources with scale subresource

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

[Pod Disruption Budget (PDB)](https://kubernetes.io/docs/tasks/run-application/configure-pdb/)
is a Kubernetes API that limits the number of pods of a collection that are down simultaneously from voluntary disruptions. PDBs allows a user to specify the allowed disruption through either min available or max unavailable number of pods. In order to support PDBs where max number of unavailable pods is set, the PDB controller needs to be able to look up the desired number of replicas. It does this by looking at the controller(s) (as specified by the owner ref) for the pods covered by the PDB. Currently only four of the basic workload controllers are supported for PDBs, namely Deployment, StatefulSet, ReplicaSet and ReplicationController. 

The scale subresource allows any resource to specify its desired number of replicas and a generic way to look up this information. This document lays out a plan to use the scale subresource to allow setting PDBs for any resource implementing the scale subresource.

## Motivation

PDBs are an important tool to control the number of voluntary disruptions for workloads on Kubernetes. As more users start deploying custom controllers/operators based on CRDs (EtcdCluster, MySQLReplicaSet...), it is inconvenient that they cannot take advantage of PDBs. This doesn't work today because the PDB controller needs to know the desired number of replicas specified in a controller and the PDB controller only knows how to find this from the four Kubernetes workload controllers mentioned above. The scale subresource is already in use by the autoscaler and it provides a generic way to look up the desired number of replicas from any custom resource with a scale subresource. We can leverage this to support PDBs for custom controllers.

### Goals

- Implement support for scale subresources in PDBs
- Avoid major performance impact from the change

## Proposal

### Implementation Details/Notes/Constraints

In the current implementation, the PDB controller identifies the workload controller by going through all the pods covered by the PDB, and for each pod, check in sequence for each of the four workload controllers. Each check looks at the controller reference, and if the kind is correct, looks up the controller from the shared informer. The PDB controller then looks up the desired number of replicas from the identified controller. If the set of pods covered by the PDB identfies more than one controller, that is considered an error. 

The plan is to keep this approach, but check for the scale subresource if neither of the kubernetes workload controllers match. Each of the kubernetes workload controllers actually implement the scale subresource, so it is possible to only use that one. But since shared informers can not be used with the scale subresource, we want to rely on the informers for checking the kubernetes workload controllers, and only try the scale subresource if none of the them matches. This way the controller will only need to hit the apiserver in the less likely scenario where the pods have a controller other than the standard kubernetes workload controllers.

As mentioned above, the PDB controller will check the desired number of replicas from the controller for each pod covered by the PDB. In almost all scenarios the controller will be the same for all pods and therefore the desired number of replicas will be the same. This is not a concern for the kubernetes workload controllers since the controller uses shared informers. When using the scale subresource, this would lead to unnecessary calls to the apiserver. We will implement a per reconcile loop cache, so the PDB controller only needs to look up each controller once per reconcile loop.

### Risks and Mitigations

The major risk with this change is the additional load on the apiserver since we can't use shared informers for scale subresources. As described above, we plan to mitigate this by only checking the scale subresource when the kind of the controller ref doesn't match any of the kubernetes workload controllers, and use a cache to make sure we only need to look up each controller once per reconcile loop.

## Design Details

### Test Plan

* Unit tests covering the usage of PDBs with custom resources that is implementing the scale subresource.
* Integration tests to make sure using the scale subresource endpoint from the PDB controller works as expected.

### Graduation Criteria

This will be added as a beta enhancement to PDBs. It doesn't change the existing API or behvior but only adds an additional code path to handle non built-in types.

[KEP](https://github.com/kubernetes/enhancements/pull/904) for graduating PDBs to GA is already underway. It involves a change to make PDBs mutable. [PR](https://github.com/kubernetes/kubernetes/pull/69867) for this is almost ready to merge. The goal is to get both that change and this one into the next version of Kubernetes (1.15), and unless any serious issues come up, promote PDBs to GA the following release (1.16).

## Implementation History

- Initial PR: https://github.com/kubernetes/kubernetes/pull/76294

