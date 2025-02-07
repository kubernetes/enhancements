
# KEP-4960: Container Stop Signals 

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API](#api)
  - [CRI API](#cri-api)
  - [Container runtime changes](#container-runtime-changes)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

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

### CRI API

The CRI API would be updated so the stop signal in the container spec (if it is not nil or unset) is sent to the container runtime via ContainerConfig. This would be passed down to the container runtime's StopContainer method ultimately:

```diff
// ContainerConfig holds all the required and optional fields for creating a
// container.
message ContainerConfig {
  // ...
+ Lifecycle lifecycle = 18;
}

+message Lifecycle {
+  string stop_signal = 1;
+}
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
+   Lifecycle:  &runtimeapi.Lifecycle{
+     StopSignal: container.Lifecycle.StopSignal 
+   },
	}
  // ...
}
```

Since the new stop lifecycle is optional, the default stop signal for a container can be unset or nil. In this case, the container runtime will fallback to the existing behaviour. 

Additionally, the stop signal would also be added to `ContainerStatus` (as `containerStatus[].Lifecycle.StopSignal`) so that we can pass the stop signal extracted from the image/container runtime back to the container status at the Kubernetes API level.

### Container runtime changes

Once the stop signal from `containerSpec.Lifecycle.StopSignal` is passed down to the container runtime via `ContainerConfig` during creation/updation of the container, we can use the value of the stop signal from the container runtime's implementation of `stopContainer` method. In the case of containerd, it would look like this:

```diff
//internal/cri/server/container_stop.go

func (c *criService) StopContainer(ctx context.Context, r *runtime.StopContainerRequest) (*runtime.StopContainerResponse, error) {
// ...
-	if err := c.stopContainer(ctx, container, time.Duration(r.GetTimeout())*time.Second); err != nil {
+ 	if err := c.stopContainer(ctx, container, time.Duration(r.GetTimeout())*time.Second, container.Config.Lifecycle.StopSignal); err != nil {
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

### User Stories (Optional)

#### Story 1

Kubernetes by default sends a SIGTERM to all containers while killing them. When running nginx on Kubernetes, this can result in nginx dropping requests as reported [here](https://github.com/Kong/kubernetes-ingress-controller/pull/283). The current solution for this issue would be to build custom images with a SIGQUIT stop signal or to write a PreStop lifecycle hook that manually kills the process gracefully, which is what is done in the PR. If we had stop signal support at the Container spec level, this would've been easier and straightforward to implement. Users wouldn't have to patch the applications running on Kubernetes to handle different termination behavior. This is also similar to [this issue](https://github.com/github/resque/pull/21). 

### Risks and Mitigations

- We'll be adding the complexity of signal handling to the pod/container spec. If users define an signal that is not handled by their containers, this can lead to pods hanging. 

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
  
##### e2e tests

- Test that containers are killed with the right stop signal when a stop signal is passed
- Test that containers are killed with SIGTERM when no stop signal is passed
- Test that the Status returns the correct stop signal in all the following cases:
   - When stop signal is defined in the Container Spec (Status should have signal is defined in the Spec)
   - When stop signal is only defined in the container image (Status should have the signal defined in the image)
   - When no stop signal is defined (Stop signal in Status should be SIGTERM)
- Test that the stop signal is gracefully degraded when stop signal is specified but the container runtime is on a version that doesn't support the implementation
- Test that the feature is gracefully degraded when stop signal is not supported in Kubelet but is supported in the container runtime

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- CRI API implementation completed in containerd marked as experimental
- CRI API implementation completed for CRI-O
- Initial e2e tests completed and enabled, testing the feature against containerd
- Unit tests for validation, e2e tests for version skew

#### Beta

- Add support for Windows
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

If only the kube-apiserver enables this feature, validation will pass, but kubelet won't understand the new lifecycle hook and will not add the stop signal when creating the ContainerConfig.

If only the kubelet has enabled this feature, you won't be able to create a Pod which has a StopSignal lifecycle hook via the apiserver and hence the feature won't be usable even if the kubelet supports it. `containerSpec.Lifecycle.StopSignal` can be an empty value and kubelet functions as if no custom stop signal has been set for any container.

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

###### What happens if we reenable the feature if it was previously rolled back?

If you reenable the feature, you'll be able to create Pods with StopSignal lifecycle hooks for their containers. Without the feature gate enabled, this would make your workloads invalid.

###### Are there any tests for feature enablement/disablement?

Yes, unit tests are planned for alpha for testing the disabling and reenabling of the feature gate.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The change is opt-in, it doesn't impact already running workloads.

###### What specific metrics should inform a rollback?

Pods/Containers not getting terminated properly might indicate that something is wrong, although we will aim to handle all such cases gracefully and show proper error messages if something is missing.

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
  - Details: Check if the containers with custom stop signals are being killed with the stop signal provided. For example your container might want to take SIGUSR1 to be exited. You can achieve this by defining it in the Container spec and have to bake it into your container image.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [x] Other (treat as last resort)
  - Details:  Check if the containers with custom stop signals are being killed with the stop signal provided.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

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

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

Disable the ContainerStopSignal feature gate, and restart the kube-apiserver and kubelet.

## Implementation History

## Drawbacks

There aren't any drawbacks to why this KEP shouldn't be implemented since it does not change the default behaviour.

## Alternatives

As discussed above, one alternative would be to bake the stop signal into the container image definition itself. This is tricky when you're using pre-built image or when you cannot or do not want to build custom images just to update the stop signal.

## Infrastructure Needed (Optional)

N/A
