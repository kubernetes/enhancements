# KEP-2008: Add checkpoint and restore to the API

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation](#implementation)
  - [Pod Checkpoint Restore Lifecycle](#pod-checkpoint-restore-lifecycle)
    - [Pod Checkpointing](#pod-checkpointing)
    - [Pod Checkpoint Archive Distribution](#pod-checkpoint-archive-distribution)
    - [Pod Restore](#pod-restore)
    - [Pod Checkpoint Trigger](#pod-checkpoint-trigger)
  - [Checkpoint Storage](#checkpoint-storage)
  - [Hooks](#hooks)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
    - [Removing a Deprecated Flag](#removing-a-deprecated-flag)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
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

This KEP tries to be a first step towards container migration. In its simplest
form container migration is the process of saving the state of a running
container to disk, transferring it to the migration destination and restarting
the container from the saved state. To enable container migration the minimal
primitives *checkpoint* and *restore* are needed independent of all other
things. Therefore this KEP proposes to extend the API to provide a
*checkpoint* and a *restore* interface.

Additionally, everything mentioned here relies on [CRIU](https://criu.org) and
its integration in runc or crun. In theory it should be independent of the
actual checkpoint/restore tool used by the OCI runtime, but currently CRIU
seems to be the only tool capable of what is needed for container migration.

## Motivation

The motivation to write this KEP to add checkpoint and restore to the API
is definitely to come a step closer to container migration. As mentioned
above this explicitly is only about checkpoint and restore to keep it simple.
Container migration is the motivation for this KEP but it is not the goal.

Container migration is also on the wishlist of the community since January
2015 as seen in [#3949](https://github.com/kubernetes/kubernetes/issues/3949)
and its possible uses cases can be put in the following categories:

 * Stateful reboot: This enables the user to update a critical
   component (e.g. the kernel for a security update) without shutting down
   containers. Containers are checkpointed, system can be rebooted and
   containers are then restored from the checkpoint.

 * Fast startup: In order to quickly startup containers it is possible
   to start containers from checkpoints. This is especially interesting
   for containers that require a long time to initialize. Either for
   loading many libraries or for filling large caches.

 * Container migration: Migrate a running container from one host to
   another host without losing the container state.

Container migration can be seen in different container environments. The following
gives an overview of CRIU integration to support container migration. As this
KEP would not be possible without CRIU OpenVZ needs to be mentioned:

 * CRIU has been written to support container migration in OpenVZ, which
   ensures that CRIU based migration has been designed with containers
   in mind:
   * https://wiki.openvz.org/Checkpointing_and_live_migration

 * LXC/LXD also provides the possibility to migrate containers from one
   host to another: `lxc move <container> <remote>:<container>`

   In addition to the simplest form of container migration (checkpoint,
   transfer, restore) LXD also supports optimizations to decrease the
   downtime during migration by using CRIU's pre-copy migration support.

   * https://archive.fosdem.org/2018/schedule/event/containers_optimized_migration/
   * https://lisas.de/~adrian/posts/2017-Dec-06-optimizing-live-container-migration-in-lxd.html

 * Borg uses CRIU to live migrate containers between hosts to free up resources
   on hosts which are missing resources under load:

   * [Task Migration at Google Using CRIU](https://www.linuxplumbersconf.org/event/2/contributions/209/)
   * [Update on Task Migration at Google Using CRIU](https://linuxplumbersconf.org/event/4/contributions/508/)

   See especially the second presentation for limitations Google is
   experiencing using CRIU based container migration in production.

 * Podman supports container migration in its simplest form (checkpoint,
   transfer, restore): https://criu.org/Podman

The motivation is to get closer to container migration by taking the first
step and providing support for simple checkpoint and restore. The motivation
is existing checkpoint/restore/migration support in other container
environments showing that it is useful and production ready.

### Goals

The goal of this KEP is to introduce *checkpoint* and *restore* to the API
by introducing a new *experimental* API.

### Non-Goals

Out of scope of this KEP are high level discussions about how to implement
container migration. This is only about low level primitives to add
*checkpoint* and *restore* to the API.

## Proposal

### Implementation

There are already draft pull requests opened to show a possible
implementation of this proposal:

* https://github.com/cri-o/cri-o/pull/4199
* https://github.com/kubernetes-sigs/cri-tools/pull/662
* https://github.com/kubernetes/kubernetes/pull/97194
* https://github.com/kubernetes/kubernetes/pull/97689

During discussions with the SIG Node members one possible approach to
introduce *checkpoint* and *restore* to the CRI API is to provide a new
*experimental* API. The idea of the *experimental* API is to have an easy
way to introduce new features without the danger of breaking existing
functionality. The *experimental* API also makes it easier to rework an API
as the *experimental* API does not have to be stable.

The existing proof of concept implementation as seen in the listed pull
requests implements `kubectl drain --checkpoint`. This implementation
(https://github.com/kubernetes/kubernetes/pull/97194) shows what is necessary
to implement one of the possible *checkpoint* and *restore* use cases
(*stateful reboot*). Instead of stopping the containers to drain a node, the
containers are checkpointed and all state is kept. After draining the node the
containers will be automatically restored by the kubelet.

The reason to implement the drain use case was that it seemed to be the
most simple *checkpoint* and *restore* use case. It is not triggered
by some policy or scheduling decision but only by the user wanting to
drain a node.

For easier review of the actual changes to the newly introduced
*experimental* API a second pull request exists with only the
changes to the API: https://github.com/kubernetes/kubernetes/pull/97689

The proof of concept CRI-O pull request is based on the Podman
checkpoint/restore implementation which can be seen as a preparation for
this KEP. Especially as Podman uses CNI for network configuration
it required changes to CRIU and runc to allow restoring into a
previously set up network namespace instead of letting CRIU handle
the network namespace restore:

* https://github.com/opencontainers/runc/pull/1849

This work lead to CRIU's support to allow restoring processes
out of and into PID namespaces:

* https://github.com/checkpoint-restore/criu/pull/1056
* https://github.com/opencontainers/runc/pull/2525

As well as the runc support to allow restoring into all kinds of
namespaces: https://github.com/opencontainers/runc/pull/2583

The support to allow restoring into all kinds of namespaces is
especially important when checkpointing and restoring containers
out of and into pods which is possible with the proposed CRI-O changes.

### Pod Checkpoint Restore Lifecycle

#### Pod Checkpointing

A Pod can be checkpointed at any time during its lifetime.  The checkpointing
can either be triggered by `kubectl` (for user triggered checkpoints) or
through the Kubernetes API (for controller triggered checkpoints). The actual
checkpointing is done by the container engine (CRI-O in the existing proof of
concept) which calls the container runtime (runc or crun). The container
runtime writes the checkpoint data and metadata to a directory specified by the
kubelet.

The container engine creates a tar archive including the actual checkpoint
(from CRIU) and the file system difference to the OCI image the container is
based on. With this tar archive, containing the CRIU checkpoint and the file
system differences, the container can be restored on any other CRI-O
instance. In the current proof of concept the container engine automatically
pulls missing OCI images from the registry. This means that the checkpoint
archive contains all necessary information to restore a checkpointed
container on any system that has access to a registry with the base image.
The file system differences from the checkpoint archive will be applied and
the container runtime will restore the container on any other host.

A Pod checkpoint is then created by the container engine by combining all
container checkpoint archives (one for each container in the Pod) into a Pod
checkpoint tar archive.

The default location for the Pod checkpoint archive is
`/var/lib/kubelet/checkpoints` and once the container engine has written the
Pod checkpoint archive, the kubelet will add additional metadata. In the
current proof of concept this metadata includes Pod information like the
`RestartCounter`, Pod UID, `TerminationGracePeriod`, containers in the Pod,
used IP addresses and labels. All this metadata is needed during restore to
prepare the Pod before the actual restore.

When looking at the container engine level (see *Implementation* section for
corresponding CRI-O pull requests) checkpointing can happen on the container
level as well as on the Pod level. Checkpointing a Pod will result in all
container being checkpointed with additional metadata for the Pod. At the
container engine level it is possible to checkpoint a single container out of a
Pod or all containers at the same time.

During restore it is also possible to restore one or more containers into
existing Pods or it is possible to restore a complete Pod with all its
containers.

If a Pod with all its containers is checkpointed the containers will be
checkpointed in the order the container engine returns the list of containers
in that Pod to the internal checkpointing function.

During checkpointing the container runtime tells CRIU which cgroup the
container is running in and CRIU will use the cgroup freezer to stop all
processes in that cgroup. All processes will be paused during checkpointing of
a single container, but between checkpointing the complete Pod the remaining,
not yet checkpointed, containers will continue to run.  To be able to have a
Pod wide coordinated checkpointing, the complete Pod must be paused before
initiating the Pod checkpoint. The time required to quiesce a container is the
time the cgroup freezer requires to pause all processes in the corresponding
cgroup.

It is not possible to checkpoint processes or containers which directly access
a device like GPU, InfiniBand or SRIOV, because CRIU cannot extract the
hardware state from those devices and therefore CRIU is not able to correctly
restore the complete state of the process and container.  Such containers will
automatically be excluded from checkpointing.

All processes and containers can only be restored with the same UID and PID. It
is not easily possible to change any properties of the restored processes. This
also means that any file ownership must be the same on restore. External
mountpoints cannot have different file owners. As service accounts, from CRIU's
point of view, are just bind mounts from an external source it is possible to
change service accounts between checkpoint and restore by either having
different content at the service account location or by using another external
service account location.

Anything concerning SELinux will be correctly restored. Upon checkpointing CRIU
will record the SELinux process labels and mount labels. In single container
use cases like Podman the restored container will have the same process labels
and mount labels. In the Pod use case the restored processes and the restored
mount points will get the corresponding SELinux labels of the Pod
infrastructure container.

In the current proof of concept implementation (see corresponding PRs)
checkpointing is a subresource on the pod API and mainly following the code
flow of the `/drain` API where possible. This also means that the existing
proof of concept implementation does not support single container checkpoints
yet on the Kubernetes level. It is, however, possible on the CRI-O level.

The goal is to offer the user the possibility to globally disable checkpointing
as well as to query the container engine if it supports checkpointing or not.

In contrast to the proof of concept pull request which shows a possible
implementation of checkpoint and restore and all possible layers of Kubernetes
the goal for the actual introduction of checkpoint restore in Kubernetes would
be bottom up. First on the CRI API level and kubelet and then wiring it through
to the higher layers until reaching `kubectl`.  This process would include a
re-evaluation on each layer if checkpoint restore needs to be exposed on higher
levels or not.  The start would be a kubelet local API and then depending on
community input extending it to higher layers or not.

Currently rootless checkpointing is not supported neither on the container
runtime level nor on the container engine level. But there have been discussion
in CRIU and the kernel and with the recent introduction of
`CAP_CHECKPOINT_RESTORE` rootless container checkpointing should be doable.

#### Pod Checkpoint Archive Distribution

The distribution aspect is solved by the use of image registries. We can
introduce an operator under kubernetes-sigs that is responsible for the
distribution of the checkpointed image from the local registry to
a configured in-cluster or off-cluster registry.

This would also intersect with the work in CRI to pin images that the
kubelet should ignore during GC giving time to move the image away
from local storage.

Checkpoint archives written to local disk are not encrypted but created with
mode 600 to avoid everyone having access to exported checkpoint archives.

Secrets mounted into the container are not part of the checkpoint as all
external mountpoints are not part of the checkpoint archive. Every memory page
and all containers internal `tmpfs` are, however, part of the checkpoint
archive any may content confidential data.

In the current proof of concept implementation the kubelet reads some
checkpoint metadata. Especially important is the list of the containers
which are checkpointed in the Pod. The kubelet needs this information
to prepare itself for containers being restored by the container engine.

Accessing the checkpoint archives can be outsourced to the container engine
and all necessary restore metadata could be queried from the container
engine via the CRI API.

A checkpoint archive needs to be stored for at least for a short moment on
the local disk even if it is distributed via a registry. This might
affect local disk accounting. The size of the checkpoint archive depends
on the memory used by the container processes and the differences of
the container file system when compared to the image the container is based
on.

#### Pod Restore

To restore a checkpointed Pod the corresponding checkpoint archive image is
pulled from the registry it has been previously pushed to or read from the
local file system. The checkpoint archive is referenced in the Pod
specification just like any other image.

The restore will use the information in the checkpoint archive (either from
the local file system or from a registry) to restore the Pod. This
includes Pod and container metadata as well as file system differences to
the base OCI image. Restoring a Pod will pull the OCI images the checkpoint
is based on automatically.

Currently there is no way of letting a process know that it was restored.
There are discussions about introducing an interface which can be used
to let applications know that they have been restored. There was also the
idea of an operator doing the restore notification.

The current idea is to not restore init containers. Maybe it makes sense
to mark init containers to run-on-restore in the future to offer the
possibility to have the init container concept on restore.

Currently ephemeral containers would be checkpointed if a Pod is
checkpointed just as any other container. It still needs to be decided
if ephemeral containers should always be excluded.

Restoring a single container into an existing pod is implemented in CRI-O
but not part of the proof of concept implementation on the Kubernetes
level.

The following is an example how a container could be checkpointed out of
a Pod and stored in a registry. Later that checkpoint image
(`registry/checkpoint1`) can then be used to create new Pods
based on the previously checkpointed container.
```
checkpoint container      ┌──────────────────────────────────┐
         │                │apiVersion: v1                    │
┌────────▼──────────┐     │kind: Pod                         │
│apiVersion: v1     │     │metadata:                         │
│kind: Pod          │     │  name: pod2                      │
│metadata:          │     │spec:                             │
│  name: pod1       │     │  containers:                     │
│spec:              │     │  - name: ctr1                    │
│  containers:      │     │    image: registry/checkpoint1   │
│  - name: ctr1     │     └────────▲─────────────────────────┘
│    image: image1  │              │
└────────┬──────────┘       restore│container from checkpoint
         │                         │
         ▼                ┌────────▼─────────────────────────┐
  checkpoint image        │apiVersion: v1                    │
         │                │kind: Pod                         │
         │                │metadata:                         │
         ▼                │  name: pod1                      │
  registry/checkpoint1    │spec:                             │
                          │  containers:                     │
                          │  - name: ctr1                    │
                          │    image: registry/checkpoint1   │
                          └──────────────────────────────────┘
```

If the container is restored on another machine it is necessary
to ensure that external Pod level and container level data is
available before restoring.

Container level data is part of the checkpoint image.

#### Pod Checkpoint Trigger

For initial simplicity checkpointing is a manual process (`kubectl`) for
now and not triggered by any scheduling or policy decision. The initial
proof of concept provides access to checkpointing via `kubectl` and the
API.

The current proof of concept implementation is just a starting point
and it might make sense to not make checkpointing available at multiple
different API endpoints because of security concerns. The right location
to expose checkpoint and restore interface is still open to discussions
and subject to change as this KEP is primarily aimed at introducing
the CRI API definitions.

### Checkpoint Storage

The size of a checkpoint archive is directly proportional to the size of memory
the processes in the container are using. All other data in the checkpoint is a
few kilobytes for each checkpointed process. Additionally the changes to the
container file system are also part of the checkpoint archive. All this is put
in a compressed (or uncompressed) tar archive.

The default location for the kubelet checkpoints is currently below `--root-dir`
and defaults to `/var/lib/kubelet/checkpoints`. This directory can be anywhere. A
local directory or especially for cluster setup where Pods will be migrated, a
network mounted storage.

To work with all possible storage backends the only requirement on the
checkpoint storage is that it can be used to store and retrieve the previously
mentioned Pod checkpoint archive.

Additionally there have been discussions to use the OCI image format to store
checkpoint archives. With this approach it would be possible to push a
checkpoint to a registry and also retrieve existing images from a registry.

Another possibility is to use a checkpoint image server which can be used to
stream checkpoints to and where they can be later retrieved from. This is,
however, not part of the current KEP and pull requests. Just mentioned as
additional information.

The actual checkpointing does not require much additional memory. CRIU
writes the pages directly to disk so that checkpointing should not be
problematic concerning cgroup memory limits.

### Hooks

A possible use case is fast startup of JVMs by starting from a checkpoint and
especially for the JVM use case we have been discussing the possibilities to
have a way to tell the application in the container that it is about to be
checkpointed. The JVM could drop certain caches to remove unnecessary
information from the checkpoint or also remove secrets from memory so that
these things are not included in the checkpoint.

Once a container has been restored it would also be helpful to let the
application know that it has been restored.

### User Stories

As this KEP is explicitly not about container migration the user stories
are not mentioning container migration based user stories. If this would
be helpful to understand this KEP better it can be easily added.

#### Story 1

Although containers are supposed to be stateless there are still containers
which either require some time to startup or which have data cached.
If a system is rebooted to update the kernel these containers can be
checkpointed. Once the system has been rebooted into the new kernel the
container can be restored and continue from its previous memory state
without the need to wait for long start up times or to reload cached data
from disk.

#### Story 2

Just as in Story 1 a stateful container needs to be moved to another pod.
Using checkpoint the container can be written to disk and using restore
the container can be restored in another pod while keeping its state and
all data already loaded into memory.

### Notes/Constraints/Caveats (Optional)

Not sure, but probably not applicable as it is only an API change at this
stage.

### Risks and Mitigations

Not sure, but probably not applicable as it is only an API change at this
stage. Especially as it is using the new *experimental* API it can be
easily reverted.

## Design Details

See Proposal and corresponding implementation pull requests.

### Test Plan

Not clear yet and not familiar enough with the existing tests, but happy to add
a test if possible.

### Graduation Criteria

Not clear yet and not familiar enough with the KEP process.

#### Alpha -> Beta Graduation

Not clear yet and not familiar enough with the KEP process.

#### Beta -> GA Graduation

Not clear yet and not familiar enough with the KEP process.

#### Removing a Deprecated Flag

Probably not required.

### Upgrade / Downgrade Strategy

Probably not required.

### Version Skew Strategy

Probably not required.

## Production Readiness Review Questionnaire

Not clear yet and not familiar enough with the KEP process.

## Implementation History

* 2020-09-16: Initial version of this KEP
* 2020-12-10: Opened pull request showing an end-to-end implementation of a possible use case
* 2021-02-12: Changed KEP to mention the *experimental* API as suggested in the SIG Node meeting 2021-02-09
* 2021-04-08: Added section about Pod Lifecycle, Checkpoint Storage, Alternatives and Hooks
* 2021-07-08: Reworked structure and added missing details

## Drawbacks

Not sure.

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
