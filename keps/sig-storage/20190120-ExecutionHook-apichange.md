---
title: ExecutionHook 
authors:
  - "@jingxu97"
  - "@xing-yang"
owning-sig: sig-storage
participating-sigs:
  - sig-storage
  - sig-node
  - sig-apps
  - sig-architecture
reviewers:
  - "@saad-ali"
  - "@thockin"
  - "@liyinan926"
approvers:
  - "@thockin"
  - "@saad-ali"
  - "@liyinan926"
editor: TBD
creation-date: 2019-1-20
last-updated: 2019-4-14
status: implementable
see-also:
  - n/a
replaces:
  - n/a
superseded-by:
  - n/a
---

# Title

ExecutionHook API changes

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
guarantees beyond any guarantees provided by storage systems (e.g. crash consistency).

This proposal is aimed to address that limitation by providing an API (ExecutionHook) and controller (ExecutionHook controller) for dynamically executing user’s command in pods/containers, e.g., application quiesce and unquiesce. ExecutionHook provides a general mechanism for users to trigger hook commands in their containers for their different use cases. Different options have been evaluated to decide how this ExecutionHook should be managed and executed. The preferred option is described in the Proposal section. The other options are discussed in the Alternatives section.

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

`ExecutionHook` is introduced to define actions that can be taken on a container. Specifically it can be used to perform a PreAction (freeze) operation before a volume snapshot is taken and then perform an PostAction (thaw) operation after a volume snapshot is taken to resume the application. PreAction and PostAction are used here instead of freeze and thaw so that ExecutionHook can be extended for other use cases in the future.

### Non-Goals

This proposal does not provide exact command included in the `ExecutionHook`
because every application has a different requirement.

ApplicationSnapshot and GroupSnapshot will be mentioned in this proposal whenever relevant, but detailed design will not be included in this spec.

## Proposal

In volume snapshotting, in order to provide application consistency, user has to freeze application before taking the snapshot and unfreeze it afterwards. To automate this process, we propose the ExecutionHook API and its controller. ExecutionHook API defines what commands should be executed on which pods/containers and the status of the execution. The volume snapshot controller will dynamically create and delete the hook before and after the snapshotting to freeze/unfreeze the application. An ExecutionHook controller will be responsible for watching the ExecutionHook objects and handling the execution of the commands defined in the hook.

Since the same type of applications might use the same commands for application quiesce and unquiesce, we also propose another CRD ExecutionHookTemplate so that ExecutionHooks can be easily created based on the template instead of defining it repeatedly in different places. The following section will explain the details of using ExecutionHook and ExecutionHookTemplate when taking a volume snapshot.

The ExecutionHook and ExecutionHookTemplate API objects are in the same namespace as the VolumeSnapshot object. Each hook has a pair of actions, PreAction and PostAction (optional). The Create event of ExecutionHook will trigger the hook controller to run PreAction command. The Delete event of ExecutionHook will trigger the hook controller to run PostAction command before deleting the hook object physically. PostAction is optional in case no clean up commands are needed after executing the hook PreAction.

Here is the definition of an ExecutionHook API object:

```
// ExecutionHook is in the same namespace as VolumeSnapshot
type ExecutionHook struct {
        metav1.TypeMeta
        // +optional
        metav1.ObjectMeta

        // Spec defines the behavior of a hook.
        // +optional
        Spec ExecutionHookSpec

        // Status defines the current state of a hook.
        // +optional
        Status ExecutionHookStatus
}
```

Here is the definition of the ExecutionHookSpec:

