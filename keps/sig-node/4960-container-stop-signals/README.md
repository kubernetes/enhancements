
# KEP-4960: Container Stop Signals 

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API](#api)
  - [Cross validation with Pod spec.os.name](#cross-validation-with-pod-specosname)
  - [CRI API](#cri-api)
  - [Container runtime changes](#container-runtime-changes)
  - [Windows support](#windows-support)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Upgrade](#upgrade)
    - [Downgrade](#downgrade)
  - [Version Skew Strategy](#version-skew-strategy)
    - [Version skew with CRI API and container runtime](#version-skew-with-cri-api-and-container-runtime)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Container runtimes let you define a [STOPSIGNAL](https://docs.docker.com/reference/dockerfile/#stopsignal) to let your container images change which signal is delivered to kill the container. Currently you can only configure this by defining STOPSIGNAL in the container image definition file before you build the image. This becomes difficult to change when you’re using prebuilt images. Kubernetes has no equivalent for STOPSIGNAL as part of Pod or Container APIs. This KEP proposes to add support to configure custom stop signals for containers from the ContainerSpec.

## Motivation

Container runtimes like Docker let you configure the stop signal with which a container would be killed when you start a container. This can be configured either from the container image definition file itself with the STOPSIGNAL instruction or by using the `--stop-signal` flag when starting a container with the respective CLI tool for your runtime. Currently there is no equivalent to this in the Kubernetes APIs.

While managing containers with Kubernetes, if you want to customize an existing image by changing its predefined stop signal or override the default stop signal of SIGTERM, currently you would have to rebuild the container image and update the stop signal at the image definition level. 

Having stop signal as a first class citizen in the Pod's container specification would make it easier for users to set custom stop signals for their containers across all types of workloads.

### Goals

- Add a new Stop lifecycle handler to container lifecycle which can be configured with a Signal option, which takes a string value
- Update the CRI API to pass down the stop signal to the container runtime via ContainerConfig
- Update the implementation of the StopContainer method in container runtimes to use the container’s stop signal defined in the container spec (if present) to kill containers
- Add support to show the effective stop signal of containers in the container status field in the pod status

### Non-Goals

- Change any existing behaviour with how stop signals work when defined in the container image

## Proposal

### API

A new StopSignal lifecycle hook will be added to container lifecycle. The StopSignal lifecycle hook can be configured with a signal, which is of type `Signal`. This new `Signal` type can take a string value, and can be used to define a stop signal for containers when creating Pods. `Signal` will hold string values which can be mapped to Go's syscall.Signal. For example, see the [list of signals supported in Linux environments by moby](https://github.com/containerd/containerd/blob/main/vendor/github.com/moby/sys/signal/signal_linux.go). If the user doesn't define a particular stop signal, the behaviour would default to what it is today and fallback to the stop signal defined in the container image or use the default stop signal of the container runtime (SIGTERM in case of containerd, CRI-O).

```go
// pkg/apis/core/types.go
type Signal string //parseable into Go's syscall.Signal

type Lifecycle struct {
  // ...
  // +optional
  StopSignal *Signal
}
```

Users will be able to define custom stop signals for their containers like so:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
    lifecycle:
      stopSignal: SIGUSR1
```

The stop signal would also be shown in the containers' status. The value of the stop signal shown in the status can be from the spec, if a stop cycle is defined in the spec, else it will be the effective stop signal which is used by the container runtime to kill your container. This can either be read from the container image or will be the default stop signal of the container runtime. Users will be able to see a container's stop signal in its status even if they're not using a custom stop signal from the spec.

```yaml
status:
  containerStatuses:
  - containerID: containerd://19d9bb24f5d6633dddf8f97d0b2aed1158ceb1030440082f3f3dbea8ce4d2be6
    image: nginx:1.14.2
    lastState: {}
    lifecyle:
      stopSignal: SIGUSR1
    name: redis
    ready: true
    restartCount: 0
    started: true
    state:
      running:
        startedAt: "2025-01-16T09:13:15Z"
```

### Cross validation with Pod spec.os.name

In order to make sure that users are setting valid stop signals for the nodes the pods are being scheduled to, we cross validate the `ContainerSpec.Lifecycle.StopSignal` with `spec.os.name`. Here are the details of this validation:
- We require `spec.os.name` to be set to a valid value (`linux` or `windows`) to use `ContainerSpec.Lifecycle.StopSignal`.
- We have a list of valid stop signals for both linux and windows nodes (as shown below). If the Pod OS is set to `linux`, only the signals supported for `linux` would be allowed. 
- Similarly for Pods with OS set to `windows`, we only allow SIGTERM and SIGKILL as valid stop signals.

The full list of valid signals for the two platforms are as follows:

```go
var supportedStopSignalsLinux = sets.New(
	core.SIGABRT, core.SIGALRM, core.SIGBUS, core.SIGCHLD,
	core.SIGCLD, core.SIGCONT, core.SIGFPE, core.SIGHUP,
	core.SIGILL, core.SIGINT, core.SIGIO, core.SIGIOT,
	core.SIGKILL, core.SIGPIPE, core.SIGPOLL, core.SIGPROF,
	core.SIGPWR, core.SIGQUIT, core.SIGSEGV, core.SIGSTKFLT,
	core.SIGSTOP, core.SIGSYS, core.SIGTERM, core.SIGTRAP,
	core.SIGTSTP, core.SIGTTIN, core.SIGTTOU, core.SIGURG,
	core.SIGUSR1, core.SIGUSR2, core.SIGVTALRM, core.SIGWINCH,
	core.SIGXCPU, core.SIGXFSZ, core.SIGRTMIN, core.SIGRTMINPLUS1,
	core.SIGRTMINPLUS2, core.SIGRTMINPLUS3, core.SIGRTMINPLUS4,
	core.SIGRTMINPLUS5, core.SIGRTMINPLUS6, core.SIGRTMINPLUS7,
	core.SIGRTMINPLUS8, core.SIGRTMINPLUS9, core.SIGRTMINPLUS10,
	core.SIGRTMINPLUS11, core.SIGRTMINPLUS12, core.SIGRTMINPLUS13,
	core.SIGRTMINPLUS14, core.SIGRTMINPLUS15, core.SIGRTMAXMINUS14,
	core.SIGRTMAXMINUS13, core.SIGRTMAXMINUS12, core.SIGRTMAXMINUS11,
	core.SIGRTMAXMINUS10, core.SIGRTMAXMINUS9, core.SIGRTMAXMINUS8,
	core.SIGRTMAXMINUS7, core.SIGRTMAXMINUS6, core.SIGRTMAXMINUS5,
	core.SIGRTMAXMINUS4, core.SIGRTMAXMINUS3, core.SIGRTMAXMINUS2,
	core.SIGRTMAXMINUS1, core.SIGRTMAX)

var supportedStopSignalsWindows = sets.New(core.SIGKILL, core.SIGTERM)
```

You can find the validation logic implemented in [this commit](https://github.com/kubernetes/kubernetes/pull/130556/commits/0380f2c41cdc4df992294603f7844709072628b1#diff-c713e8919642d873fdf48fe8fb6d43e5cb2f53fd601066ff53580ea655948f0d).

### CRI API

The CRI API would be updated so the stop signal in the container spec (if it is not nil or unset) is sent to the container runtime via ContainerConfig. This would be passed down to the container runtime's StopContainer method ultimately:

```diff
// ContainerConfig holds all the required and optional fields for creating a
// container.
message ContainerConfig {
  // ...
+ Signal stop_signal = 18;
}

+ enum Signal {
+   RUNTIME_DEFAULT   = 0;
+   SIGABRT           = 1;
+   SIGALRM           = 2;
+   ...
+   SIGRTMAX          = 65;
+ }
```

We can pass the container's stop signal to the container runtime with this new field to ContainerConfig.

```diff
// pkg/kubelet/kuberuntime/kuberuntime_container.go

// generateContainerConfig generates container config for kubelet runtime v1.
func (m *kubeGenericRuntimeManager) generateContainerConfig(...) (*runtimeapi.ContainerConfig, func(), error) {
  // ...
  config := &runtimeapi.ContainerConfig{
    Metadata: &runtimeapi.ContainerMetadata{
      Name:    container.Name,
      Attempt: restartCountUint32,
    },
    Image:       &runtimeapi.ImageSpec{Image: imageRef, UserSpecifiedImage: container.Image},
    Command:     command,
    Args:        args,
    WorkingDir:  container.WorkingDir,
    Labels:      newContainerLabels(container, pod),
    Annotations: newContainerAnnotations(container, pod, restartCount, opts),
    Devices:     makeDevices(opts),
    CDIDevices:  makeCDIDevices(opts),
    Mounts:      m.makeMounts(opts, container),
    LogPath:     containerLogsPath,
    Stdin:       container.Stdin,
    StdinOnce:   container.StdinOnce,
    Tty:         container.TTY,
	}

+ stopsignal := getContainerConfigStopSignal(container)

+ if stopsignal != nil {
+   config.StopSignal = *stopsignal
+ }
  // ...
}
```

Since the new stop lifecycle is optional, the default stop signal for a container can be unset or nil. In this case, the container runtime will fallback to the existing behaviour. 

Additionally, the stop signal would also be added to `ContainerStatus` (as `containerStatus[].StopSignal`) so that we can pass the stop signal extracted from the image/container runtime back to the container status at the Kubernetes API level.

### Container runtime changes

Once the stop signal from `containerSpec.Lifecycle.StopSignal` is passed down to the container runtime via `ContainerConfig` during creation/updation of the container, we can use the value of the stop signal from the container runtime's implementation of `stopContainer` method. In the case of containerd, it would look like this:

```diff
//internal/cri/server/container_stop.go

func (c *criService) StopContainer(ctx context.Context, r *runtime.StopContainerRequest) (*runtime.StopContainerResponse, error) {
// ...
-	if err := c.stopContainer(ctx, container, time.Duration(r.GetTimeout())*time.Second); err != nil {
+ 	if err := c.stopContainer(ctx, container, time.Duration(r.GetTimeout())*time.Second, container.Config.GetStopSignal().String()); err != nil {
		return nil, err
	}
// ...
}

-func (c *criService) stopContainer(ctx context.Context, container containerstore.Container, timeout time.Duration) error {
+func (c *criService) stopContainer(ctx context.Context, container containerstore.Container, timeout time.Duration, containerStopSignal string) error {
//...
    if timeout > 0 {
	    stopSignal := "SIGTERM"
-       if container.StopSignal != "" {
+       if containerStopSignal != "" {
+   	    stopSignal = containerStopSignal 
+       } else if  container.StopSignal != "" {
			stopSignal = container.StopSignal
		} else {
// rest of the code...
```

The signal that we get from `ContainerConfig` can be validated with [ParseSignal](https://github.com/containerd/containerd/blob/main/vendor/github.com/moby/sys/signal/signal.go#L38) to further validate that we've received a valid stop signal. Also here `container.StopSignal` is reading the stop signal from the image. We can add another condition before that to use the stop signal defined in spec if there is one. If nothing is defined in the spec ("" or unset), containerd behaves like how it is today. Also note that `SIGTERM` is hardcoded in containerd's stopContainer method as the default stop signal to fallback to, in case the image doesn't defined a stop signal. Similar logic in also present in CRI-O [here](https://github.com/cri-o/cri-o/blob/main/internal/oci/container.go#L259-L272).

Find the entire diff for containerd which was done for the POC [here](https://github.com/containerd/containerd/compare/main...sreeram-venkitesh:containerd:added-custom-stop-signal?expand=1).

### Windows support

Currently using the hcsshim is the only way to run containers on Windows nodes. hcsshim [supports SIGTERM and SIGKILL and a few Windows specific CTRL events](https://github.com/microsoft/hcsshim/blob/e5c83a121b980b1b85f4df0813cfba2d83572bac/internal/signals/signal.go#L74-L126). After discussing with SIG Windows, we came to the decision that for Windows Pods we'll only support SIGTERM and SIGKILL as the valid stop signals. The behaviour of how kubelet handles stop signals is not different for Linux and Windows environments and the CRI API works in both cases.

We will have additional validation for Windows Pods to restrict the set of valid stop signals to SIGTERM and SIGKILL. There will be an admission check that validates that the stop signal is only set to either SIGTERM or SIGKILL if `spec.os.name` == windows. This OS specific cross validation is further described in [Cross validation with Pod spec.os.name](#cross-validation-with-pod-specosname).

### User Stories (Optional)

#### Story 1

Kubernetes by default sends a SIGTERM to all containers while killing them. When running nginx on Kubernetes, this can result in nginx dropping requests as reported [here](https://github.com/Kong/kubernetes-ingress-controller/pull/283). The current solution for this issue would be to build custom images with a SIGQUIT stop signal or to write a PreStop lifecycle hook that manually kills the process gracefully, which is what is done in the PR. The PreStop hook solution looks like the following:

```yaml
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "kill -SIGUSR1 $(pidof my-app)"]
```

If we had stop signal support at the Container spec level, this would've been easier and straightforward to implement. Users wouldn't have to patch the applications running on Kubernetes to handle different termination behavior. This is also similar to [this issue](https://github.com/github/resque/pull/21).

### Risks and Mitigations

We'll be adding the complexity of signal handling to the pod/container spec. If users define an signal that is not handled by their containers, this can lead to pods hanging. In such a scenario, if the stop signal is not properly handled, the pod will hang for the terminationGracePeriodSeconds before it is forcefully killed with SIGKILL. In the default case where the terminationGracePeriodSeconds is 30 seconds, the pods will hang for 30 seconds before being killed.

## Design Details

On top of the details described in the [Proposal](#proposal), these are some details on how exactly the new field will work.
- `ContainerSpec.Lifecycle.StopSignal` is totally optional and can be a nil value. In this case, the stop signal defined in the container image or the container runtime's default stop signal (SIGTERM for containerd and CRI-O) would be used.
- If set, `ContainerSpec.Lifecycle.StopSignal` will override the stop signal set from the container image definition.
- The order of priority for the different stop signals would look like this
	`Stop signal from Container Spec > STOPSIGNAL from container image > Default stop signal of container runtime`

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

Alpha:
- Test that the validation fails when given a non string value for container lifecycle StopSignal hook
- Test that the validation passes when given a proper string value representing a standard stop signal
- Test that the validation fails when we configure a custom stop signal with the feature gate disabled
- Test that the validation returns the appropriate error message when an invalid string value is given for the stop signal
- Tests for verifying behavior when feature gate is disabled after being used to create Pods where the stop signal field is used
- Tests for verifying behavior when feature gate is reenabled after being disabled after creating Pods with stop signal
- Test that the validation allows SIGTERM and SIGKILL signals for Windows Pods 
- Test that the validation doesn't allow non valid signals for Windows Pods
  
##### e2e tests

- Test that containers are killed with the right stop signal when a stop signal is passed
- Test that containers are killed with SIGTERM when no stop signal is passed
- Test that the Status returns the correct stop signal in all the following cases:
   - When stop signal is defined in the Container Spec (Status should have signal is defined in the Spec)
   - When stop signal is only defined in the container image (Status should have the signal defined in the image)
   - When no stop signal is defined (Stop signal in Status should be SIGTERM)
- Test that the stop signal is gracefully degraded when stop signal is specified but the container runtime is on a version that doesn't support the implementation
- Test that the feature is gracefully degraded when stop signal is not supported in Kubelet but is supported in the container runtime
- Test that Windows Pods can be created with the correct stop signals and that the containers are killed with the respective signals

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- CRI API implementation completed in containerd marked as experimental
- CRI API implementation completed for CRI-O
- Best effort support for Windows for SIGTERM and SIGKILL signals
- Initial e2e tests completed and enabled, testing the feature against containerd
- Unit tests for validation, e2e tests for version skew

#### Beta

- Further test Windows support and add e2e tests
- Gather feedback from developers and surveys
- e2e tests for CRI-O

#### GA

- Both containerd and CRI-O having a GA release with containerStopSignal parameter implemented for `stopContainer` method.
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for user feedback

### Upgrade / Downgrade Strategy

#### Upgrade

When upgrading to a new Kubernetes version which supports Container Stop Signals, users can enable the feature gate and start using the feature. If the user is running an older version of the container runtime, the feature will be gracefully degraded as mentioned [here](https://www.kubernetes.dev/docs/code/cri-api-version-skew-policy/#version-skew-policy-for-cri-api) in the CRI API version skew doc. In this case the user will be able to set a StopSignal lifecycle hook in the Container spec, but the kubelet will not pass this value to the container runtime when calling the `runtimeService.stopContainer` method. The container status would also not have stop signal since the container runtime is not updated to return the effective stop signal extracted from the image.

#### Downgrade

If the kube-apiserver or the kubelet's version is downgraded, you will no longer be able to create or update container specs to include the StopSignal lifecycle hook. Existing containers which have the field set would not be cleared. If you're running a version of the kubelet which doesn't support ContainerStopSignals, the CRI API wouldn't pass the stop signal to the runtime as part of ContainerConfig. Even if the container runtime is on a newer version supporting stop signal, it would handle this and default to the stop signal defined in the image or to SIGTERM.

### Version Skew Strategy

Both kubelet and kube-apiserver will need to enable the feature gate for the full featureset to be present and working. If both components disable the feature gate, this feature will be completely unavailable.

If only the kube-apiserver enables this feature, validation will pass, but kubelet won't understand the new lifecycle hook and will not add the stop signal when creating the ContainerConfig. The StopSignal lifecycle hook would be silently dropped by the kubelet before sending the ContainerConfig to the runtime since the feature gate is disabled in the kubelet.

If only the kubelet has enabled this feature, you won't be able to create a Pod which has a StopSignal lifecycle hook via the apiserver and hence the feature won't be usable even if the kubelet supports it. `containerSpec.Lifecycle.StopSignal` can be an empty value and kubelet functions as if no custom stop signal has been set for any container.

For static pods, if the feature is only enabled in the kube-apiserver and not in the kubelet, the pod will be created but the StopSignal lifecycle hook would be dropped by the kubelet since the feature gate is not enabled in the kubelet. If the feature is enabled on the kubelet but not in the kube-apiserver, the pod would have the StopSignal lifecycle hook, but the apiserver wouldn't report the pod as having a StopSignal lifecycle hook since the feature is disabled in the kube-apiserver. This would be the case if we create regular pods with StopSignal and later turn off the feature in the kube-apiserver. The pods with a stop signal would continue working in this case.

#### Version skew with CRI API and container runtime

As described above in the upgrade/downgrade strategies,

- **If the container runtime is in an older version than kubelet**, the feature will be gracefully degraded. In this case the user will be able to set the stop signal in the Container spec, but the kubelet will not pass this value to the container runtime via ContainerConfig and the container runtime will use the stop signal defined in the image or use the default SIGTERM.

- **If you're running an older version of the kubelet with a newer version of the container runtime**, the CRI API call from the kubelet would be made with the older version of ContainerConfig which doesn't include the stop signal. The container runtime doesn't receive any custom stop signal from the container spec in this case. The container runtime code, even if it is running the newer version supporting stop signal, would fall back to the current behaviour and use the stop signal defined in the container image or default to SIGTERM since it doesn't receive any stop signal from ContainerSpec.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ContainerStopSignals
  - Components depending on the feature gate: kube-apiserver, kubelet
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

No, enabling the feature gate does not change existing behaviour.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the feature gate can be turned off to disable the feature once it has been enabled.

The feature gate is present in kube-apiserver and kubelet. Both enabling and disabling the feature gate once it has been turned on would involve restarting the kube-apiserver and the kubelet. The update strategy of how to roll out the feature gate flips for both kube-apiserver and kubelet are described in the [Version Skew Strategy](#version-skew-strategy) section.

###### What happens if we reenable the feature if it was previously rolled back?

If you reenable the feature, you'll be able to create Pods with StopSignal lifecycle hooks for their containers. Once the gate is disabled, if you try to create new Pods with StopSignal, those would be invalid and wouldn't pass validation. Existing worklods using StopSignal should still continue to function.

If the feature gate is turned off in the kubelet alone, users would be able to create Pods with StopSignal, but the field will be dropped in the kubelet. When they turn the feature gate back on in the kubelet, the kubelet would start sending the StopSignal to the container runtime again, and the container runtime starts using the custom stop signal to kill the containers. If cluster operator disables or reenables the feature gate, no change should happen to exisiting workloads.

When the kubelet restarts after the feature gate is reenabled, it polls the CRI implementation for existing containers and then compares them to the ones that the API is requesting. It is recommended to drain the node before enabling/disabling the feature gate in the kubelet.

###### Are there any tests for feature enablement/disablement?

Yes, unit tests are planned for alpha for testing the disabling and reenabling of the feature gate.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The change is opt-in, it doesn't impact already running workloads. The only change to the existing workloads would be the stop signal showing up in the container statuses for existing Pods once the change is rolled out in the kubelet.

###### What specific metrics should inform a rollback?

Pods/Containers not getting terminated properly might indicate that something is wrong, although we will aim to handle all such cases gracefully and show proper error messages if something is missing. 

You can also look at the newly proposed metric `kubelet_pod_termination_grace_period_exceeded_total` which gives you the number of Pods which are killed forcefully after the timeout for graceful termination exceeded. A high value for this metric could mean that Pods are not getting killed gracefully. This could mean that Pods might have a misconfigured stop signals and might need a rollback.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This is an opt-in feature, and it does not change any default behavior. I will manually tested enabling and disabling this feature by changing the configs for kube-apiserver and kubelet and restarting them in a kind cluster. The details of the expected behavior are described in the Proposal and Upgrade/Downgrade sections.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Inspect the workloads' Container spec for Stop lifecycle hook and also check if the ContainerStopSignal feature gate is enabled.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [x] Other (treat as last resort)
  - Check if the containers with custom stop signals are being killed with the stop signal provided. For example your container might want to take SIGUSR1 to be exited. You can achieve this by defining it in the Container spec and have to bake it into your container image.
  - Since we're showing the effective stop signal in the container status, irrespective of whether a custom signal is used or not, users can check whether Pods scheduled to every node has a StopSignal field in their statuses to confirm whether the feature is enabled and working in that particular instance of kubelet.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name:
    - kubelet_pod_stop_signals_count (Gauge measuring number of  pods configured with each stop signal)
    - kubelet_pod_termination_grace_period_exceeded_total (Counter counting the number of Pods that doesn't get terminated gracefully with the duration of terminationGracePeriodSeconds)
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [x] Other (treat as last resort)
  - Details:  Check if the containers with custom stop signals are being killed with the stop signal provided.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Metrics related to pods termination could be useful to improve the observability of stop signal usage. There aren't any metrics related to the terminationGracePeriodSeconds for Pods and one such metric has been proposed above, `kubelet_pod_termination_grace_period_exceeded_total`, which counts the number of Pods that gets terminated forcefully after exceeding terminationGracePeriodSeconds.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No, but the CRI API update would require us to update the logic in the container runtimes as well for the feature to work.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

We are adding a new lifecycle hook called StopSignal, which takes a string value. These are optional values however and can increase the size of the API object.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The same way any write to kube-apiserver/etcd would behave. This feature doesn't change this behaviour.  

###### What are other known failure modes?

Pods can fail and hang if the user configures a stop signal that is not handled in the container. This is a new failure mode that is introduced by this KEP. The KEPs would hang until they're forcecully killed with SIGKILL after the terminationGracePeriodSeconds.

###### What steps should be taken if SLOs are not being met to determine the problem?

Disable the ContainerStopSignal feature gate, and restart the kube-apiserver and kubelet.

## Implementation History

- 2025-02-13: Alpha [KEP PR](https://github.com/kubernetes/enhancements/pull/5122) approved and merged for v1.33
- 2025-03-25: Alpha [code changes to k/k](https://github.com/kubernetes/kubernetes/pull/130556) merged with API changes, validation and CRI API implementation
- 2025-04-04: [CRI-O implementation PR](https://github.com/cri-o/cri-o/pull/9086) merged

## Drawbacks

One of the drawbacks of introducing stop signal to the container spec is that this introduces the scope of users misconfiguring the stop signal leading to unexpected behaviour such as the hanging pods as mentioned in the [Risks and Mitigations](#risks-and-mitigations) section.

## Alternatives

As discussed above, one alternative would be to bake the stop signal into the container image definition itself. This is tricky when you're using pre-built image or when you cannot or do not want to build custom images just to update the stop signal.

Another alternative is to define the stop signal as a PreStop lifecycle hook like so as mentioned in user story #1.

```yaml
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "kill -SIGUSR1 $(pidof my-app)"]
```

## Infrastructure Needed (Optional)

N/A
