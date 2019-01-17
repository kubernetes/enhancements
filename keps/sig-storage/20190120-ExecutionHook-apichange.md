---
title: ExecutionHook 
authors:
  - "@jingxu97"
  - "@xing-yang"
owning-sig: sig-storage
participating-sigs:
  - sig-storage
  - sig-node
  - sig-architecture
reviewers:
  - "@saad-ali"
  - "@thockin"
approvers:
  - "@thockin"
  - "@saad-ali"
editor: TBD
creation-date: 2019-1-20
last-updated: 2019-1-20
status: implementable
see-also:
  - n/a
replaces:
  - n/a
superseded-by:
  - n/a
---

# Title

ExecutionHook API change

## Table of Contents

  * [Title](#title)
      * [Table of Contents](#table-of-contents)
      * [Summary](#summary)
      * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals)
      * [Proposal](#proposal)
         * [User Stories](#user-stories)
      * [Workarounds](#workarounds)
      * [Alternatives](#alternatives)
         * [Risks and Mitigations](#risks-and-mitigations)
      * [Graduation Criteria](#graduation-criteria)
      * [Implementation History](#implementation-history)

## Summary

Volume snapshot support was introduced in Kubernetes v1.12 as an alpha feature. 
In the alpha implementation of snapshots for Kubernetes, there is no snapshot consistency
guarantees beyond any guarantees provided by storage system (e.g. crash consistency).

This proposal is aimed to address that limitation by providing an `ExecutionHook`
in the `Container` struct. The snapshot controller will look up this hook before
taking a snapshot and execute it accordingly.

## Motivation

The volume snapshot feature allows creating/deleting volume snapshots, and the
ability to create new volumes from a snapshot natively using the Kubernetes API.

However, application consistency is not guaranteed. An user has to figure out how
to quiece an application before taking a snapshot and unquiece it after taking
the snapshot.

So we want to introduce an `ExecutionHook` to facilitate the quiesce and unquiesce
actions when taking a snapshot. There is an existing lifecycle hook in the `Container`
struct. The lifecycle hook is called immediately after a container is created or
immediately before a container is terminated. The proposed execution hook is not
tied to the start or termination time of the container. Instead it is called before
or after taking a snapshot while the container is still running.

### Goals

`ExecutionHook` is introduced to define actions that can be taken on a container.
Specifically it can be used to perform a quiese (freeze) operation before a volume
snapshot is taken and then perform an unquiesce (thaw) operation after a volume
snapshot is taken to resume the application.

## Non-Goals

This proposal does not provide exact command included in the `ExecutionHook`
because every application has a different requirement.

## Proposal

An `ExecutionHook` is defined in the following.

```
// ExecutionHook defines a specific action that should be taken with timeout
type ExecutionHook struct {
        // Name of an ExecutionHook
        Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
        // Type of an ExecutionHook
        Type string `json:"type" protobuf:"bytes,2,opt,name=type"`
        // Command to execute for a particular trigger
        Handler `json:",inline" protobuf:"bytes,3,opt,name=handler"`
        // How long the controller should wait for the hook to complete execution
        // before giving up. The controller needs to succeed or fail within this
        // timeout period, regardless of the retries. If not set, the controller
        // will set a default timeout.
        // +optional
        TimeOutSeconds int64 `json:"timeOutSeconds,omitempty" protobuf:"varint,4,opt,name=timeOutSeconds"`
}
```

An `ExecutionHook` includes a name and a type. The type has to be one of valid types
specified for `ExecutionHookType` as follows.

```
type ExecutionHookType string

// These are valid types of ExecutionHooks
const (
        // An ExecutionHook that freezes an application
        ExecutionHookFreeze ExecutionHookType = "Freeze"
        // An ExecutionHook that thaws an application
        ExecutionHookThaw ExecutionHookType = "Thaw"
)
```

Since this proposal is focusing on introducing `ExecutionHook` for volume snapshotting,
the valid `ExecutionHookTypes` only include `Freeze` and `Thaw` types specifically
used for taking an application consistent volume snapshot.

`ExecutionHook` can be used for other purposes in addition to volume snapshotting.
For example, `ExecutionHook` may be used for the following cases:

* Upgrade
* Rolling upgrade
* Debugging
* Prepare for some lifecycle event like a database migration
* Reload a config file
* Restart a container

In order to use `ExecutionHook` for non-snapshotting purposes, new `ExecutionHookType`
needs to be added. This is to make sure that an controller knows what type of
`ExecutionHook` to watch and avoid the user from defining a type that is already
watched by an controller unintentionally.

In this proposal, `ExecutionHook` is specifically used for defining `Freeze` and `Thaw`
hooks when taking a snapshot. Definitions for other use cases can be added later when
needed.

`ExecutionHooks` is added to the `Container` struct.

```
// A single application container that you want to run within a pod.
type Container struct {
        ......
        // Defines the hooks to trigger an action in a container
        // +optional
        ExecutionHooks []ExecutionHook `json:"executionHooks,omitempty" protobuf:"bytes,22,opt,name=executionHooks"`
}
```

For volume snapshotting, `Freeze` and `Thaw` hooks shall be included in `ExecutionHooks`.
The snapshot controller will be watching those hooks.

Workflow for the snapshot controller is as follows:

* Snapshot controller watches `VolumeAttachment` object and searches for a `PersistentVolumeClaimName` that matches the `PersistentVolumeClaim` of the snapshot source.
* Snapshot controller verifies `Attacher` is the same as `Snapshotter`.
* Snapshot controller checks if `Attached` in `VolumeAttachmentStatus` is `true`. If it is attached, it means a freeze operation is needed.
* Snapshot controller also watches `Pod`, and finds out which pod is using this volume from the `PodSpec`.
* If containers in the pod contains a `Freeze` execution hook, the snapshot controller runs it before snapshotting. It runs it remotely using the `Pod` `SubResource` `exec`.
* If containers in the pod contains a `Thaw` execution hook, the snapshot controller runs it after snapshotting. It runs it remotely using the `Pod` `SubResource` `exec`.
* `Freeze` and `Thaw` hooks are not required. However, they must be specified together. If only one is specified, the snapshot controller will fail the snapshotting operation.
* If `Freeze` is issued by the snapshot controller, a `Thaw` must be issued regardless whether the snapshotting is successful or not.
* If `Freeze` fails, the snapshot controller will fail the snapshotting operation.
* If `Thaw` fails, the snapshot controller will log an event so the admin can intervene and fix the problem with the application.
* The snapshot controller will guarantee that a `Freeze` or `Thaw` command is issued, however, it cannot guarantee the command will succeed because that is determined by the application.
* It is recommended that the user adds "thaw" command inside the "Freeze" script and makes sure that "thaw" is automatically run after the "freeze" command has completed successfully for a period of time.
* Both `Freeze` and `Thaw` should be idempotent.

Here is an example of how the `ExecutionHook` is used in a container:

```
apiVersion: v1
kind: Pod
metadata:
  name: hook-demo
spec:
  containers:
  - name: demo-container
    image: mysql
    executionHooks:
      - name: freeze
        type: Freeze
        exec:
          command: [“run_quiesce.sh”]
        timeoutSeconds: 30
      - name: thaw
        type: Thaw
        exec:
          command: [“run_unquiesce.sh”]
        timeoutSeconds: 30
```

The following RBAC rules will be added for the snapshot controller:

```
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["pods/exec"]
    verbs: ["create"]
```

The snapshot controller only handles taking individual volume snapshots. For taking consistent
group snapshot of multiple volumes that belong to the same application, a higher level operator
needs to trigger all the `Freeze` execution hooks before taking snapshots and trigger all the
`Thaw` execution hooks after taking snapshots. The snapshot controller will only be responsible
for taking the snapshot without handling `Freeze` and `Thaw` in this case. An option will be
added to the `VolumeSnapshotSpec` to indicate whether the execution hook should be handled
by the snapshot controller and the default is `false`.

### User Stories

## Workarounds

## Alternatives

The user can use Annotations to define the execution hook if it is not in the
container struct.

### Risks and Mitigations

## Graduation Criteria

When the existing volume snapshot alpha feature goes beta, the `ExecutionHook`
feature will become beta as well. 

## Implementation History

* Feature description: https://github.com/kubernetes/enhancements/issues/177
* VolumeSnapshotDataSource feature gate: https://github.com/kubernetes/kubernetes/pull/67087