```
// ExecutionHookTemplateName is copied to ExecutionHookSpec by the controller such as
// the Snapshot Controller.
type ExecutionHookSpec struct {
        // PodContainerNamesList lists the pods/containers on which the ExecutionHook
        // should be executed. If not specified, the ExecutionHook controller will find
        // all pods and containers based on PodSelector and ContainerSelector.
        // If both PodContainerNamesList and PodSelector/ContainerSelector are not
        // specified, the ExecutionHook cannot be executed and it will fail.
        // +optional
        PodContainerNamesList []PodContainerNames

        // PodSelector is a selector which must be true for the hook to run on the pod
        // Hook controller will find all pods based on PodSelector, if specified.
        // +optional
        PodSelector *metav1.LabelSelector

        // ContainerSelector is a selector which must be true for the hook to run on the container
        // Hook controller will find all containers in the selected pods based on ContainerSelector,
        // if specified.
        // The ContainerSeletor map uses container names to select containers because the Container
        // struct does not have labels. For example: ContainerName: mysql
        // +optional
        ContainerSelector map[string]string

	// Name of the ExecutionHookTemplate. This is required.
	ExecutionHookTemplateName string
}

type PodContainerNames struct {
        PodName string
        ContainerNames []string
}

type ExecutionHookAction struct {
        // This is required
        Action core_v1.Handler

        // +optional
        ActionTimeoutSeconds *int64
}
```

Here is the definition of the ExecutionHookStatus. This represents the current state of a hook for all selected containers in the selected pods.
```
// ExecutionHookStatus represents the current state of a hook
type ExecutionHookStatus struct {
        // This is a list of ContainerExecutionHookStatus, with each status representing information
        // about how hook is executed in a container, including pod name, container name, preAction
        // timestamp, ActionSucceed, etc.
        // +optional
        ContainerExecutionHookStatuses []ContainerExecutionHookStatus

        // PreAction Summary status
        // Default is nil
        // +optional
        PreActionSucceed *bool

        // PostAction Summary status
        // Default is nil
        // +optional
        PostActionSucceed *bool
}

// ContainerExecutionHookStatus represents the current state of a hook for a specific container in a pod
type ContainerExecutionHookStatus struct {
        // This field is required
        PodName string

        // This field is required
        ContainerName string

        PreActionStatus *ExecutionHookActionStatus

        PostActionStatus *ExecutionHookActionStatus
 }

type ExecutionHookActionStatus struct {
        // If not set, it is nil, indicating Action has not started
        // If set, it means Action has started at the specified time
	// +optional
        ActionTimestamp *int64

        // +optional
        // If not set, it is nil, indicating Action is not complete on the specified container in the pod
        ActionSucceed *bool

        // The last error encountered when running the action
	// +optional
        Error *HookError
}

type HookError storage.VolumeError
```

In the ExecutionHookStatus object, there is a list of ContainerExecutionHookStatus for all selected containers in the pods, each ContainerExecutionHookStatus represents the state of the hook on a specific container.

The PreActionSucceed field is a summary status field to indicate the final status of the PreAction in the hook after the commands are run on all selected containers. If not set, this field is nil, meaning the hook controller still needs to wait for the PreAction commands to finish running on all containers. If set, either true or false, it means the PreAction commands have finished running on all containers.

The PostActionSucceed field is a summary status field to indicate the final status of the PostAction in the hook after the commands are run on all selected containers in the pods. If not set, this field is nil, meaning the hook controller still needs to wait for the PostAction commands to finish running on all containers. If set, either true or false, it means the PostAction commands have finished running on all containers in the pods.

There is a PreActionStatus and a PostActionStatus fields in ContainerExecutionHookStatus object, both are of ExecutionHookActionStatus type.

If ActionTimestamp is not set, it indicates PreAction or PostAction has not started for the specific container in the pod. If set, it means PreAction or PostAction has started at the specified time.

If ActionSucceed is not set, it indicates PreAction or PostAction has not completed on the specific container in the pod. If ActionSucceed is set, whether true or false, it means PreAction or PostAction has completed on the specific container in the pod successfully (true) or timed out (false).

For each execution hook in the specified container, the hook controller will retry until it times out. ActionTimeoutSeconds is used to limit a single hook execution in one container. If it is not successful when it times out, it will be marked as failed, meaning ActionSucceed in ExecutionHookActionStatus will be set to false. If it hangs, it is also marked as failed when it times out.

