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
  - [User Stories](#user-stories)
    - [Story 1: Rerun with init containers](#story-1-rerun-with-init-containers)
    - [Story 2: Efficient in-place restart](#story-2-efficient-in-place-restart)
    - [Story 3: RestartPod with init container providing items from a queue](#story-3-restartpod-with-init-container-providing-items-from-a-queue)
    - [Story 4: Restart main container on sidecar failures](#story-4-restart-main-container-on-sidecar-failures)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Unintended Pod Restart Loops](#unintended-pod-restart-loops)
- [Design Details](#design-details)
  - [API](#api)
  - [Restart Phases](#restart-phases)
    - [Best-effort Pre-stop hooks](#best-effort-pre-stop-hooks)
    - [Termination Grace Periods](#termination-grace-periods)
    - [Containers in Runtime](#containers-in-runtime)
    - [ContainerStatuses in API](#containerstatuses-in-api)
    - [Sandbox](#sandbox)
    - [Volumes](#volumes)
    - [Init Containers](#init-containers)
    - [Regular Containers](#regular-containers)
    - [Ephemeral Containers](#ephemeral-containers)
    - [Probes](#probes)
  - [Pod Status](#pod-status)
    - [[New] Pod condition PodRestartInPlace](#new-pod-condition-podrestartinplace)
    - [Existing Pod Conditions](#existing-pod-conditions)
    - [Pod Phase](#pod-phase)
  - [Kubelet Implementation](#kubelet-implementation)
    - [Kubelet Restarts](#kubelet-restarts)
    - [Node Restarts](#node-restarts)
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
  - [Giving access to CRI API or subset](#giving-access-to-cri-api-or-subset)
  - [Pod Self-orchestration](#pod-self-orchestration)
  - [Livesness probes on regular containers that point to a sidecar container](#livesness-probes-on-regular-containers-that-point-to-a-sidecar-container)
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

This KEP proposes an extension to the container restart rules introduced in [KEP-5307](https://github.com/kubernetes/enhancements/issues/5307) to allow a container's exit to trigger a restart of the entire pod. This is part of the Pod / Container Restart roadmap planned earlier (see [discussion](https://docs.google.com/document/d/1UmJHJzdmMA1hWwkoP1f3rG9nS0oZ2cRcOx8rO8MsExA/edit?resourcekey=0-OuKspBji_1KJlj2JbnZkgQ&tab=t.0#heading=h.e2gx4umk9f64)). This "in-place" pod restart will terminate and then restart all of the pod's containers (including init and sidecar containers) while preserving the pod's sandbox, UID, network namespace, attached devices, and IP address. This provides a more efficient way to reset a pod's state compared to deleting and recreating the pod, which is particularly beneficial for workloads like AI/ML training, where rescheduling is costly.

## Motivation

While KEP-5307 introduces container-level restart policies, there are scenarios where restarting the entire pod is more desirable than restarting a single container. The benefits of restarting the whole pod in-place includes the following.

**Re-run with Init Containers**

Many applications rely on init containers to prepare the environment, such as mounting volumes with gcsfuse or performing other setup tasks. When a container fails, a full pod restart ensures that these init containers are re-executed, guaranteeing a clean and correctly configured environment for the new set of application containers. 

Another scenario is when Init container "takes the next item from the queue". And main container exists with the indication that it want's a "new item' to process. See also

- https://github.com/kubernetes/enhancements/issues/3676
- https://github.com/kubernetes/enhancements/issues/3759#issuecomment-2389506153

**Handles Init Container Failures**

Sidecar container failures sometimes render the main container not ready, and restarting the sidecar is insufficient. For example, if a sidecar that manages the remote volume fails and restarts, the main container may be trying to access an outdated volume. With RestartPod action, the sidecar could force the main container to restart as well for a clean environment.

**Efficient In-Place Restart**

Deleting and recreating a pod is a heavy operation involving the scheduler, node resource allocation, and re-initialization of networking and storage. An in-place restart, which preserves the pod sandbox and its associated resources (UID, IP, devices), is significantly faster and reduces resource churn. 

This is especially helpful for ML training workloads, where computation resources are expensive and in-place restarts improves resource usage efficiency. This is also helpful if the workload can run in seconds, and restart in-place is much more efficient than rescheduling. See also

- https://docs.google.com/document/d/16zexVooHKPc80F4dVtUjDYK9DOpkVPRNfSv0zRtfFpk/edit?tab=t.0#heading=h.y6xl7juq7465

**Separates Watcher-sidecars from worker containers**

In ML training workloads we have setups with a watcher process that listens for failures, and restarts the worker from the previous checkpoint if needed. Without a RestartPod action, the watcher process and worker process have to be coupled in a single container, increasing complexity and decreasing cohesion. The RestartPod action eliminates this coupling. See also

- https://docs.google.com/document/d/16zexVooHKPc80F4dVtUjDYK9DOpkVPRNfSv0zRtfFpk/edit?tab=t.0#heading=h.y6xl7juq7465
- https://github.com/kubernetes/enhancements/issues/4438 

**Improved Predictability and Debugging**

Restarting all containers together brings the entire pod to a known good state. This is often easier to reason about and debug than a state where some containers are running while others have been restarted independently.

### Goals

- Introduce a `RestartPod` action to the `ContainerRestartRule` API.
- Implement the kubelet logic to perform an in-place pod restart, which includes:
<<[UNRESOLVED whether containers should be killed or terminated gracefully]>>
    - Terminating all containers (not including graceful termination).
<<[/UNRESOLVED]>>
    - Re-running init containers.
    - Restarting all regular and sidecar containers.
    - Preserving the pod sandbox, UID, and network identity.
- Introduce a new PodCondition to make the pod restart process observable.

### Non-Goals

- Introducing triggers for pod restart other than container exits (e.g., via a direct API call). This could be a future enhancement.
- Tearing down and recreating the all pod resources during the restart. The focus is on an efficient "in-place" restart of the containers and preserve the environment.

## Proposal

This proposal extends the API defined in KEP-5307 by adding a new action, `RestartPod`, to `ContainerRestartRuleAction`. When a container exits, the kubelet will evaluate the `restartPolicyRules`. If a rule with the `RestartPod` action matches the exit condition (e.g., a specific exit code), the kubelet will initiate an in-place restart of the pod.

### User Stories

#### Story 1: Rerun with init containers

As a developer, I have a pod where an init container is responsible for setting up a resource, like mounting a volume or preparing a configuration file, that the main container depends on. If the main application container fails in a way that corrupts this resource's state, I want the entire pod to restart. This ensures the init container runs again to provide a clean setup before the application container starts. I can configure the main container to exit with a specific code that triggers the `RestartPod` action.

#### Story 2: Efficient in-place restart

As an ML engineer, I run distributed training jobs where a sidecar container monitors the main training container. If the training process encounters a specific, retriable error, the sidecar detects it and needs to restart the whole worker pod from the last checkpoint. With this feature, I can program the sidecar to simply exit with a specific code. This triggers the `RestartPod` action, which efficiently resets the worker without involving the Job controller for a full pod recreation or needing complex communication between the sidecar and the main container.

See details in https://docs.google.com/document/d/16zexVooHKPc80F4dVtUjDYK9DOpkVPRNfSv0zRtfFpk/edit?tab=t.0#heading=h.y6xl7juq7465

#### Story 3: RestartPod with init container providing items from a queue

As a developer, I want a pod with an init container and a main container. The init container takes the next item from the queue, and the main container process the item. The main container should be able to exit and indicate that it wants a "new item" to process.

#### Story 4: Restart main container on sidecar failures

As a developer, I have a pod with a sidecar container that provides resources to the main container. If the sidecar fails and restarts, the main container would be trying to access an outdated resource. I want to be able to restasrt all containers if the sidecar fails. This helps to keep my main container up-to-date with the sidecar containers.

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
  - name: setup-envs
    image: setup
  - name: watcher-sidecar
    image: watcher
    restartPolicy: Always
    restartPolicyRules:
    - action: RestartPod
      onExit:
        exitCodes:
          operator: In
          values: [88] # A specific exit code indicating the pod should be restarted.
  containers:
  - name: main-container
    image: training-app
```

The history of the container statuses will be preserved. The restart count of all containers and pod will increment as well. This will be tested in unit test and e2e test, as well as working with the JobSet APIs.

If the pod restart policy is "Never", and the init container fails after the `RestartPod` action requested, the Pod will be marked as Failed.

### Restart Phases
The pod restart can be split into two phases.

The first phase is pod **termination**. The kubelet compares the containerStatuses with restart rules and decides to terminate the pod. The kubelet sets the PodRestartInPlace=True pod condition to the API. The SyncLoop will try to 1) kill all running containers, 2) remove all init and regular containers from the container runtime. The sandbox is preserved to keep the pod IP, UID, devices, and network namespace. The API endpoint slice is also kept. 

Steps to terminate the pod includes:

1) Add pod condition PodRestartInPlace
1) Kill all running containers
    - No ordering during the kill
    - Best-effort: prestop hooks
    - Termination grace periods are not respected.
1) Remove all init and regular containers from container runtime
    - ContainerStatuses are kept in the API
    - Exited containers on the runtime is removed
    - Necessary for a clean restart; otherwise kubelet cannot tell if a container exited before the restart (expected) or after the restart (a new failure).
1) No changes to probes
1) No changes to other pod resources, such as sandbox, IP, network namespcae, devices, volumes, etc.

The second phase is pod **startup**. With all containers terminated and removed, the kubelet unset the PodRestartInPlace pod condition to the API. Because kubelet sees no containers from the container runtime, it can proceed with the **normal Pod startup** actions in the SyncLoop. This will follow the regular pod startup flow, except the sandbox already exists. 

This includes the following steps:

1) The pod resources (sandbox, IP, devices, volumes, etc.) already exists; kubelet will skip recreating those resources.
1) Running init containers in sequence
    - Any new failures will be handled according to restartPolicy, e.g. fail pod if restartPolicy=Never
    - Only proceeds after success
1) Running all sidecar containers in sequence
    - Only proceeds after startupProbe succeeds
1) regular containers
1) poststart hooks
1) Probes became active again

#### Best-effort Pre-stop hooks
The prestop hooks will be respected on a best-effort basis.

#### Termination Grace Periods
The `TerminationGracePeriodSeconds` is not respected. In many cases, best effort cleanups and termination grace periods are desired for real terminations, such as pod being deleted or evicted. However, they might not be expected for quick in-place restarts. Because the container will restart in-place relatively quickly, there shouldn’t be much concern about skipping the cleanup. The termination grace periods will still be respected if the pod is terminating (not restart in-place).

This provides “graceful termination” for real terminations and “fast and nongraceful termination" for in-place restarts.

**Alternative1**: Respect pod.Spec.terminationGracePeriodSeconds

An alternative would be to respect the terminationGracePeriod on the pod level; all containers will be using the same value of pod-level termination grace period. This gives containers the opportunity to perform graceful termination even during restarts. However, this could cause “unexpected cleanup” being performed during the PodRestart; and could slow down the restart process.

This provides “graceful termination” for real terminations as well as “slow and graceful termination” for in-place restarts.

**Alternative2**: Customizable TerminationGracePeriod for RestartPod

Another alternative is to allow users to specify a separate terminationGracePeriod for RestartPod action. With this setup, containers can have appropriate time to cleanup for real terminations, and can have shorter (or even none) periods for in-place restarts. Similar to the probe-level termination grace periods, which overrides the pod-level termination grace period.

This does add extra complexity to the API and implementation. It can be extended in the future if there are feature requests for RestartPod specific termination grace periods.

#### Containers in Runtime
Init containers, sidecar containers, and regular containers are all removed from the runtime to ensure a clean restart of the pod. Ephemeral containers are kept, because they are ephemeral in nature and should not be executed again.

#### ContainerStatuses in API
Container statuses in the API are kept for observability and clarity. However, they will not affect how kubelet restart the pod and containers.

#### Sandbox
Sandbox is preserved. This means pod UID, IP, devices are all preserved. This ensures a faster restart and the pod will get the same resources.

#### Volumes
Volumes are kept. PodRestart focuses on container restart, instead of resetting the environment.

Note: In some cases, remounting the volumes might be desired. This is not in-scope of this KEP. There are ongoing discussions around a separate KEP that focusing on marking volumes as "required for remount" during the container-level restarts or RestartPod actions.

#### Init Containers
Init containers are started in order, including sidecar containers. 
- Requires init containers to be reentrant.
- A failing init container with restartAction=RestartPod can keep the pod restarting (also possible today).

#### Regular Containers
All regular containers will be restarted during a RestartPod action.
- Including succeeded containers with restartPolicy=OnFailure or restartPolicy=Never
- Including all failed containers with restartPolicy=Never
- RestartPod makes more sense to restart all the containers, skipping containers can make reasoning harder.
- In the case of Jobs, it is preferable to restart everything, so the worker can run from scratch again.
- Also possible today if the node got restarted.
- Failed / Succeeded containers can run multiple times if misconfigured. 

#### Ephemeral Containers
Will not be restarted due to their ephemeral nature.

#### Probes
Probes are not deactivated during the restart. All probes are expected to fail during pod restart. The failure of probes should not trigger another pod restart.

**Liveness Probes**
Liveness probes on containers that were running before the restart are expected to fail (because the container is being restarted). The kubelet will coordinate the liveness probe with the SyncPod cycle to ensure that the container is started in order and not affected by liveness probes.

**Readiness Probes**
Readiness probes are expected to fail as well. It is expected that the readiness probe may render the container as not ready.

**Startup Probes**
Startup probes are expected to fail during the restart. After the restart, Startup probes will become active and valid again. The execution of startup probes after the restart will affect the pod lifecycle (e.g. if startup probe failed, the pod will be marked as failed if restartPolicy=Never).

### Pod Status

#### [New] Pod condition PodRestartInPlace

To make the restart process observable, a new pod condition will be added to the `Pod.status.conditions`.

```
type: PodRestartInPlace
status: True / False
reason: ContainerExited
message: 'Container my-container exited with code 88, triggering pod restart'
```

The kubelet will set this condition to `True` at the beginning of the termination phase. The kubelet will set it to `False` at the end of the termination phase (with all containers removed from the runtime).

This condition has the following benefits:
- Restart status is kept across reboots and updates.
- Consistent with 1) API server is single truth; 2) the SyncLoop read Pod from API server and update pod status and perform actions
- Pod lifecycle is reported to the API server and visible to user / other components

#### Existing Pod Conditions

When a container is stopped, pod condition `Ready` and `ContainersReady` will be marked as False. 

However, pod condition `Initialized` will not be marked as false, because currently it is assumed that once a pod is initialized, it cannot be “uninitialized”. The reasoning is that `PodRestart` should be considered “restarting all containers of the pod”, not necessarily recreating the pod itself.

#### Pod Phase

The pod pod should be in the `Pending` phase throughout the restart. This means if the pod was in the `Running` phase, it could be reverted to the `Pending` phase. This is possible today as well.

### Kubelet Implementation

The in-place pod restart will be implemented in the kubelet as a state machine based on the PodCondition mentioned above. If the PodRestartInPlace condition is true, then the pod is in the Termination Phase. Otherwise, it is considered the Startup Phase (which is the same as pod regular startup).

When a `RestartPod` rule is triggered, the kubelet will set the PodCondition `PodRestartInPlace=True`. In this state, the kubelet's only goal is to kill and remove all of the pod's containers. This process is similar to a normal pod shutdown but skips tearing down the sandbox. The container statuses from the previous run are preserved for history.

Once the kubelet verifies that all containers are removed, it transitions to startup phase by setting the PodCondition `PodRestartInPlace=False`. In this state, the kubelet's goal is to start the pod from the beginning, preserving the existing sandbox. This is the same as a normal pod startup sequence. 

#### Kubelet Restarts

If kubelet restarted in the Termination Phase, because the PodCondition is preserved on the API server, kubelet could continue the cleanup.
- If the kubelet did not preserve pod condition, it could also infer from the container statuses from the CRI that a RestartPod action is triggered.

If kubelet restarted in the Startup Phase, it proceeds normally as today by synchronizing all pods. From kubelet’s perspective, the pod just got created and assigned.

#### Node Restarts

On node restarts, kubelet and container runtime loses all containers. In the first pass, kubelet would sync the pods assigned to it. 

- If the pod was previously restarted in place, and was in the **Termination Phase**, it would have the pod condition `PodRestartInPlace=True`. Since kubelet sees all containers do not exist, it will set the pod condition `PodRestartInPlace=False` and proceed with normal pod start up sequence.

- If the pod was previously restarted in place, and was in the **Startup Phase**, then kubelet will proceed as if the pod just got created.

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

-   Create a pod with a container that has a `restartPolicyRule` with the `RestartPod` action. Verify that when the container (init, regular, or sidecar) exits with the specified code, the entire pod is restarted in-place (same UID, IP).
-   Verify that init containers are re-executed after a pod restart is triggered.
-   Verify that all regular and sidecar containers are restarted.
-   Verify that the `PodRestartInPlace` condition is added to the pod status during the restart and removed after it completes.

List of other restart sequences that need to be tested:

-   1st or 2nd init container fails and trigger RestartPod, all containers should be restarted.
-   Init container failing after RestartPod, restartPolicy=Never, the pod should be failed.
-   Init container failing after RestartPod, restartPolicy=Always, then init container eventually succeeds, the pod should be started.
-   Sidecar failing before the regular container started and triggers RestartPod, all containers should be restarted.


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

The size of the PodCondition API object will be increased for account for the new PodRestartInPlace status, example:
```
type: PodRestartInPlace
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

### Giving access to CRI API or subset

Pods could be given direct, albeit limited, access to the Container Runtime Interface (CRI) on the node. A container within the pod could make a CRI call to the kubelet or container runtime to request a restart of itself or other containers. To restart the whole pod, it would need to request the termination and recreation of all containers.

**Pros**

* Provides fine-grained control over the pod's lifecycle from within the pod itself.

**Cons**

* Major security risk. Exposing the CRI API, even a subset, to workloads is a significant security concern. A compromised container could potentially disrupt other pods on the same node.
* Breaks abstraction. It violates the abstraction layer between the pod and the node infrastructure. Pods should be managed by the Kubernetes control plane and kubelet, not manage themselves at the runtime level.
* Increased complexity for developers. Application developers would need to understand and interact with the CRI, which is a low-level infrastructure detail.

### Pod Self-orchestration

See also https://github.com/kubernetes/enhancements/issues/5309. This is a more structured version of the CRI access idea is to introduce a formal concept of an "orchestration container" within a pod. One container in the pod would be designated as the orchestrator. This container would have special privileges to manage the lifecycle (start, stop, monitor) of other containers within the same pod.

**Pros**
* Structured and flexible management of container lifecycle. Provides a very flexible way to handle complex inter-container dependencies and startup/shutdown sequences. Powerful for specific use cases Ideal for scenarios where containers have complex, ordered dependencies that go beyond existing pod lifecycle.

**Cons**
* High complexity. This is a much larger and more complex feature than what is needed for "restart the whole pod" action. Probably an overkill for the use-cases.
* Significant implementation effort. Implementing full pod self-orchestration is a major undertaking with wide-ranging implications for the kubelet and API.
* Imperative management of container lifecycles. Kubernetes tend to manage container and pod lifecycle declaratively. The imperative kill could have a steep learning curve for the users.

### Livesness probes on regular containers that point to a sidecar container

Liveness probes can be configured on regular containers that point to a sidecar container. The exit of (or the unexpected response from) the sidecar container can lead to liveness failures and causing regular containers to terminate / restart.

**Pros**

* The sidecar do not necessarily need to terminate, an error response is sufficient to trigger the regular container restart. 
* Reusing the existing probes, which could have some flexibility and tolerations.

**Cons**

* Does not trigger a full pod restart. It cannot re-run init containers or restart other containers.
* Indirect and complex. It can be difficult to debug and understand the relationship between the sidecar's health and the main container's lifecycle. It also need to be configured for every regular container to achieve this.
* There is a delay between the sidecar detecting the failure and the liveness probe failing, based on the probe's `periodSeconds` and `failureThreshold`.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
