# KEP-2008: Forensic Container Checkpointing

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation](#implementation)
    - [CRI Updates](#cri-updates)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Future Enhancements](#future-enhancements)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha to Beta Graduation](#alpha-to-beta-graduation)
    - [Beta to GA Graduation](#beta-to-ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Provide an interface to trigger a container checkpoint for forensic analysis.

## Motivation

Container checkpointing provides the functionality to take a snapshot of a
running container. The checkpointed container can be transferred to another
node and the original container will never know that it was checkpointed.

Restoring the container in a sandboxed environment provides a mean to
forensically analyse a copy of the container to understand if it might
have been a possible threat. As the analysis is happening on a copy of
the original container a possible attacker of the original container
will not be aware of any sandboxed analysis.

### Goals

The goal of this KEP is to introduce *checkpoint* to the CRI API.
This includes extending the *kubelet* API to support checkpointing single
containers with the forensic use case in mind.

### Non-Goals

Although *checkpoint* and *restore* can be used to implement container
migration this KEP is only about enabling the forensic use case. Checkpointing
a pod is not part of this proposal and left for future enhancements.

## Proposal

### Implementation

For the forensic use case we want to offer the functionality to checkpoint a
container out of a running Pod without stopping the checkpointed container or
letting the container know that it was checkpointed.

The corresponding code changes for the forensic use case can be found in the
following pull request:

* https://github.com/kubernetes/kubernetes/pull/104907

The goal is to introduce *checkpoint* and *restore* in a bottom-up approach.
In a first step we only want to extend the CRI API to trigger a checkpoint
by the container engine and to have the low level primitives in the *kubelet*
to trigger a checkpoint. It is necessary to enable the feature gate
`ContainerCheckpoint` to be able to checkpoint containers.

In the corresponding pull request a checkpoint is triggered using the *kubelet*
API:

```
curl -skv -X POST "https://localhost:10250/checkpoint/default/counters/wildfly"
```

For the first implementation we do not want to support restore in the
*kubelet*. With the focus on the forensic use case the restore should happen
outside of Kubernetes. The restore is a container engine only operation
in this first step.

#### CRI Updates

The CRI API will be extended to introduce one new RPC:
```
    // CheckpointContainer checkpoints a container
    rpc CheckpointContainer(CheckpointContainerRequest) returns (CheckpointContainerResponse) {}
```
with the following parameters:
```
message CheckpointContainerRequest {
    // ID of the container to be checkpointed.
    string container_id = 1;
    // Location of the checkpoint archive used for export/import
    string location = 2;
}

message CheckpointContainerResponse {}
```

### User Stories

To analyze unusual activities in a container, the container should
be checkpointed without stopping the container or without the container
knowing it was checkpointed. Using checkpointing it is possible to take
a copy of a running container for forensic analysis. The container will
continue to run without knowing a copy was created. This copy can then
be restored in another (sandboxed) environment in the context of another
container engine for detailed analysis of a possible attack.

### Risks and Mitigations

In its first implementation the risks are low as it tries to be a CRI API
change with minimal changes to the kubelet and it is gated by the feature
gate `ContainerCheckpoint`.

## Design Details

The feature gate `ContainerCheckpoint` will ensure that the API
graduation can be done in the standard Kubernetes way.

A kubelet API to trigger the checkpointing of a container will be
introduced as described in [Implementation](#implementation).

Also see https://github.com/kubernetes/kubernetes/pull/104907 for details.

### Future Enhancements

The initial implementation is only about checkpointing specific containers
out of a pod. In future versions we probably want to support checkpointing
complete pods. To checkpoint a complete pod the expectation on the container
engine would be to do a pod level cgroup freeze before checkpointing the
containers in the pod to ensure that all containers are checkpointed at the
same point in time and that the containers do not keep running while other
containers in the pod are checkpointed.

One possible result of being able to checkpoint and restore containers and pods
might be the possibility to migrate containers and pods in the future as
discussed in [#3949](https://github.com/kubernetes/kubernetes/issues/3949).

### Test Plan

For alpha:
- Unit tests available

For beta:
- CRI API changes need to be implemented by at least one
  container engine
- Enable e2e testing

### Graduation Criteria

#### Alpha

- [ ] Implement the new feature gate and kubelet implementation
- [ ] Ensure proper tests are in place
- [ ] Update documentation to make the feature visible

#### Alpha to Beta Graduation

At least one container engine has to have implemented the
corresponding CRI APIs to introduce e2e test for checkpointing.

- [ ] Enable the feature per default
- [ ] No major bugs reported in the previous cycle

#### Beta to GA Graduation

TBD

### Upgrade / Downgrade Strategy

No changes are required on upgrade if the container engine supports
the corresponding CRI API changes.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `ContainerCheckpoint`

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. By disabling the feature gate `ContainerCheckpoint` again.

###### What happens if we reenable the feature if it was previously rolled back?

Checkpointing containers will be possible again.

###### Are there any tests for feature enablement/disablement?

Currently no.

### Dependencies

CRIU needs to be installed on the node, but on most distributions it is already
a dependency of runc/crun. It does not require any specific services on the
cluster.

### Scalability

###### Will enabling / using this feature result in any new API calls?

The newly introduced CRI API call to checkpoint a container/pod will be
used by this feature. The kubelet will make the CRI API calls and it
will only be done when a checkpoint is triggered. No periodic API calls
will happen.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. It will only affect checkpoint CRI API calls.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

During checkpointing each memory page will be written to disk. Disk usage will increase by
the size of all memory pages in the checkpointed container. Each file in the container that
has been changed compared to the original version will also be part of the checkpoint.
Disk usage will overall increase by the used memory of the container and the changed files.
Checkpoint archive written to disk can optionally be compressed. The current implementation
does not compress the checkpoint archive on disk.

## Implementation History

* 2020-09-16: Initial version of this KEP
* 2020-12-10: Opened pull request showing an end-to-end implementation of a possible use case
* 2021-02-12: Changed KEP to mention the *experimental* API as suggested in the SIG Node meeting 2021-02-09
* 2021-04-08: Added section about Pod Lifecycle, Checkpoint Storage, Alternatives and Hooks
* 2021-07-08: Reworked structure and added missing details
* 2021-08-03: Added the forensic user story and highlight the goal to implement it in small steps
* 2021-08-10: Added future work with information about pod level cgroup freezing
* 2021-09-15: Removed references to first proof of concept implementation
* 2021-09-21: Mention feature gate `ContainerCheckpointRestore`
* 2021-09-22: Removed everything which is not directly related to the forensic use case
* 2022-01-06: Reworked based on review
* 2022-01-20: Reworked based on review and renamed feature gate to `ContainerCheckpoint`
* 2022-04-05: Added CRI API section and targeted 1.25
* 2022-05-17: Remove *restore* RPC from the CRI API

## Drawbacks

During checkpointing each memory page of the checkpointed container is written to disk
which can result in slightly lower performance because each memory page is copied
to disk. It can also result in increased disk IO operations during checkpoint
creation.

In the current CRI-O implementation the checkpoint archive is created so that only
the `root` user can access it. As the checkpoint archive contains all memory pages
a checkpoint archive can potentially contain secrets which are expected to be
in memory only.

The current CRI-O implementations handles SELinux labels as well as seccomp and restores
these setting as they were before. A possibly restored container is as secure as
before, but it is important to be careful where the checkpoint archive is stored.

During checkpointing CRIU injects parasite code into the to be checkpointed process.
On a SELinux enabled system the access to the parasite code is limited to the
label of corresponding container. On a non SELinux system it is limited to the
`root` user (which can access the process in any way).

## Alternatives

Another possibility to use checkpoint restore would be, for example, to trigger
the checkpoint by a privileged sidecar container (`CAP_SYS_ADMIN`) and do the
restore through an Init container.

The reason to integrate checkpoint restore directly into Kubernetes and not
with helpers like sidecar and init containers is that checkpointing is already,
for many years, deeply integrated into multiple container runtimes and engines
and this integration has been reliable and well tested. Going another way in
Kubernetes would make the whole process much more complicated and fragile. Not
using checkpoint and restore in Kubernetes through the existing paths of
runtimes and engines is not well known and maybe not even possible as
checkpointing and restoring is tightly integrated as it requires much
information only available by working closely with runtimes and engines.
