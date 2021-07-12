# KEP-1977: Container notifier
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
    - [Phase 1](#phase-1)
    - [Phase 2](#phase-2)
    - [Phase 3](#phase-3)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API Changes](#api-changes)
    - [Phase 1 API Changes](#phase-1-api-changes)
      - [Inline Definition - ContainerNotifier](#inline-definition---containernotifier)
      - [PodNotification](#podnotification)
    - [Phase 2 API Additions](#phase-2-api-additions)
      - [Notification](#notification)
      - [ContainerNotifierHandler](#containernotifierhandler)
    - [Phase 3 API Additions](#phase-3-api-additions)
  - [Kubelet Impact in Phase 2 and Beyond](#kubelet-impact-in-phase-2-and-beyond)
    - [CRI Changes](#cri-changes)
- [Implementation Plan](#implementation-plan)
  - [Phase 1](#phase-1-1)
  - [Phase 2](#phase-2-1)
  - [Phase 3](#phase-3-1)
- [Example Workflows](#example-workflows)
  - [Quiesce Hooks](#quiesce-hooks)
  - [Example Workflow with SIGHUP (Phase 2)](#example-workflow-with-sighup-phase-2)
    - [With Probe (Phase 3)](#with-probe-phase-3)
  - [Example Workflow to Change Log Verbosity (Phase 2)](#example-workflow-to-change-log-verbosity-phase-2)
    - [With Probe (Phase 3)](#with-probe-phase-3-1)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Test Plan](#test-plan)
  - [Unit tests](#unit-tests)
  - [E2E tests](#e2e-tests)
- [Graduation Criteria](#graduation-criteria)
    - [Alpha Graduation](#alpha-graduation)
  - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
  - [Beta -&gt; GA Graduation](#beta---ga-graduation)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

## Summary

This KEP proposes to introduce a mechanism to notify a selected set of `Pod`(s) to run pre-specified commands inlined in those `Pod`(s) specification.

This is achieved by:
* Adding an optional `Notifiers` field in [Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#container-v1-core) type to allow users to specify command(s)/signal(s) which can be executed/sent.
* Introducing a core API object(`PodNotification`) to trigger a specific command/signal specified in a `Pod`.

## Motivation

Being able to take application consistent snapshots/backups is a hard requirement to protect stateful workloads in Kubernetes. It's required for many applications to temporally quiesce themselves before taking snapshots of their PersistentVolume(s) and unquiesce afterwards to ensure application consistency. A mechanism to send a quiesce/unquiesce command to `Pod`(s) is needed.

The first attempt was to introduce an [ExecutionHook](https://github.com/kubernetes/enhancements/blob/master/keps/sig-storage/20190120-execution-hook-design.md) CRD which allows arbitrary commands to be executed against containers, and a controller which manages the lifecycle of the custom resources. This approach, though solves the application quiesce/unquiesce use cases, has been considered neither secure nor generic enough. The controller needs to be granted super user privileges on nodes.

Meanwhile, there are other interesting use cases which requires a similar mechanism:
* Users have been asking a feature to signal `Pod`s in this [issue](https://github.com/kubernetes/kubernetes/issues/24957). For example, sending a signal to a `Pod` to reload configuration, to flush/roll logs, to change log verbosity etc.
* Mechanism to notify `Pod`s on a to-be-evicted node.

With these new motivations, we are proposing a more generic design in this KEP. As the eventual goal is to have Kubelet executing the commands against `Pod`s, this approach is considered to be more secure as it will not widen the scope to grant super user privileges.

### Goals

#### Phase 1

- Users can specify an optional list of commands in `Pod`s specification at container level.
- Users or upper level controllers can request to execute commands in `Pod`s by creation of an API object(`PodNotification`).
- Implement a *trusted* controller which monitors requests to execute commands, notifies the corresponding `Pod`s to run those commands, and updates status of requests corresponding to the execution results.

#### Phase 2

- Introduce an upper level API(`Notification`) which allows users to send `PodNotification` to a set of `Pod`s which can be matched by a label selector.
- Extend `ContainerNotifier` API in Phase 1 to support sending signals to containers.
- Move *trusted* controller logic into Kubelet.

#### Phase 3

- Add a `Probe` to verify the results from the commands/signals if needed.

### Non-Goals

- Writing concrete commands, for example quiesce scripts, which could be executed. It is the responsibility of users who set up pod definitions.

## Proposal

In phase 1, propose to introduce several API changes:
- Adding an optional field `Notifiers` which is a list of `ContainerNotifier`s into `Container`.
- Adding an inlined type `ContainerNotifierHandler` which defines a command.
- Adding a core API type `PodNotification` which defines request type to trigger execution of `ContainerNotifier`(s) in a `Pod`.
- Introduce a new gate `ContainerNotifier` to toggle this feature.

A SINGLE *trusted* controller(Pod Notification Controller) will be implemented to watch `PodNotification` resources, execute the command and update their statuses accordingly.

In phase 2, propose to:
- Add a core API `Notification` type and a controller which processes `Notification` resources.
- Add an inline `Pod` definition for signals and allows the API object to send a request to trigger delivering of those signals.
- Move logic in the Pod Notification controller into Kubelet. Kubelet watches `PodNotification` objects, runs the command and updates statuses of `PodNotification` objects accordingly.

In phase 3, a `Probe` may be added if needed as an inline `Pod` definition to verify the signal is delivered or whether the command is run and results in the desired outcome.

Details of proposed API changes can be found in *API Changes* section  bellow.

### API Changes

#### Phase 1 API Changes

##### Inline Definition - ContainerNotifier

Add []ContainerNotifier to the Container struct:

```
type Container struct {
    ......
    // +optional
    Lifecycle *Lifecycle `json:"lifecycle,omitempty" protobuf:"bytes,12,opt,name=lifecycle"`

    // Notifiers can be triggered by PodNotification resources.
    // Each notifier must have a unique name within the Container.
    // Upon creation of a PodNotification, Container(s) in the specified
    // Pod with a matching notifier name will be executed.
    // +optional
    Notifiers []ContainerNotifier
    ......
}

type ContainerNotifier struct {
    // Name of the ContainerNotifier. Name must be a valid label key. More info:
    // https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set.
    // Each ContainerNotifier within a Container must have a unique name.
    // Names with the prefix `k8s.io/` are reserved for Kubernetes-defined
    // "well-known" notifiers.
    // Names with other prefixes such as `example.com/` or unprefixed are custom names.
    // Immutable.
    Name string

    // The action to take when a matching PodNotification is observed.
    // Exactly-once execution of the action is NOT guaranteed, so it's user's
    // responsibility to make sure the handler is idempotent to get consistent
    // behavior.
    // For example, when a quiesce command has been ran on a database, the
    // database will stay quiesced, and any consecutive execution of the
    // same quiesce command should not have additional impact.
    Handler *ContainerNotifierHandler

    // Number of seconds after which the notifier will time out.
    // Default to 1 second. Minimum value is 1.
    // +optional
    TimeoutSeconds int32
}

// ContainerNotifierHandler specifies an action to take when a ContainerNotifier
// has been triggered.
type ContainerNotifierHandler struct {
    // Exec specifies the action to take.
    // +optional
    Exec *ExecAction `json:"exec,omitempty" protobuf:"bytes,1,opt,name=exec"`
}
```

##### PodNotification

The PodNotification object is a request for execution a ContainerNotifier in a `Pod`. It will be an in-tree API object in the core API group so that Kubelet can watch and trigger the actions.

All containers in the specified `Pod` which have the requested "ContainerNotifier" defined in their spec will have their corresponding "ContainerNotifierHandler" executed.

Note that there is no guarantee of execution ordering between containers.

```
type PodNotification struct {
    metav1.TypeMeta
    // +optional
    metav1.ObjectMeta
    // Spec defines the behavior of a notification.
    // +optional
    Spec PodNotificationSpec
    // Status defines the current state of a notification.
    // +optional
    Status PodNotificationStatus
}

type PodNotificationSpec struct {
    // The name of the Pod to which notification will be addressed.
    // Note that a PodNotificaion only notifies a Pod in the same namespace.
    // The PodNotification fails immediately if the specified Pod does not exist.
    // Required.
    // Immutable.
    PodName string


    // The name of the ContainerNotifier to trigger.
    // The PodNotification completes immediately if no Container in the
    // Pod has the requested ContainerNotifier defined.
    // Required.
    // Immutable.
    Notifier string
}
```

###### PodNotificationStatus
`PodNotificationStatus` represents the current status of a `PodNotification`. It contains a list of `ContainerNotificationStatus`, a start time stamp suggesting the first time the `PodNotification` has been observed, a completion time stamp suggesting the end of the notification, and a succeeded flag representing the final state of the `PodNotification` upon completion.

A `PodNotification` is considered to be successful(i.e., with succeeded flag set to `true`) only if the notification have been successfully executed in all qualified containers in the specified `Pod`.

Note that the `PodNotification` controller (or Kubelet in phase 2) will NOT retry upon failures/errors returned from `ContainerNotifier`. Top level controllers or users can create another `PodNotification` to do retries.

```
type PodNotificationStatus struct {
    // A list of ContainerNotificationStatus, with each item representing the
    // execution result of the corresponding ContainerNotifier in Container(s)
    // in the specified Pod.
    // +optional
    Containers []ContainerNotificationStatus

    // If set, it suggests the timestamp(UTC) at when the first time the
    // PodNotification has been acknowledged by the PodNotification Controller.
    // +optional
    StartTime *metav1.Time

    // If set, it suggests the timestamp(UTC) at when the PodNotification has
    // been marked as completed. A PodNotification is completed when the
    // action has been executed in all eligible Containers in this Pod.
    // Note that a completed PodNotification does NOT mean the notification
    // has been executed successfully. Refer to `Succeeded` field for more details.
    // +optional
    CompleteTime *metav1.Time

    // State represents the current stage of the PodNotification.
    // Default to PodNotificationStateNew.
    State PodNotificationStateType

    // Error represents the failing reason of the PodNotification.
    // +optional
    Error *PodNotificationError
}

// PodNotificationStateType represents the current stage of the PodNotification.
type PodNotificationStateType string
const(
  // PodNotificationStateNew suggests the PodNotification is newly created and
  // has not been picked up yet.
  PodNotificationStateNew PodNotificationStateType = "New"

  // PodNotificationStateSucceeded suggests the PodNotidication has been
  // successfully executed against all qualified Containers.
  PodNotificationStateSucceeded PodNotificationStateType = "Succeeded"

  // PodNotificationStateFailed suggests the PodNotidication has been executed
  // however at least one of the qualified Containers returned an error. Details
  // can be found in 'Containers'.
  PodNotificationStateFailed PodNotificationStateType = "Failed"
)

// ContainerNotificationStatus represents the execution state of the handler in
// the requested ContainerNotifier.
type ContainerNotificationStatus struct {
    // The name of the Container in which the handler has been executed.
    // Required.
    Name string

    // If set, it suggests the timestamp at when the first time the execution
    // of the action has started in UTC.
    // +optional
    StartTime *metav1.Time

    // If set, it suggests the timestamp at when the first time the execution
    // of the handler has completed in UTC.
    // Note that `CompleteTime` does NOT mean the handler has been executed
    // successfully. Refer to `Succeeded` field for more details.
    // +optional
    CompleteTime *metav1.Time

    // Succeeded will be set to true when the handler has been executed
    // successfully, and it will be set to false otherwise. It will not be
    // specified if the status of execution is unknown(i.e., in execution).
    // +optional
    Succeeded *bool

    // If specified, Error represents the last error encountered when executing
    // the handler.
    // +optional
    Error *PodNotificationError
}

type PodNotificationError struct {
    // Type of the error
    // Required.
    Type PodNotificationErrorType

    // A human-readable message indicating details why the action failed.
    Message string

    // It indicates when the error occurred
    Timestamp metav1.Time
}

// PodNotificationErrorType represents the type of error occurred in a
// Notifiication.
type PodNotificationErrorType string

// More error types could be added, e.g., Forbidden, Unauthorized, etc.
const (
    // Pod not found error
    PodNotificationErrorPodNotFound PodNotificationErrorType = "PodNotFound"

    // The execution of a handler timed out.
    PodNotificationErrorHandlerTimeout PodNotificationErrorType = "HandlerTimeout"
)
```

Creators of `PodNotification` resources are responsible of the full lifecycle of them.

#### Phase 2 API Additions

We propose to further extend the API in two aspects:
- A `Notification` type which allows users or upper level controllers to specify a label selector to target a set of `Pod`s to notify. This makes the feature much user friendly for use cases like application consistent snapshot, signaling a group of `Pod`s.
- Enhancement on `ContainerNotifierHandler` to support signal.

##### Notification
```
type Notification struct {
    metav1.TypeMeta
    // +optional
    metav1.ObjectMeta

    // Spec specifies the desired state of a Notification.
    // +optional
    Spec NotificationSpec
    // Status indicates the current state of a Notification.
    // +optional
    Status NotificationStatus
}

type NotificationSpec struct {
    // A label query over Pods under the same namespace of the Notification
    // to notify.
    // Note: Only Pods in the ‘Running’ phase are eligible for notification.
    // Ref: kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-phase
    // Required.
    Selector metav1.LabelSelector

    // Policy specifies pod selection policy. If not specified, default to
    // "AllPods".
    // +optional.
    Policy *PodSelectionPolicy

    // The name of the ContainerNotifier to trigger.
    Notifier string

    // Parallelism specifies the maximum number of Pods to notify at any given
    // time concurrently, which effectively is the same as the number of
    // uncompleted PodNotifications created from this Notification.
    // If not specified or set to 0, all eligible Pods will be notified.
    // If set to a number greater than the number of eligible Pods, all
    // eligible Pods will be notified concurrently.
    // Setting this field to a negative number fails the Notification immediately.
    Parallelism int
}

// PodSelectionPolicy specifies a policy to scope Pods which are qualified for
// a Notification.
type PodSelectionPolicy string

const (
    // PodSelectionPolicyAllPods means all Pods matches the specified label
    // selector in Notification will be notified. If this policy is specified,
    // CompleteTime field of the NotificationStatus will not be set.
    PodSelectionPolicyAllPods PodSelectionPolicy = "AllPods"


    // PodSelectionPolicyPreExistingPods means only those Pods created before
    // the Notification(i.e., create timestamps smaller than Notification's) are
    // qualified for notification.
    // Newly created Pods after the Notification will not be notified.
    PodSelectionPolicyPreExistingPods PodSelectionPolicy = "PreExistingPods"
)

// NotificationStatus represents the current state of a Notification
type NotificationStatus struct {
    // FailedCount indicates the current number of failed PodNotification
    // resources created from this Notification.
    // +optional
    FailedCount int

    // SucceededCount indicates the current number of succeeded PodNotification
    // resources created from this Notification.
    // +optional
    SucceededCount int

    // If set, it suggests the timestamp(UTC) at when the first time the
    // Notification has been acknowledged by the Notification Controller.
    // +optional
    StartTime *metav1.Time

    // If set, it suggests the timestamp(UTC) at when the Notification has
    // been marked as completed. A Notification is completed when all
    // PodNotifications created from this Notification have completed.
    // Note that this field will never be set if "AllPods" is specified as
    // pod selection policy.
    // +optional
    CompleteTime *metav1.Time
}
```

##### ContainerNotifierHandler
```
// ContainerNotifierHandler defines a specific action to be executed or a signal which will be delivered.
// Exactly one of Exec and Signal should be set.
type ContainerNotifierHandler struct {
    // Exec specifies the action to take.
    // +optional
    Exec *ExecAction `json:"exec,omitempty" protobuf:"bytes,1,opt,name=exec"`

    // Signal specifies a signal to send to the container
    // +optional
    // define constants for signals?
    // validate the signals are valid?  windows?
++    Signal string
}
```

#### Phase 3 API Additions

If needed, we may add a probe to ContainerNotifier in Phase 3 to verify that the ContainerNotifierHandler has delivered the signal or executed the command.

Phase 3 is more loosely decided than phases 1 and 2 at this stage, we will have to revisit the design details.

```
type ContainerNotifier struct {
    // Name of the ContainerNotifier specified as a DNS_LABEL. Each
    // ContainerNotifier within a Container must have a unique name.
    // Names with the prefix `k8s.io/` are reserved for Kubernetes-defined
    // "well-known" notifiers.
    // Names with other prefixes such as `example.com/` are custom names.
    // Immutable.
    Name string

    // The action to take when a matching PodNotification is observed.
    // Exactly-once execution of the action is NOT guaranteed, so it's user's
    // responsibility to make sure the handler is idempotent to get consistent
    // behavior.
    // For example, when a quiesce command has been ran on a database, the
    // database will stay quiesced, and any consecutive execution of the
    // same quiesce command should not have additional impact.
    Handler *ContainerNotifierHandler

    // Number of seconds after which the notifier will time out.
    // Default to 1 second. Minimum value is 1.
    // +optional
    TimeoutSeconds int32

+   // Add a Probe in Phase 3. Probe defined in the core API here will be used.
+   Probe *Probe
}
```

### Kubelet Impact in Phase 2 and Beyond

When moving the logic from the PodNotification controller to Kubelet, we foresee the following changes to happen in Kubelet:

- Kubelet will watch PodNotification resources for Pods on the node it's running.

- Kubelet will execute command/send signal specified in ContainerNotifier against containers. CRI changes are required to support signals.

- Kubelet will update PodNotification status.(Possible QPS concern when trying to update 100+ of PodNotifications concurrently).

- Kubelet will have some method of measuring the success/failure of a PodNotification.
* For exec, it will depend on the return value of the ExecInContainer method.
* For signals, it will depend on the new CRI changes.

Kubelet will not retry if the ContainerNotifier call fails. It is up to upper level controllers or users who have requested the PodNotification to manually retry by creating another PodNotification resource.

#### CRI Changes

To support signals directly, we might need to enhance CRI. One potential approach is to add a new "Signal" interface with an input parameter "signal string". Initially, a predefined set of signals will be supported. i.e., "SIGHUP". The initial set of signals is yet to be defined.

```
// Runtime is the interface to execute the commands and provide the streams.
type Runtime interface {
        Exec(containerID string, cmd []string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan remotecommand.TerminalSize) error
+        Signal(containerID string, signal string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan remotecommand.TerminalSize) error

        Attach(containerID string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan remotecommand.TerminalSize) error
        PortForward(podSandboxID string, port int32, stream io.ReadWriteCloser) error
}
```

To support "SIGHUP" directly, we can translate that into a call to the docker client method `ContainerKill`.

```
https://github.com/moby/moby/blob/master/client/container_kill.go#L9
```

We can add a `SignalKill` method in `~/go/src/k8s.io/kubernetes/pkg/kubelet/dockershim/libdocker/kube_docker_client.go` and call docker client `ContainerKill`, similar to how `CreateExec` method calls docker client `ContainerExecCreate`.

## Implementation Plan

### Phase 1

Phase 1 API definition will be added to Kubernetes. Controller logic implementation in this phase will not happen in Kubelet. Instead, it will be implemented in a single *trusted* controller which watches `PodNotification`s and execute the command via exec(with the goal being to move that into Kubelet in the next steps). In this phase, only exec will be implemented.

The PodNotifier controller will be created in a separate repo under Kubernetes. The repo will be sponsored by sig-node: `https://github.com/kubernetes/containernotifier`.

### Phase 2

* Introduce `Notification` API and its corresponding controller.
* Make CRI changes to support signals.
* Move PodNotification controller logic to Kubelet.

### Phase 3

* Add Probe if needed.

## Example Workflows

### Quiesce Hooks

For example, there are 3 commands that need to be executed sequentially to quiesce a mysql database before taking a snapshot of its persistent volumes: *lockTables*, *flushDisk*, and *fsfreeze*. After the snapshots have been taken, there are 2 commands to be executed sequentially to unquiesce it: *fsUnfreeze*, *unlockTables*. For simplicity, assume we only need to run *fsfreeze* for one volume and we only need to run each command in one container in one pod. These commands are defined in "Notifiers []ContainerNotifier" inside the Container.

1. lockTables
Upper level controller creates a `PodNotification` object to request the *lockTables* `ContainerNotifier`.
PodNotification controller(or Kubelet in phase 2) noticed the creation of the `PodNotification` object. It starts to run the *lockTables* command specified in the *ContainerNotifier* and updates the `ContainerNotificationStatus` to set the StartTimestamp.

When the command finishes successfully, PodNotification controller(or Kubelet in phase 2) sets the Succeed field in ContainerNotificationStatus to True.
If it fails or times out, PodNotification controller(or Kubelet in phase 2) sets the Succeed field to False.

2. flushDisk
If the *lockTables* command succeeds, upper level controller will proceed to create another `PodNotification` object to request the *flushDisk* `ContainerNotifier`.

PodNotification controller(or Kubelet in phase 2) starts to run the *flushDisk* command specified in the `ContainerNotifier` and updates the `PodNotificationStatus` correspondingly.

When the *flushDisk* command finishes successfully, PodNotification controller(or Kubelet in phase 2) sets the Succeed field in the `ContainerNotificationStatus` to True.
If it fails or times out, PodNotification controller(or Kubelet in phase 2) sets the Succeed field to False.

If the *lockTables* command fails, upper level controller will create a new `PodNotification` object to request the *unlockTables* `ContainerNotifier`. It will not proceed to the next PodNotification and the snapshot creation will be marked as failure.

3. fsfreeze
If the *flushDisk* command succeeds, upper level controller will create a new `PodNotification` object to request the *fsfreeze* `ContainerNotifier`.

PodNotification controller(or Kubelet in phase 2) starts to run the *fsfreeze* command specified in the `ContainerNotifier` and updates the `PodNotificationStatus` correspondingly.

When the *fsfreeze* command finishes successfully, PodNotification controller(or Kubelet in phase 2) sets the Succeed field in the `ContainerNotificationStatus` to True.
If it fails or times out, PodNotification controller(or Kubelet in phase 2) sets the Succeed field to False.

4. Take snapshot
If the *fsfreeze* command succeeds, upper level controller will proceed to take a snapshot.

5. fsUnfreeze
Upper level controller creates a `PodNotification` object to request the *fsUnfreeze* `ContainerNotifier` as long as step 3 has been executed.
If *fsFreeze* in step 3 is called, *fsUnfreeze* should always be called.

PodNotification controller(or Kubelet in phase 2) starts to run the *fsUnfreeze* command specified in the `ContainerNotifier` and updates the `PodNotificationStatus`correspondingly.

When the *fsUnfreeze* command finishes successfully, PodNotification controller(or Kubelet in phase 2) sets the Succeed field in the `ContainerNotificationStatus` to True, otherwise, it sets the Succeed field to False.

6. unlockTables
Upper level controller proceeds to create a `PodNotification` object to request the *unlockTables* `ContainerNotifier`. If `lockTables` is called in step 1, `unlockTables` should always be called.

PodNotification controller(or Kubelet in phase 2) starts to run the *unlockTables* command specified in the `ContainerNotifier` and updates the `PodNotificationStatus` correspondingly.

When the *unlockTables* command finishes successfully, PodNotification controller(or Kubelet in phase 2) sets the Succeed field in the `ContainerNotificationStatus` to True, otherwise it sets the Succeed field to False.

7. It is the upper level controller's responsibility to make sure unquiesce is always called after a quiesce command for the snapshot use case. This means *fsUnfreeze* is always called after *fsFreeze* and *unlockTables* is always called after *lockTables*.

8. Upper level controller is responsible to delete all `PodNotification` objects it has created after the commands have completed.

9. Upper level controller is responsible to handle retries if a command fails by creating another `PodNotification` object.

### Example Workflow with SIGHUP (Phase 2)

This example involves a signal so it will be in phase 2 and beyond.

For example, a user wants to send SIGHUB to a container in a `Pod`. This signal is defined in the `ContainterNotificationHandler` inside the `Container`.

The user creates a `PodNotification` object to request the SIGHUP `ContainerNotifier`.
Kubelet watches the `PodNotification` object and gets notified. It sends the SIGHUP signal defined in the `ContainterNotificationHandler` to the container. This is similar to "docker kill --signal=SIGHUP my_container".

#### With Probe (Phase 3)

If Probe is implement, Kubelet also sends a probe to check if the container is still running. If the probe detects that the container is stopped, Kubelet sets the Succeed field in `ContainerNotificationStatus` to True.
If the container is still running, Kubelet will retry probes periodically until it times out. When that happens, it stops retrying and sets the `ContainerNotificationStatus` Succeed field to False.

### Example Workflow to Change Log Verbosity (Phase 2)

This example involves a signal so it will be in phase 2 and beyond.

For example, the user wants to change log level to verbose in a container in a Pod. This signal is defined in the ContainterNotifierAction inside the Container.

External controller creates a Notification object to request the ChangeLogLevel ContainerNotifier to change log to verbose.
Kubelet watches the Notification object and gets notified. It sends the ChangeLogLevel to verbose signal defined in the ContainerNotifierAction to the container.

#### With Probe (Phase 3)

If Probe is implemented, Kubelet also sends a probe to check the log level in the container.

If the probe detects that the log level inside the container is indeed verbose, Kubelet sets the Succeed field in ContainerNotificationStatus to True.
If the probe detects that the log level is not changed yet, Kubelet will retry probes periodically until it times out. When that happens, it stops retrying and sets the ContainerNotificationStatus Succeed field to False.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Test Plan

### Unit tests

* Unit tests for container notifier controller will be added in phase 1.

### E2E tests

* E2e tests for creating a Notification object to request a ContainerNotifier when the feature flag is enabled.

## Graduation Criteria

#### Alpha Graduation

* Feature Flag is present.
* Container notifier controller is implemented in a sig-node sponsored repo.
* E2E tests are implemented.

### Alpha -> Beta Graduation

* Container notifier controller logic is moved to Kubelet.
* Signal supported is added to Kubelet.
* After the controller logic is moved to Kubelet, a decision will need to be
  made on whether a Probe is needed and whether we are ready to go Beta.
  If Probe is added as a separate feature, then we can go Beta with just
  ContainerNotifier.
* Feedback has been gathered from users.
* A Blog post has been written and published on the Kubernetes blog.

### Beta -> GA Graduation

* Gather feedback from users and address feedback.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/20190731-production-readiness-review-process.md.

The production readiness review questionnaire must be completed for features in
v1.19 or later, but is non-blocking at this time. That is, approval is not
required in order to be in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.

-->


### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: ContainerNotifier
    - Components depending on the feature gate: kube-apiserver
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
      Yes. Kube API Server needs to be restarted.
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
      No.

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.
  No.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Also set `disable-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?).
  Yes. If it is disabled once it has been enabled, user can no longer leverage
  the ContainerNotifier feature to request to execute command in a Pod.

* **What happens if we reenable the feature if it was previously rolled back?**
  If we reenable the feature if it was previously rolled back, user can continue
  to leverage the ContainerNotifier feature to request to execute a command in a Pod.

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling or disabling feature
  gates. However, unit tests in each component dealing with managing data, created
  with and without the feature, are necessary. At the very least, think about
  conversion tests if API types are being modified.
  We can add unit tests for enabling or disabling the feature gate.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  Think about both cluster-level services (e.g. metrics-server) as well
  as node-level agents (e.g. specific version of CRI). Focus on external or
  optional services that are needed. For example, if this feature depends on
  a cloud provider API, or upon an external software-defined storage or network
  control plane.

  For each of these, fill in the following—thinking about running existing user workloads
  and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
  For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

## Drawbacks

None.

## Alternatives

This was initially proposed as a CRD approach in the [ExecutionHook KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-storage/20190120-execution-hook-design.md). However this approach means an external controller would be responsible for running exec on a pod. Having Kubelet execute the hooks instead of an external controller would be more secure as an external controller is considerably easier to be compromised than kubelet and be able to arbitrarily exec on any pod.

## Infrastructure Needed (Optional)

None.