The summary status, PreActionSucceed or PostActionSucceed in ExecutionHookStatus, will be marked as failed (false) if any hook command on any container fails.

The summary status, PreActionSucceed or PostActionSucceed in ExecutionHookStatus, will only be marked as succeed (true) after all hook commands succeed on all specified containers in the pods.

Here is the definition for ExecutionHookTemplate:
```
// ExecutionHookTemplate describes pre and post action commands to run on pods/containers based
// on specified policies.
// ExecutionHookTemplate does not contain information on pods/containers because those are
// runtime info.
// ExecutionHookTemplate is namespaced
type ExecutionHookTemplate struct {
        metav1.TypeMeta
        // +optional
        metav1.ObjectMeta

        // Policy specifies whether to execute once on one selected pod or execute on all selected pods
        // +optional
        Policy *HookPolicy

        // This is required.
        PreAction *ExecutionHookAction

        // +optional
        PostAction *ExecutionHookAction

        // ExpirationSeconds is the time between PreAction and PostAction.
        // When a hook command is executed on multiple pods/containers, each
        // ContainerExecutionHookStatus has a PreActionStatus and a PostActionStatus.
        // Each Pre/PostActionStatus has a ActionTimeStamp to specify when the action
        // started to run. So the start time of ExpirationSeconds is determined by
        // the first ActionTimeStamp in a PreActionStatus and the end time of
        // ExpirationSeconds is determined by the last ActionTimeStamp of a PostActionStatus.
        // ExecutionHook controller will check the ExpirationSeconds timeout. Even If
        // API server dies, hook controller will still trigger PostAction command if
        // ExpirationSeconds is set and timeout happens.
        // +optional
        ExpirationSeconds *int64
}
```

Here is the definition for HookPolicy:
```
type HookPolicy string

// These are valid policies of ExecutionHook
const (
        // Execute commands only once on any one selected pod
        HookExecuteOnce HookPolicy = "ExecuteOnce"
        // Execute commands on all selected pods
        HookExecuteAll HookPolicy = "ExecuteAll"
)
```

The following explains the details of how to use ExecutionHook for volume snapshotting use case.

API Change: Add ExecutionHookInfos to VolumeSnapshotSpec:
```
// VolumeSnapshotSpec describes the common attributes of a volume snapshot
type VolumeSnapshotSpec struct {
        // Source has the information about where the snapshot is created from.
        // In Alpha version, only PersistentVolumeClaim is supported as the source.
        // If not specified, user can create VolumeSnapshotContent and bind it with VolumeSnapshot
        //  manually.
        // +optional
        Source *core_v1.TypedLocalObjectReference `json:"source" protobuf:"bytes,1,opt,name=source"`

        // SnapshotContentName binds the VolumeSnapshot object with the VolumeSnapshotContent
        // +optional
        SnapshotContentName string `json:"snapshotContentName" protobuf:"bytes,2,opt,name=snapshotContentName"`

        // Name of the VolumeSnapshotClass used by the VolumeSnapshot. If not specified, a default
        // snapshot class will be used if it is available.
        // +optional
        VolumeSnapshotClassName *string `json:"snapshotClassName" protobuf:"bytes,3,opt,name=snapshotClassName"`

        // ExecutionHookInfos is a new field in VolumeSnapshotSpec
        // List of ExecutionHookInfo. It is used to build ExecutionHooks
	// Hooks will be run sequencially now.
	// TODO: In the future, we may add ExecutionOrdering to indicate the ordering
	// of running the hooks.
        // +optional
        ExecutionHookInfos []ExecutionHookInfo
}
```

ExectionHookNames will be copied to VolumeSnapshotStatus when the volume snapshot
controller creates the hooks. If a hook execution times out on a container, execution
hook controller won't delete the hook. User can look up the hook status and clean up
later. If a hook runs successfully and snapshot is taken successfully, the hook will
be deleted but the hook name will still be in VolumeSnapshotStatus to indiate the
hook was run.

