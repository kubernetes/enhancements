# ExecutionHook API Design

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Workflow](#workflow)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
  - [User Stories](#user-stories)
- [Workarounds](#workarounds)
- [Alternatives](#alternatives)
  - [Alternative Option 1a](#alternative-option-1a)
  - [Alternative Option 1b](#alternative-option-1b)
  - [Controller Handlings for Option 1a and 1b](#controller-handlings-for-option-1a-and-1b)
  - [Alternative Option 2](#alternative-option-2)
  - [Risks and Mitigations](#risks-and-mitigations-1)
- [Graduation Criteria](#graduation-criteria-1)
- [Implementation History](#implementation-history-1)
<!-- /toc -->

## Summary

This proposal is to introduce an API (ExecutionHook) for dynamically executing user’s commands in a pod/container or a group of pods/containers and a controller (ExecutionHookController) to manage the hook lifecycle. ExecutionHook provides a general mechanism for users to trigger hook commands in their containers for their different use cases. Different options have been evaluated to decide how this ExecutionHook should be managed and executed. The preferred option is described in the Proposal section. The other options are discussed in the Alternatives section.

## Motivation

The volume snapshot feature allows creating/deleting volume snapshots, and the ability to create new volumes from a snapshot natively using the Kubernetes API. However, application consistency is not guaranteed. An user has to figure out how
to quiece an application before taking a snapshot and unquiece it after taking the snapshot.

So we want to introduce an `ExecutionHook` to facilitate the quiesce and unquiesce actions when taking a snapshot. There is an existing lifecycle hook in the `Container` struct. The lifecycle hook is called immediately after a container is created or immediately before a container is terminated. The proposed execution hook is not tied to the start or termination time of the container. It can be triggered on demand by callers (users or controllers) and the status will be updated dynamically.

### Goals

`ExecutionHook` is introduced to define actions that can be taken on a container or a group of containers. The ExecutionHook controller is responsible for triggering the commands in containers and updating the status on whether the execution is succeeded or not. The controller will also garbage collect the hook objects based on some predefined policy.

### Non-Goals

This proposal does not provide exact command included in the `ExecutionHook`
because every application has a different requirement.

ApplicationSnapshot and GroupSnapshot will be mentioned in this proposal whenever relevant, but detailed design will not be included in this spec.

## Proposal

The general ExecutionHook API has two parts, spec and status. The hook spec has two piece of information, what are the commands to execute and where to execute them. In many use cases, different hooks share the same commands and user or controller will create many hooks for execution repeatedly. To reduce the work of copying the same execution commands in different hooks, we also propose a second API, HookAction to record the execution commands which can be referenced by ExecutionHook API. Both APIs are namespaced and they should be in the same namespace.

The Create event of ExecutionHook will trigger the hook controller to run HookAction command. The status will be updated once it is available by controller. It is the caller's (user or controller) responsibility to delete the hook once the execution finishes (either succeeded or failed). Otherwise, ExecutionHook controller will garbage collect them after a certain amount of time.

Here is the definition of an ExecutionHook API object:

```
// ExecutionHook is in the tenant namespace
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
// HookActionName is copied to ExecutionHookSpec by the controller such as
// the Snapshot Controller.
type ExecutionHookSpec struct {
        // PodSelection defines how to select pods and containers to run
	// the executionhook. If multiple pod/containers are selected, the action will executed on them
	// asynchronously. If execution ordering is required, caller has to implement the logic and create
	// different hooks in order.
	// This field is required.
        PodSelection PodSelection

        // Name of the HookAction. This is required.
        ActionName string
}

// PodSelection contains two fields, PodContainerNamesList and PodContainerSelector,
// where one of them must be defined so that the hook controller knows where to
// run the hook.
type PodSelection struct {
        // PodContainerNamesList lists the pods/containers on which the ExecutionHook
        // should be executed. If not specified, the ExecutionHook controller will find
        // all pods and containers based on PodContainerSelector.
        // If both PodContainerNamesList and PodContainerSelector are not
        // specified, the ExecutionHook cannot be executed and it will fail.
        // +optional
        PodContainerNamesList []PodContainerNames

        // PodContainerSelector is for hook controller to find pods and containers
        // based on the pod label selector and container names
        // If PodContainerNamesList is specified, this field will not be used.
        // +optional
        PodContainerSelector *PodContainerSelector
}

type PodContainerNames struct {
        PodName string
        ContainerNames []string
}

type PodContainerSelector struct {
	// PodSelector specifies a label query over a set of pods.
        // +optional
	PodSelector *metav1.LabelSelector

        // If specified, controller only select the containers that are listed from the selected pods based on PodSelector. 
	// Otherwise, all containers of the pods will be selected
        // +optional
        ContainerList []string
 }
```

Here is the definition of the ExecutionHookStatus. This represents the current state of a hook for all selected containers in the selected pods.

```
// ExecutionHookStatus represents the current state of a hook
type ExecutionHookStatus struct {
        // This is a list of ContainerExecutionHookStatus, with each status representing information
        // about how hook is executed in a container, including pod name, container name, ActionTimestamp,
        // ActionSucceed, etc.
        // +optional
        HookStatuses []ContainerExecutionHookStatus
}

// ContainerExecutionHookStatus represents the current state of a hook for a specific container in a pod
type ContainerExecutionHookStatus struct {
        // This field is required
        PodName string

        // This field is required
        ContainerName string

        // If not set, it is nil, indicating Action has not started
        // If set, it means Action has started at the specified time
	// +optional
        Timestamp *int64

        // ActionSucceed is set to true when the action is executed in the container successfully.
	// It will be set to false if the action cannot be executed successfully after ActionTimeoutSeconds passes.
        // +optional
        Succeed *bool

        // The last error encountered when executing the action. The hook controller might update this field each time
	// it retries the execution.
	// +optional
        Error *HookError
}

type HookError struct {
        // Type of the error
        // This is required
        ErrorType ErrorType

        // Error message
        // +optional
        Message *string

        // More detailed reason why error happens
        // +optional
        Reason *string
	
        // It indicates when the error occurred
	// +optional
        Timestamp *int64
}

type ErrorType string

// More error types could be added, e.g., Forbidden, Unauthorized, AlreadyInProgress, etc.
const (
        // The execution hook times out
        Timeout ErrorType = "Timeout"

        // The execution hook fails with an error
        Error ErrorType = "Error"
)
```

In the ExecutionHookStatus object, there is a list of ContainerExecutionHookStatus for all selected containers in the pods, each ContainerExecutionHookStatus represents the state of the hook on a specific container.

Here is the definition of HookAction:

```
// HookAction describes action commands to run on pods/containers based
// on specified policies. HookAction will be created by the user and
// can be re-used later. Snapshot Controller will create ExecutionHooks
// based on HookActions specified in the snapshot spec. For example,
// two HookActions, preSnapshotExecutionHook and postSnapshotExecutionHook,
// are expected in the snapshot spec.
// HookAction does not contain information on pods/containers because those are
// runtime info.
// HookAction is namespaced
type HookAction struct {
        metav1.TypeMeta
        // +optional
        metav1.ObjectMeta

        // This contains the command to run on a container.
	// The command should be idempotent because the system does not guarantee exactly-once semantics.
	// Any action may be triggered more than once but only the latest results will be logged in status.
	// As alpha feature, only ExecAction type in Handler will be support, not the HTTPGETAction or TCPSocketAction.
        // This is required.
        Action core_v1.Handler

        // ActionTimeoutSeconds defines when the execution hook controller should stop retrying.
        // If execution fails, the execution hook controller will keep retrying until reaching
        // ActionTimeoutSeconds. If execution still fails or hangs, execution hook controller
        // stops retrying and updates executionhook status to failed.
        // If controller loses its state, counter restarts. In this case, controller will retry
        // for at least this long, before stopping.
        // Once an action is started, controller has no way to stop it even if
        // ActionTimeoutSeconds is exceeded. This simply controls if retry happens or not.
        // retry is based on exponential backoff policy. If ActionTimeoutSeconds is not
        // specified, it will retry until the hook object is deleted.
        // +optional
        ActionTimeoutSeconds *int64
}
```

`ExecutionHook` may be used for different cases, such as

* Application-consistency snapshotting
* Upgrade
* Rolling upgrade
* Debugging
* Prepare for some lifecycle event like a database migration
* Reload a config file
* Restart a container


The following gives an example of how to use ExecutionHook for application-consistency snapshotting use case.

### Workflow

* Snapshot API should carry two execution hook information (which will reference two HookAction - preSnapshotExecutionHook and postSnapshotExecutionHook) for quiescing and unquiescing application.
* Snapshot controller watches Snapshot objects. If there is a request to create a Snapshot, it checks if there is ExecutionHook Information defined in Snapshot.
* If hook is defined in the snapshot, snapshot controller will create one or multiple ExecutionHooks for quiescing application if necessary, one for each command running in pods/containers.
* Snapshot controller waits for all Action hooks to complete running before taking snapshot.
* The ExecutionHook controller watches the ExecutionHook API object and take actions based on the object status and also update the status. 
* Snapshot controller waits until hook is run on all pods (if more than 1 pod). No matter the Action succeeds or fails, snapshot controller should create execution hooks for unquiescing application. Snapshot controller can also decide when to delete those hooks.

Here is an example of an HookAction:
```
apiVersion: apps.k8s.io/v1alpha1
kind: HookAction
metadata:
  name: action-demo
Action:
  exec:
    command: [“run_quiesce.sh”]
  actionTimeoutSeconds: 10
```

Here is an ExecutionHook created by the snapshot controller:
```
apiVersion: apps.k8s.io/v1alpha1
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
  hookActionName: action-demo
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

### Risks and Mitigations
The security concern is that ExecutionHook controller has the authority to execute commands in any pods. For alpha and proof of concept, we propose to use external controller to handle executionhooks. But to move to beta and graduate as GA, we will evaluate it and move it to kubelet which already has the privilege to execute commands in pod/containers.

## Graduation Criteria
Please see above Risks and Mitigations

## Implementation History

* Feature description: https://github.com/kubernetes/enhancements/issues/962


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
