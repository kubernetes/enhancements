# KEP-5532: Restart Pod on Container Exits

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Rerun with Init Containers](#story-1-rerun-with-init-containers)
    - [Story 2: Simplified Application and Sidecar Logic](#story-2-simplified-application-and-sidecar-logic)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Unintended Pod Restart Loops](#unintended-pod-restart-loops)
- [Design Details](#design-details)
  - [API](#api)
  - [Kubelet Implementation](#kubelet-implementation)
    - [Handling Kubelet Restarts](#handling-kubelet-restarts)
  - [PodCondition PodRestarting](#podcondition-podrestarting)
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
  - [Wrapping entrypoint](#wrapping-entrypoint)
  - [Non-declarative (callbacks based) restart policy](#non-declarative-callbacks-based-restart-policy)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

This KEP proposes an extension to the container restart rules introduced in [KEP-5307](https://github.com/kubernetes/enhancements/issues/5307) to allow a container's exit to trigger a restart of the entire pod. This "in-place" pod restart will terminate and then restart all of the pod's containers (including init and sidecar containers) while preserving the pod's sandbox, UID, network namespace, and IP address. This provides a more efficient way to reset a pod's state compared to deleting and recreating the pod, which is particularly beneficial for workloads like AI/ML training, where rescheduling is costly.

## Motivation

While KEP-5307 introduces container-level restart policies, there are scenarios where restarting the entire pod is more desirable than restarting a single container.

1.  **Rerunning Init Containers:** Many applications rely on init containers to prepare the environment, such as mounting volumes with gcsfuse or performing other setup tasks. When a container fails, a full pod restart ensures that these init containers are re-executed, guaranteeing a clean and correctly configured environment for the new set of application containers.

2.  **Simplified Application and Sidecar Logic:** In complex pods, such as ML training workloads with sidecars monitoring for failures, a `RestartPod` action simplifies lifecycle management. Instead of implementing complex inter-container communication to signal a pod-level failure, a sidecar can simply exit with a specific code to trigger a full pod restart, resetting the main application from its last checkpoint.

3.  **Efficient In-Place Restart:** Deleting and recreating a pod is a heavy operation involving the scheduler, node resource allocation, and re-initialization of networking and storage. An in-place restart, which preserves the pod sandbox and its associated resources (UID, IP, devices), is significantly faster and reduces resource churn.

4.  **Improved Predictability and Debugging:** Restarting all containers together brings the entire pod to a known good state. This is often easier to reason about and debug than a state where some containers are running while others have been restarted independently.

### Goals

- Introduce a `RestartPod` action to the `ContainerRestartRule` API.
- Implement the kubelet logic to perform an in-place pod restart, which includes:
    - Terminating all containers in the pod gracefully.
    - Re-running init containers.
    - Restarting all regular and sidecar containers.
    - Preserving the pod sandbox, UID, and network identity.
- Introduce a new PodCondition to make the pod restart process observable.

### Non-Goals

- Introducing triggers for pod restart other than container exits (e.g., via a direct API call). This could be a future enhancement.
- Tearing down and recreating the pod sandbox during the restart. The focus is on an efficient "in-place" restart.

## Proposal

This proposal extends the API defined in KEP-5307 by adding a new action, `RestartPod`, to `ContainerRestartRuleAction`. When a container exits, the kubelet will evaluate the `restartPolicyRules`. If a rule with the `RestartPod` action matches the exit condition (e.g., a specific exit code), the kubelet will initiate an in-place restart of the pod.

### User Stories (Optional)

#### Story 1: Rerun with Init Containers

As a developer, I have a pod where an init container is responsible for setting up a resource, like mounting a volume or preparing a configuration file, that the main container depends on. If the main application container fails in a way that corrupts this resource's state, I want the entire pod to restart. This ensures the init container runs again to provide a clean setup before the application container starts. I can configure the main container to exit with a specific code that triggers the `RestartPod` action.

#### Story 2: Simplified Application and Sidecar Logic

As an ML engineer, I run distributed training jobs where a sidecar container monitors the main training container. If the training process encounters a specific, retriable error, the sidecar detects it and needs to restart the whole worker pod from the last checkpoint. With this feature, I can program the sidecar to simply exit with a specific code. This triggers the `RestartPod` action, which efficiently resets the worker without needing complex communication between the sidecar and the main container or involving the Job controller for a full pod recreation.

### Risks and Mitigations

#### Unintended Pod Restart Loops

A container might persistently exit with an exit code that triggers a `RestartPod` action, causing the entire pod to enter a restart loop. This could consume significant node resources and mask the underlying problem.

**Mitigation:** The kubelet already implements an exponential backoff for container restarts. This same backoff mechanism will be applied to pod restarts triggered by this feature. This will introduce increasing delays between restart attempts, preventing rapid, resource-intensive restart loops and giving operators time to diagnose the issue.

## Design Details

### API

The proposal is to extend the `ContainerRestartRuleAction` enum with `RestartPod`.

```go
type ContainerRestartRuleAction string

const (
  // Restarts the container that exited.
  ContainerRestartRuleActionRestart ContainerRestartRuleAction = "Restart"
  // Restarts the entire pod.
  ContainerRestartRuleActionRestartPod ContainerRestartRuleAction = "RestartPod"
)
```

Example usage in a Pod manifest:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-ml-worker
spec:
  restartPolicy: Never
  initContainers:
  - name: watcher-sidecar
    image: watcher
    restartPolicy: Always
  containers:
  - name: main-container
    image: training-app
    restartPolicy: Never
    restartPolicyRules:
    - action: RestartPod
      onExit:
        exitCodes:
          operator: In
          values: [88] # A specific exit code indicating a retriable error.
```

### Kubelet Implementation

The in-place pod restart will be implemented in the kubelet as a state machine with a timestamp to robustly manage the lifecycle transition. When a `RestartPod` rule is triggered, the kubelet will cycle the pod through the following phases:

1.  **Termination Phase:** When a `RestartPod` rule is triggered, the kubelet will transition its internal, in-memory state for the pod to `TerminatingForRestart`. In this state, the kubelet's only goal is to terminate all of the pod's containers. This process is similar to a normal pod shutdown but skips tearing down the sandbox.

    -   The kubelet updates the pod's status on the API server with the `PodRestarting=True` condition for observability.
    -   Probes are stopped.
    -   `preStop` hooks are executed.
    -   Containers are sent a `TERM` signal for graceful shutdown.
    -   Any container exits that occur while in this state are considered part of the planned shutdown.
<<[UNRESOLVED whether volumes mounts need to be cleaned up]>>
    -   Volumes are unmounted.
<<[/UNRESOLVED]>>

2.  **Startup Phase:** Once the kubelet verifies that all containers have terminated, it transitions its internal state to `Startup` with a timestamp. In this state, the kubelet's goal is to start the pod from the beginning, preserving the existing sandbox. The container statuses from the previous run are preserved for history.

    -   The container statuses from the previous run are preserved for history.
<<[UNRESOLVED whether volumes mounts need to be cleaned up]>>
    -   Volumes are mounted.
<<[/UNRESOLVED]>>
    -   Init containers are executed in order.
    -   Regular and sidecar containers are started.
    -   `postStart` hooks are executed.
    -   Probes are re-activated.

If any container is terminated while the pod is in the `Startup` phase, the kubelet evaluates the failure timestamp against the state machine’s timestamp. If the container termination timestamp is before the state machine’s timestamp, then it means the container has not been restarted. Otherwise, it means the container has already been restarted once, and it is a new, genuine failure. It will then handle this failure according to the pod's `restartPolicy` (e.g., for a policy of `Never`, the pod will be marked as `Failed`).

Once all containers are running successfully, the kubelet transitions its internal state back to `Running` and updates the pod's status to set the `PodRestarting` condition to `False`.

During this process, the pod's UID, IP address, network namespace, devices, and CGroups will be preserved. All regular containers will be restarted, regardless of their individual `restartPolicy` or previous exit status. Ephemeral containers will not be restarted.

#### Handling Kubelet Restarts

The in-memory state machine is lost if the kubelet restarts. To ensure the pod restart process is resilient, the `PodRestarting` condition in the `Pod.status` serves as the persistent record of the state.

When the kubelet starts, it syncs all pods assigned to it. If it finds a pod with the `PodRestarting` condition set to `True`, it reconstructs its internal state machine by inspecting the pod's `containerStatuses`:
- If any containers are still in a running state, the kubelet deduces it was in the **Termination Phase**. It will resume this phase by terminating the remaining running containers.
- If all containers are in a terminated state, the kubelet deduces it was in the **Startup Phase**. It will begin or resume this phase by running init containers and starting the main containers.

If the pod entered Startup Phase, and kubelet got restarted, the kubelet might think the pod is in Termination Phase. This could cause a second restart of the pod. However, since the pod is already marked for restart, a repeated restart is not a significant threat.

This mechanism ensures that the in-place restart operation can continue correctly even if the kubelet is restarted mid-process.

### PodCondition PodRestarting

To make the restart process observable, a new `PodCondition` will be added to the `Pod.status.conditions`.

```
type: PodRestarting
status: True / False
reason: ContainerExited
message: 'Container my-container exited with code 88, triggering pod restart'
```

The kubelet will set this condition to `True` at the beginning of the termination phase and set it to `False` once the startup phase is complete and the containers are running.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

N/A

##### Unit tests

- `k8s.io/apis/core`
- `k8s.io/apis/core/v1/validations`
- `k8s.io/features`
- `k8s.io/kubelet`
- `k8s.io/kubelet/container`

##### Integration tests

Unit and E2E tests are expected to provide sufficient coverage.

##### e2e tests

-   Create a pod with a container that has a `restartPolicyRule` with the `RestartPod` action. Verify that when the container exits with the specified code, the entire pod is restarted in-place (same UID, IP).
-   Verify that init containers are re-executed after a pod restart is triggered.
-   Verify that all regular and sidecar containers are restarted.
-   Verify that the `PodRestarting` condition is added to the pod status during the restart and removed after it completes.

### Graduation Criteria

#### Alpha

-   Feature implemented behind a `RestartPodOnContainerExits` feature gate.
-   The `RestartPod` action is added to the API.
-   Kubelet implementation of the in-place pod restart logic is complete.
-   Initial e2e tests are completed and enabled to verify the core functionality.
-   Documentation is added.

#### Beta

- Container restart policy functionality running behind feature flag
for at least one release.
- Container restart policy runs well with Job controller.

#### GA

- No major bugs reported for three months.
- User feedback (ideally from at least two distinct users) is green.

### Upgrade / Downgrade Strategy

The feature gate `RestartPodOnContainerExits` will protect the new functionality.

-   **Upgrade:** When upgrading, the API server should be upgraded before the kubelets. If a pod with the `RestartPod` rule is scheduled on an older kubelet that doesn't support the feature, the rule will be ignored, and the pod's `restartPolicy` will be used.
-   **Downgrade:** If the feature is disabled or kubelets are downgraded, any `RestartPod` rules in existing pods will be ignored. The pod will revert to the behavior defined by its `restartPolicy`.

### Version Skew Strategy

Previous kubelet client unaware of the RestartPod action will ignore
this field and keep the existing behavior determined by pod's restart policy.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `RestartPodOnContainerExits`
  - Components depending on the feature gate: kube-apiserver, kubelet

###### Does enabling the feature change any default behavior?

No. The feature is opt-in. It only takes effect when the `RestartPod` action is explicitly used in a container's `restartPolicyRules`. Existing workloads are unaffected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate `RestartPodOnContainerExits` on the API server and kubelets will cause the `RestartPod` action to be ignored. Pods will fall back to the behavior defined by their `restartPolicy`.

###### What happens if we reenable the feature if it was previously rolled back?

If the feature is re-enabled, kubelets will once again recognize and enforce the `RestartPod` rules for any pods that have them defined.

###### Are there any tests for feature enablement/disablement?

- Unit test for the API's validation with the feature enabled and disabled.
- Unit test for the kubelet with the feature enabled and disabled.
- Unit test for API on the new field for the Pod API. First enable
the feature gate, create a Pod with a container including RestartPod action,
validation should pass and the Pod API should match the expected result.
Second, disable the feature gate, validate the Pod API should still pass
and it should match the expected result. Lastly, re-enable the feature
gate, validate the Pod API should pass and it should match the expected result.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

During a rollout, a cluster may have a mix of kubelets with the feature enabled and disabled. If a pod using the `RestartPod` feature is scheduled on a node where the feature is not yet enabled, it will not have the desired restart behavior. This could lead to inconsistent behavior for a given workload during the rollout period, but it will not cause running workloads to fail.

###### What specific metrics should inform a rollback?

Repeated restarts of containers or pods, especially if they are not progressing.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be tested manually before graduation to Beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
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
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

A new possible value "RestartPod" for RestartRulesAction will be introduced.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

The size of the PodCondition API object will be increased for account for the new PodRestarting status, example:
```
type: PodRestarting
status: True / False
reason: ContainerExited
message: 'Container my-container exited with code 88, triggering pod restart'
```

- API type: PodCondition
- Estimated increase in size: 200B
- Estimated amount of new objects: at most one per pod.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
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
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### Wrapping entrypoint

One way to implement this KEP as a DIY solution is to wrap the entrypoint
of the container with the program that will implement this exit code handling
policy. This solution does not scale well as it needs to be working on multiple
Operating Systems across many images. So it is hard to implement universally.

### Non-declarative (callbacks based) restart policy

An alternative to the declarative failure policy is an approach that allows
containers to dynamically decide their faith. For example, a callback is called
on an “orchestration container” in a Pod when any other container has failed.
And the “orchestration container” may decide the fate of this container - restart
or keep as failed.

This may be a possibility long term, but even then, both approaches can work
in conjunction.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