API Change: Add ExecutionHookNames to VolumeSnapshotStatus:
```
type VolumeSnapshotStatus struct {
        // CreationTime is the time the snapshot was successfully created. If it is set,
        // it means the snapshot was created; Otherwise the snapshot was not created.
        // +optional
        CreationTime *metav1.Time `json:"creationTime" protobuf:"bytes,1,opt,name=creationTime"`

        // When restoring volume from the snapshot, the volume size should be equal to or
        // larger than the RestoreSize if it is specified. If RestoreSize is set to nil, it means
        // that the storage plugin does not have this information available.
        // +optional
        RestoreSize *resource.Quantity `json:"restoreSize" protobuf:"bytes,2,opt,name=restoreSize"`

        // ReadyToUse is set to true only if the snapshot is ready to use (e.g., finish uploading if
        // there is an uploading phase) and also VolumeSnapshot and its VolumeSnapshotContent
        // bind correctly with each other. If any of the above condition is not true, ReadyToUse is
        // set to false
        // +optional
        ReadyToUse bool `json:"readyToUse" protobuf:"varint,3,opt,name=readyToUse"`

        // The last error encountered during create snapshot operation, if any.
        // This field must only be set by the entity completing the create snapshot
        // operation, i.e. the external-snapshotter.
        // +optional
        Error *storage.VolumeError `json:"error,omitempty" protobuf:"bytes,4,opt,name=error,casttype=VolumeError"`

	// ExecutionHookNames is a new field in VolumeSnapshotStatus
	// ExectionHookNames will be copied to VolumeSnapshotStatus when the volume snapshot
	// controller creates the hooks.
	ExecutionHookNames []string
}
```

```
type ExecutionHookInfo struct {
        // PodSelector is a selector which must be true for the hook to run on the pod
        // Hook controller will find all pods based on PodSelector
        // +optional
        PodSelector *metav1.LabelSelector

        // ContainerSelector is a selector which must be true for the hook to run on the container
        // Hook controller will find all containers in the selected pods based on ContainerSelector.
        // This selector is based on container names because Container struct does not have labels.
        // +optional
        ContainerSelector map[string]string

        // PodContainerNamesList specified containers and pods that the hook should be executed on.
        // If both PodContainerNamesList and PodSelector/ContainerSelector are specified,
        // only PodContainerNamesList will be used.
        // Without PodSelector/ContainerSelector and PodContainerNamesList, Snapshot controller will
        // automatically detect pods/containers needed to run the hook from the volume source of the
        // snapshot and set them in the ExecutionHookSpec.
        // If PodContainerNamesList is not specified and PodSelector/ContainerSelector are specified,
        // snapshot controller will copy them to ExecutionHookSpec. ExecutionHook controller will find
        // all pods and containers based on PodSelector and ContainerSelector.
        // +optional
        PodContainerNamesList []PodContainerNames

        // Name of the ExecutionHookTemplate
        // This is required and must have a valid hook template name; an empty string is not valid
        ExecutionHookTemplateName string
}
```

If neither PodSelector/ContainerSelector nor PodContainerNamesList is given, snapshot controller will find the pod and container names automatically.

Since this proposal is focusing on introducing `ExecutionHook` for volume snapshotting, we use PreAction to indicate Freeze and PostAction to indicate Thaw commands. Freeze and Thaw are needed for taking an application consistent volume snapshot.

`ExecutionHook` can be used for other purposes in addition to volume snapshotting. `ExecutionHook` can be used for ApplicationSnapshot as well.
In addition, `ExecutionHook` may be used for the following cases:

* Upgrade
* Rolling upgrade
* Debugging
* Prepare for some lifecycle event like a database migration
* Reload a config file
* Restart a container

PreAction and PostAction are general names that can be used for non-snapshotting purposes. However, additional design discussions need to happen on how they can be used for other specific use cases in the future as this current proposal will be focusing on the use case of volume snapshotting only.

