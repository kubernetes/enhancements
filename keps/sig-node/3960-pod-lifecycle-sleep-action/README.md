# KEP-3960: Introducing Sleep Action for PreStop Hook

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Implementation](#implementation)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
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

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes the addition of a new sleep action for the PreStop lifecycle hook in Kubernetes, allowing containers to pause for a specified duration before termination. This enhancement aims to provide a more straightforward way to manage graceful shutdowns and improve the overall lifecycle management of containers, and to handle new connections from clients that have not yet finished endpoint termination during the pod termination.

An example:

To use this feature to achieve zero downtime for nginx, we need to deploy a deployment with sleep prestop hook and a service.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  selector:
    matchLabels:
      app: nginx
  replicas: 3
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.16.1
        lifecycle:
          preStop:
            sleep:
              seconds: 5
        readinessProbe:
          httpGet:
            path: /
            port: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
spec:
  selector:
    app: nginx
  ports:
  - port: 80
    targetPort: 80
```

- Restart/Update this deployment
- A delete pod event is sent to notify the kubelet and the Endpoint Controller (which manages the Service endpoints) simultaneously.
- PreStop hook starts, which will delay the shutdown sequence by 5 seconds. During this time, the Endpoint Controller will remove the terminating pod, and the traffic will be sent to other running pods.
- When the timeout of 5 seconds ended, the old pod is killed, and new pods will be created. There is no traffic sent to the terminating pod during the update.

## Motivation

Currently, Kubernetes supports two types of actions for PreStop hooks: exec and httpGet. Although these actions offer flexibility, they often require additional scripting or custom solutions to achieve a simple sleep functionality. A built-in sleep action would provide a more user-friendly and native solution for scenarios where a container needs to pause before shutting down, such as:

- Ensuring that the container gracefully releases resources and connections.
- Allowing a smooth transition in load balancers or service meshes.
- Providing a buffer period for external monitoring and alerting systems.

### Goals

- Allow containers to perform cleanup or shutdown actions before being terminated, by sleeping for a specified duration in the preStop hook.
- Improve the overall reliability and availability of Kubernetes applications by providing a way for containers to gracefully terminate.

### Non-Goals

- This KEP does not aim to replace other Kubernetes features that can be used to perform cleanup actions, such as init containers or sidecar containers.
- This KEP does not aim to provide a way to pause or delay pod termination indefinitely.

## Proposal

We propose adding a new sleep action for the PreStop hook, which will pause the container for a specified duration before termination. The API changes will include the following:

- Extending the LifecycleHandler object to support a new Sleep field.
- Adding a SleepAction object that includes a Duration field to specify the sleep period in seconds.

### User Stories (Optional)

#### Story 1
As a Kubernetes user, I want to configure my container to sleep for a specific duration during graceful termination and I want to do it without needing a sleep binary in my image.

#### Story 2
As a Kubernetes user, I want to configure my nginx service to be able to run with zero downtime.Previously,I use `command: ["/bin/sh","-c","sleep 20"]` in prestop hook with exec command to delay the shutdown,and let the Endpoint Controller remove the pod first. But this requires me to have a sleep binary in my image, and I want to do this more conveniently.

### Risks and Mitigations

N/A

## Design Details

### Implementation

- Adding a SleepAction object that includes a Duration field to specify the sleep period in seconds.
```go
type SleepAction struct {
	// Seconds is the number of seconds to sleep. 
	Seconds int32
}
```

-  Adding a Sleep field to the LifecycleHandler struct, which represents the duration in seconds that the container should sleep before being terminated during the preStop hook.
```go
type LifecycleHandler struct {
	// Sleep represents the duration in seconds that the container should sleep before being terminated. If the container terminates before the sleep finishes, this action will be interrupted.
	Sleep *SleepAction
}
```

- When Kubernetes executes the preStop hook with sleep action, it'll simply sleep for a specific seconds.
```go
func (hr *handlerRunner) Run(ctx context.Context, containerID kubecontainer.ContainerID, pod *v1.Pod, container *v1.Container, handler *v1.LifecycleHandler) (string, error) {
    switch {
    case handler.Exec != nil:...
    case handler.HTTPGet != nil:...
    case handler.Sleep != nil:
        hr.runSleepHandler(ctx, handler.Sleep.Seconds)
        return "", nil
    default:...
    }
}

func (hr *handlerRunner) runSleepHandler(ctx context.Context, seconds int32) {
    c := time.After(time.Duration(seconds) * time.Second)
    select {
    case <-ctx.Done():
        // early termination
        // some logs
        return
    case <-c:
        // sleep expired
        // some logs
        return
    }
}
```
### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- Test that the runSleepHandler function sleeps for the correct duration when given a valid duration value.
- Test that the runSleepHandler function returns without error when given a valid duration value.
- Test that the validation returns an error when given an invalid duration value (e.g., a negative value).
- Test that the validation returns an error when given duration is longer than the termination graceperiod.
- Test that the runSleepHandler function returns immediately when given a duration of zero.

##### Integration tests
N/A

##### e2e tests
- Basic functionality
  1. Create a simple pod with a container that runs a long-running process.
  2. Add a preStop hook to the container configuration, using the new sleepAction with a specified sleep duration (e.g., 5 seconds).
  3. Delete the pod and observe the time it takes for the container to terminate.
  4. Verify that the container sleeps for the specified duration before it is terminated.
  5. Verify that the container keeps executing code while sleeping.

- Sleep duration boundary testing
  1. Create a simple pod with a container that runs a long-running process.
  2. Add a preStop hook to the container configuration, using the new sleepAction with various sleep durations, including:1 seconds (minimum allowed value), values slightly above the minimum allowed value (to test edge cases).
  3. For each sleep duration, delete the pod and observe the time it takes for the container to terminate.
  4. Verify that the container sleeps for the specified duration before it is terminated.
  5. Verify that the container keeps executing code while sleeping.

- Container exit/crash testing
  1. Create a simple pod with a container that will exit soon.
  2. Add a preStop hook to the container configuration, using the new sleepAction with a specified sleep duration longer than the container's lifecycle.
  3. Exit the container and observe the time it takes for the pod to terminate.
  4. Verify that the pod be deleted successfully without waiting for the entire sleep duration.

- Interaction with termination grace period
  1. Create a simple pod with a container that runs a long-running process.
  2. Add a preStop hook to the container configuration, using the new sleepAction with a specified sleep duration (e.g., 5 seconds).
  3. Set the termination grace period to various values, including:
     - Equal to the sleep duration
     - Greater than the sleep duration
     - Greater than the sleep duration, but reduced to less than the sleep duration at runtime
  4. For each termination grace period value, delete the pod and observe the time it takes for the container to terminate.
  5. Verify that the container is terminated after the min(sleep, grace).

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial unit/e2e tests completed and enabled
- Documentation is added to demonstrate why this is useful in nginx scenario and how exactly nginx needs to be configured.

#### Beta

- Gather feedback from developers and surveys
- Additional e2e tests are completed(if needed)

#### GA

- No negative feedback
- No bug issues reported

### Upgrade / Downgrade Strategy

#### Upgrade
The previous PreStop behavior will not be broken. Users can continue to use their hooks as it is.
To use this enhancement, users need to enable the feature gate, and add sleep action in their prestop hook.

#### Downgrade
It will silently drop the sleep prestop hook if someone wants to add that to the object (or create object with it). However, existing objects that have it set will not be cleared.

### Version Skew Strategy

Both kubelet and kube-apiserver will need enable the feature gate for the full featureset
to be present. If both components disable the feature gate, this feature will be cleanly unavailable.

If only the kube-apiserver enable this feature, kubelet won't understand the new field and may either ignore it or raise an error.

If only the kubelet enable this feature, when creating/updating a resource with the sleepAction, the API server will reject the request with an error indicating that the field is unknown or unsupported.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: PodLifecycleSleepAction
  - Components depending on the feature gate: kubelet,kube-apiserver
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

The feature can be disabled in Alpha and Beta versions by restarting kube-apiserver with the feature-gate off. In terms of Stable versions, users can choose to opt-out by not setting the sleep field.

In this case, the created pods's sleepAction will take effect, and the new pod with sleepAction is not allowed to create.

###### What happens if we reenable the feature if it was previously rolled back?

New pods with sleep action in prestop hook can be created.
Previously created pod with sleep hook set will execute it before terminating.

###### Are there any tests for feature enablement/disablement?

Yes. Some unit tests will be designed to test the verification process of the "sleep" field under different scenarios, such as when the feature is enabled, disabled, or switched. These tests will be included in the alpha version.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The change is opt-in, it doesn't impact already running workloads. 

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Inspect the prestop hook configuration

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [x] Other (treat as last resort)
  - Details: Check the logs of the container during termination, check the termination duration.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [x] Other (treat as last resort)
  - Details: Check the logs of the container during termination, check the termination duration.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

N/A

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

LifecycleHandler objects have one new fields per version they define, increasing their size slightly.
See `Implementation` part

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?
- In general, if the API server and/or etcd is unavailable, Kubernetes will be unable to coordinate container termination and the preStop hook may not be executed at all. This could result in the container being terminated abruptly without the opportunity to perform any necessary cleanup actions.

- If the sleep action is enabled for the preStop hook, it will still attempt to sleep for the specified duration before the container is terminated. However, if the API server and/or etcd is unavailable, Kubernetes may be unable to send the SIGTERM signal to the container, which could cause the container to continue running beyond the specified sleep period.

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

- 2023-04-22: Initial draft KEP

## Drawbacks

N/A

## Alternatives

Another way to run `sleep` in a container is to use `exec` command in `preStop hook` like `command: ["/bin/sh","-c","sleep 20"]`. However this requires a sleep binariy in the image. We should offer sleep as a first-class thing.