For volume snapshotting, PreAction and PostAction hooks shall be included in ExecutionHooks. The snapshot controller will be creating and deleting those hooks based on the ExecutionHookTemplateNames specified in a VolumeSnapshot API object. There will be an ExecutionHook controller that will be watching the hooks and executing them.

### Workflow

Workflow for the snapshot controller is as follows:

* Snapshot controller watches VolumeSnapshot objects. If there is a request to create a VolumeSnapshot, it checks if there is ExecutionHookInfos defined in VolumeSnapshot.
* If there is ExecutionHookInfos in the VolumeSnapshot object, the snapshot controller checks whether it is necessary to run the ExecutionHook.
  * Snapshot controller watches VolumeAttachment object and searches for a PersistentVolumeClaimName that matches the PersistentVolumeClaim of the snapshot source.
  * Snapshot controller checks if Attached in VolumeAttachmentStatus is true. If it is attached, it means a PreAction operation is needed so it is necessary to run the ExecutionHook. If not, it will go ahead to cut the snapshot without creating an ExecutionHook.
  * Snapshot controller needs to create multiple ExecutionHooks if necessary, one for each command running in pods/containers, if different commands are needed for different pods/containers.
  * Snapshot controller needs to wait for all PreAction hooks to complete running before taking snapshot.
* If ExecutionHook should be run, snapshot controller will create an ExecutionHook object.
  * ExecutionHookInfo has optional pod/container selectors and PodContainerNamesList. If PodContainerNamesList is specified, it will be copied to ExecutionHook. If PodContainerNamesList is not specified but pod/container selectors are specified, snapshot controller will find pods/containers based on the selector. It is possible that it will find multiple pods. If both PodSelector/ContainerSelector and PodContainerNamesList are not defined in ExecutionHookInfos, snapshot controller will automatically figure out what pods/containers are involved. If ExecuteOnce policy is set, we need to run commands on one pod only. If ExecutionAll policy is set, we need to run commands on all pods. Only if the hook runs successfully on all selected containers in the pods, it is successful.
  * When ExecutionHook is used in other use cases, e.g., application snapshot, it is possible that the selectors in ExecutionHook may override the ones specified in ExecutionHookInfo.
* The ExecutionHook controller watches the ExecutionHook API object. When it sees a hook whose PreActionSucceed is not set (nil), it loops around and checks ContainerExecutionHookStatuses for all specified containers in the pods.
  * If ActionTimestamp in PreActionStatus is nil, it sets `ActionTimeStamp` in the ContainerExecutionHookStatus. If ActionTimeStamp exists, it means PreAction has already started on this container. The hook controller executes the command in the container based on ExecutionHook definitions. If the `PreAction` command times out and it is still not successful, the ExecutionHook controller sets ActionSucceed to false in ContainerExecutionHookStatuses. In the case of timeout or failure, log event with failure reason. Default value of ActionSucceed pointer is nil. The PreActionSucceed summary field in ExecutionHookStatus will be set to false when ActionSucceed in any ContainerExecutionHookStatus becomes false.
  * If the `PreAction` command completes successfully, the ExecutionHook controller updates the ActionSucceed field to true in the ContainerExecutionHookStatus.
* When the `PreAction` command completes successfully in all specified containers, the ExecutionHook controller updates the PreActionSucceed summary field to true in the status of the hook. Snapshot controller checks the status and knows whether to cut snapshot or continue to wait or bail out. Snapshot controller needs to wait until hook is run on all pods (if more than 1 pod) and the PreActionSucceed summary status is set to true. If PreActionSucceed is false, the snapshot controller won’t cut snapshot and will fail the operation without retries.
  * The ExecutionHook controller runs the hook command remotely using the Pod SubResource exec.
* If the PreActionSucceed summary status is set to true, the snapshot controller will start to cut the snapshot.
  * Note that if there are more than 1 ExecutionHook, the snapshot controller will have to wait for PreActionSucceed summary status to be true in all hooks before starting to cut the snapshot.
  * If there are multiple ExecutionHooks, the snapshot controller needs to coordinate and decide which hook to run first if the order is important.
* If the snapshot creation (cut) fails, the snapshot controller will fail the operation and won’t retry.
* If the snapshot creation (cut) hangs, it will eventually get a gRPC timeout and the snapshot controller will fail the operation and won’t retry.
* If the snapshot is cut successfully (CreateSnapshot returns from CSI plugin successfully), the snapshot controller will delete the ExecutionHook. This will set a delete timestamp in the hook but not really deleting it yet. It tells the hook controller that the action is complete. Note that at this point, VolumeSnapshot and VolumeSnapshotContent may not be bound and ReadyToUse may not be true yet. We want to let the ExecutionHook controller know it is safe to run the PostAction and resume the application as soon as possible.
* When the ExecutionHook controller receives the delete event, it checks whether PostAction is defined in the hook. If it is not defined, PostAction is not needed. PostActionSucceed summary status in the hook will be set to true.
* If PostAction is defined in the hook, the ExecutionHook controller goes through ContainerExecutionHookStatuses for all selected containers.
  * For each container, it will set ActionTimestamp and start to run the PostAction command. If PostAction fails, the ExecutionHook controller will retry until it times out. If the action is not successful when it times out, the ExecutionHook controller will set ActionSucceed to false in ContainerExecutionHookStatus and log an event. This would require admin intervention to fix the problem so the application won’t be frozen forever.
  * If the post action fails for any container, PostActionSucceed summary field will be set to false as well in the hook status.
  * When the PostAction command completes successfully in a specified container in the pod, the ExecutionHook controller will set ActionSucceed status to true in the ContainerExecutionHookStatus.
* When PostAction has been run successfully on all selected containers in the pods, a PostActionSucceed summary status will be set to true. If PostAction fails on any container, PostActionSucceed will be set to false but the hook controller will continue to run the command on the remaining containers. When PostActionSucceed summary status is set to true, the hook controller will finally delete the ExecutionHook.
* There could be multiple pods and containers selected for one hook. The hook controller needs to wait for hook commands to be done on all selected containers. Only if command runs successfully on all containers on all pods, the overall status is successful.
* If there are more than 1 ExecutionHooks, the snapshot controller will have to delete all hooks after cutting the snapshot.
* PreAction and PostAction hooks are not required. While most applications require both PreAction and PostAction hooks, some applications such as Mysql only require a PreAction hook. When a PreAction command is run in a mysql session to freeze the application, the PostAction command must be run from the same session to unfreeze it. If the session is closed after the PreAction command, the mysql application will automatically be unfrozen. Therefore a PreAction hook for mysql needs to be running in the background after running the freeze command, and PostAction command will be run within the same session after the snapshot is cut (or closing the session to unfreeze it). For most other applications, PreAction and PostAction do not have to happen in the same session so we will have two separate handlers.
* The ExecutionHook controller will guarantee that a PreAction or PostAction command is issued, however, it cannot guarantee the command will succeed because that is determined by the application.
* It is recommended that the user adds `PostAction` command inside the `PreAction` script and makes sure that `PostAction` is automatically run after the `PreAction` command has completed successfully for a period of time, same as the ExpirationTimeout.
* Both PreAction and PostAction should be idempotent.

Here is an example of an ExecutionHookTemplate:
```
apiVersion: storage.k8s.io/v1alpha1
kind: ExecutionHookTemplate
metadata:
  name: hook-template-demo
policy: ExecuteOnce
preAction:
  exec:
    command: [“run_quiesce.sh”]
  actionTimeoutSeconds: 10
postAction:
  exec:
    command: [“run_unquiesce.sh”]
  actionTimeoutSeconds: 10
expirationTimeSeconds: 30
```

Here is an example of how the ExecutionHookInfos is used in a VolumeSnapshot object:
```
apiVersion: snapshot.storage.k8s.io/v1alpha1
kind: VolumeSnapshot
metadata:
  name: snapshot-demo
spec:
  snapshotClassName: csi-snapshot-class
  source:
    name: pvc-demo
    kind: PersistentVolumeClaim
  executionHookInfos:
    - executionHookTemplateName: hook-template-demo
      podContainerNamesList:
        - podName: mysql
          containerNames: ["mysql"]
```

Here is an ExecutionHook created by the snapshot controller:
```
apiVersion: storage.k8s.io/v1alpha1
kind: ExecutionHook
metadata:
  name: hook-demo
spec:
  podContainerNamesList:
    -podName: mysqlPod
      -containerName: mysql1
  preAction:
    exec:
      command: [“run_quiesce.sh”]
    actionTimeoutSeconds: 10
  postAction:
    exec:
      command: [“run_unquiesce.sh”]
    actionTimeoutSeconds: 10
  expirationTimeSeconds: 30
```

Here is another example with multiple pods:
```
apiVersion: storage.k8s.io/v1alpha1
kind: ExecutionHook
metadata:
  name: hook-demo
spec:
  podContainerNamesList:
    -podName: myPod1
      -containerName: myContainer1
      -containerName: myContainer2
    -podName: myPod2
      -containerName: myContainer3
      -containerName: myContainer4
  preAction:
    exec:
      command: [“run_quiesce.sh”]
    actionTimeoutSeconds: 10
  postAction:
    exec:
      command: [“run_unquiesce.sh”]
    actionTimeoutSeconds: 10
  expirationTimeSeconds: 30
```

The following RBAC rules will be added for the ExecutionHook controller to run the hook through the pod subresource exec:
```
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["pods/exec"]
    verbs: ["create"]
```

The snapshot controller only handles taking individual volume snapshots. For taking consistent group snapshot of multiple volumes that belong to the same application, a higher level operator for GroupSnapshot or ApplicationSnapshot needs to coordinate with the ExecutionHook controller to trigger the PreAction execution hooks before taking snapshots and trigger the PostAction execution hooks after taking snapshots. The ExecutionHookTemplateNames should be added to the GroupSnapshot or ApplicationSnapshot object instead of the VolumeSnapshot object in this case. Discussions for GroupSnapshot or ApplicationSnapshot will be covered in future design proposals.

### User Stories

## Workarounds

## Alternatives

The user can use Annotations to define the execution hook if it is not in the
container struct.

We also considered several options based on feedback from design meetings and spec reviews.

### Alternative Option 1a

Define ExecutionHook as a struct, not an API object, and add it to Lifecycle in the Container struct.

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
        TimeoutSeconds int64 `json:"timeoutSeconds,omitempty" protobuf:"varint,4,opt,name=timeoutSeconds"`
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

Add `ExecutionHook` to `Lifecycle`.

```
type Lifecycle struct {
        // PostStart is called immediately after a container is created.  If the handler fails, the container
        // is terminated and restarted.
        // +optional
        PostStart *Handler
        // Defines the hooks to trigger an action in a container
        // +optional
        ExecutionHooks []ExecutionHook
        // PreStop is called immediately before a container is terminated.  The reason for termination is
        // passed to the handler.  Regardless of the outcome of the handler, the container is eventually
        // terminated.
        // +optional
        PreStop *Handler
}
```

### Alternative Option 1b

Add `Name`, `Type`, and `TimeOutSeconds` from the `ExecutionsHook` struct directly to the Handler struct. In that case we have to make `Type` optional for backward compatibility.

```
type Handler struct {
        // One and only one of the following should be specified.
        // Exec specifies the action to take.
        // +optional
        Exec *ExecAction
        // HTTPGet specifies the http request to perform.
        // +optional
        HTTPGet *HTTPGetAction
        // TCPSocket specifies an action involving a TCP port.
        // TODO: implement a realistic TCP lifecycle hook
        // +optional
        TCPSocket *TCPSocketAction
        // Name of an execution hook
        // +optional
        Name string
        // Type of an execution hook
        // +optional
        Type string
        // How long the controller should wait for the hook to complete execution
        // before giving up. The controller needs to succeed or fail within this
        // timeout period, regardless of the retries. If not set, the controller
        // will set a default timeout.
        // +optional
        TimeoutSeconds int64
}

type Lifecycle struct {
        // PostStart is called immediately after a container is created.  If the handler fails, the container
        // is terminated and restarted.
        // +optional
        PostStart *Handler
        // Defines the hooks to trigger an action in a container
        // +optional
        ExecutionHooks []Handler
        // PreStop is called immediately before a container is terminated.  The reason for termination is
        // passed to the handler.  Regardless of the outcome of the handler, the container is eventually
        // terminated.
        // +optional
        PreStop *Handler
}
```

### Controller Handlings for Option 1a and 1b

There are two options about which component takes care of these hooks.
* Kubelet
  Kubelet is responsible for handling the lifecycle hooks during starting/stopping containers. So kubelet could be the central place to handle other types of generic execution hooks. However, there are a number of challenges to support it now.
  * It is not clear when and how kubelet needs to trigger these actions, how to report the status of executing these hooks, whether it fails or succeeds.
  * If kubelet supports the execution hook, it should be designed to support general use cases instead of only specific use cases. However, at this stage, we don’t have enough information to determine how it should be defined generally to support a variety of use cases.
  * It probably takes more time to enable kubelet to support this considering the complexity and all edge cases.

* External Controller
  Even though the API is defined inside of the container spec, it is possible to allow external controller to use Pod.exec subresource to run arbitrary commands when needed. For example, snapshot controller can trigger freeze hook before taking the snapshot. It is easier for controller to understand the failure and handle the workflow of snapshotting. However, the big concern about this approach is to allow external controllers to execute untrusted code. It would be easy to lose control of how the hooks can be used without careful security considerations.

Here is an example of how the ExecutionHook is used in a container:

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

The following RBAC rules will be added for the snapshot controller if the snapshot controller is responsible for executing the hook:

```
- apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["pods/exec"]
    verbs: ["create"]
```

Pros for option 1 and 2: All the handlers are together in the `Lifecycle` struct. It looks clean.

Cons for option 1: `ExecutionHook` has its own types and looks different from the other handlers.

Cons for option 1 and 2:

* `Type` is optional here for backward compatibility, however, `Type` should be required because a `Freeze` type tells the controller it needs to be run before cutting the snapshot while a `Thaw` type tells the controller to run it after cutting the snapshot. Making it optional will make it hard for the snapshot controller to determine when to run those hooks.
* User may get confused. User may expect everything inside `Lifecycle` is handled by kubelet, however, `ExecutionHooks` will be handled by an external controller/operator.

Other differences between `ExecutionHook` and existing `Lifecycle` struct:

* The `ExecutionHook` is triggered by an external controller/operator using pod subresource exec.  Hook can be triggered at any point in the lifecycle of the container based on the `Type`.
* `Lifecycle` is triggered and handled by kubelet in the beginning and end of the container lifecycle.

### Alternative Option 2

A second option is to add this hook information into the VolumeSnapshot spec. The hook struct needs to specify where these hooks should be applied to, normally through a pod selector. Or, we can use a new CRD to store this information. The controller which requests taking snapshots will trigger these hook to specified Pods/containers. The concern is similar to the above mentioned to allow different external controllers to execute these arbitrary user defined hooks.

```
apiVersion: storage.org/v1alpha1
kind: PrePostSnapshotHook
metadata:
  name: prepostsnap-rule
spec:
  - podSelector:
      app: mysql
    actions:
    - type: command
      # this command will flush tables with read lock
      value: mysql --user=root --password=$MYSQL_ROOT_PASSWORD -Bse 'flush tables with read lock'
```

### Risks and Mitigations

## Graduation Criteria

When the existing volume snapshot alpha feature goes beta, the `ExecutionHook`
feature will become beta as well.

## Implementation History

* Feature description: https://github.com/kubernetes/enhancements/issues/962
