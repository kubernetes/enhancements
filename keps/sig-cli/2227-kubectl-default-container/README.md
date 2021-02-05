# KEP-2227: default container behavior

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Current CLI Behaviors](#current-cli-behaviors)
  - [User Stories](#user-stories)
  - [Proposal Details](#proposal-details)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
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

Introduce an optional annotation for consumption by certain kubectl commands which will allow picking a default container.

## Motivation

Pods are composed of one or more containers. This leads to an extra effort by the operators when they need to run a command on a Pod that needs to target a specific container.

It gets worse because aside from a warning, this couples the default exec container to the container order.

As an example, with Service Mesh now a Pod can always have two containers: the main application and the sidecar. The container startup order impacts on which is the "default" container nowadays, leading to command executions against wrong containers.

We don't have a general default container name attribute for pod, and this would change pod spec and is not acceptable.

Having a well-known annotation that specifies what's the default container of that Pod reduces operation efforts and improves the user experience.

> However, it gets worse because aside from a warning, this couples the default exec container to the container order. 
> The container ordering also happens to have an impact on container startup order. We have started offering an option 
> to inject our sidecar as the first container (previously, it was the last one), which has resulted in users running 
> kubectl exec and getting the "wrong" container.

Quoted from [kubernetes #96986](https://github.com/kubernetes/kubernetes/issues/96986) opened by @howardjohn 

### Goals

- Provide a way for consumers (CLI, operators) to know which is the default Container of a Pod
- Deprecate the already in use annotation `kubectl.kubernetes.io/default-logs-container`

### Non-Goals

- If the cli is not kubectl, we don't determine which is the default container.
- Automatically define/create the default container annotation. This is an user operation.

## Proposal

Define a default well-known annotation for Pods called `kubectl.kubernetes.io/default-container` which points to Kubectl (and other commands) what's the default container to be used when the command needs this information.

Also this KEP proposes the change of behavior of kubectl commands that relies on the --container flag to read and make use of this annotation when the flag is not provided.

### Current CLI Behaviors

The following is the behavior of kubectl commands that can specify a container with --container:

- `kubectl attach`, `kubectl cp` and `kubectl exec`
  The three above have similar behavior: if --container flag is omitted, the first container of the Pod will be chosen.

- `kubectl logs`
  There's no default value. If a Pod have multiple containers, the operator needs to select which container to show the logs, or use the flag `--all-containers`
  Also, there's support for the annotation kubectl.kubernetes.io/default-logs-container in the Pod, that specified the default container to show the log. It will be deprecated with a warning message and be removed in 1.25.

- `kubectl debug`
  Use this option to specify container name for debug container. 

### User Stories

User story 1: Julia, the operator of a deployment called "backend" that generates some stack traces locally. Her environment uses Service Mesh, and because of this, every time she tries to copy those stack traces she gets an error because the file does not exist, and this happens because kubectl cp points to the service mesh sidecar as the first container.

Story 2: John, a developer of a PHP application always needs to run a command inside the application Pod. Because of the way John structured the Pod (a Container with NGINX and the other with php_fpm) every time he wants to execute this command on the php_fpm but he forgets to use the flag --container, and this way the command gets executed in the wrong container.

### Proposal Details

A single, generic and well known annotation for all above commands like `kubectl.kubernetes.io/default-container` is a good choice to avoid needing many new annotations in the future.

There are currently 3 commands that consume this annotation if `--container` is not specified.
- `kubectl exec`
- `kubectl attach`
- `kubectl cp`
- `kubectl logs`

However, there is an exception that `kubectl logs` will consume this annotation only if no `--all-containers` option is specified. If `--all-containers` is specified, all pods logs should be returned.

`kubectl debug` will not consume this annotation, as `--container` for `kubectl debug` is to speicify new debug container name to use. It is quite differenet with the annotation meaning here.

If `kubectl.kubernetes.io/default-logs-container` is specified, we should use this annotation instead of the general one for `kubectl log` and use general annotation for other commands. We need add a deprecation warn message for users with default-logs-container annotation and keep the old annotation working until 1.25.

### Notes/Constraints/Caveats

As the annotation `kubectl.kubernetes.io/default-container` will not be automatically added, users and Pod owners will need to be aware of this new annotation.

### Risks and Mitigations
**Note:** No server-side changes are required for this, all Request and Response template expansion is performed on
the client side.

- None

## Design Details

**Publishing Data:**

Alpha:  default container annotation

- Define a well known annotation `kubectl.kubernetes.io/default-container` in a Pod to provide a way to consumers (CLI, operators) to know which is the default Container.
- Define a global function GetDefaultContainerName for kubectl that uses this annotation like below
-- 1. if the command is `logs` check `--all-containers` flag at first: if it is specified, ignore container flag or annotations; if not, next step.
-- 2. check `-c`/`--container` flag: if it is specified, use it; if not, next step.
-- 3. check containers number in pod: if only one container, use its name; if more than one, next step.
-- 4. check annotations: 
-- 4.1 if command is `log` and pod has `kubectl.kubernetes.io/default-logs-container` annotation, then use the annotation value and print a warning message for deprecation(removing this step when GA); 
-- 4.2 check `kubectl.kubernetes.io/default-container` annotation: if specified, use it; if not, next step
-- 5. use the first container name as the default. Print a notice message before it.
- Add test cases to make sure that the command is running with the right container. When `--container` is specified, the annotation will be ignored.
- Validate the annotation value before using it, as the container name should follow RFC 1123. If the annotation value is invalid or not found in the pod, a warning message is needed before exiting.
- By default, this feature should be enabled as this feature is opt-in, and it only works once user adds the specified annotation to their pods.
- Ensure that when `kubectl.kubernetes.io/default-logs-container` is specified, we should use this annotation instead of the general one for `kubectl log` and use general annotation for other commands.

**Data Command Structure:**


**Example Command:**
Users might specify the `kubectl.kubernetes.io/default-container` annotation in a Pod to preselect container for kubectl exec and all kubectl commands.

An example Pod yaml is like below:

```
apiVersion: v1
kind: Pod
metadata:
 annotations:
    kubectl.kubernetes.io/default-container: sidecar-container
```

### Test Plan
Add a unit test for each command, testing the behavior with the annotation, without the annotation, and with the --container flag

### Graduation Criteria

#### Alpha -> Beta Graduation

As this is an opt-in feature, no gate is expected.
- At least 2 release cycles pass to gather feedback and bug reports during
- Documentations, add it to [well-known annotations docs](https://kubernetes.io/docs/reference/kubernetes-api/labels-annotations-taints/)
- Add a warning deprecation message when using the annotation `kubectl.kubernetes.io/default-logs-container`

#### Beta -> GA Graduation

- Gather feedback from developers and surveys
- At least 2 release cycles pass to gather feedback and bug reports during
- The deprecation message of the annotation `kubectl.kubernetes.io/default-logs-container` will be removed and this annotation will stop working.

### Upgrade / Downgrade Strategy

If kubectl is upgraded and no annotation is found, nothing happens.
If there's an annotation and kubectl is downgraded, the old behavior of using the first container will come back to the users.

### Version Skew Strategy
None

## Production Readiness Review Questionnaire


### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
  - [x] Other
    - Describe the mechanism:
      - This feature is explicitly opt-in since it need user to add specified
        annotation in pod.
    - Will enabling / disabling the feature require downtime of the control
      plane?
        - No. Disabling the feature would be a client behaviour.
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? 
        - No. It is a client behaviour only.

* **Does enabling the feature change any default behavior?**
  - No. The old behavior is notification when no default container is specified.
    Current behavior is cover that once specified annotation is in the pod spec.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  - Yes. When `--container` flag is specified, we will ignore
    the specified annotation.

* **What happens if we reenable the feature if it was previously rolled back?**
  - Nothing. It uses the first container as default in old behivior. It uses the 
    annotation with this feature implemented, the first container will be used
    if the annotation is not defined.

* **Are there any tests for feature enablement/disablement?**
  - There are unit tests in `staging/src/k8s.io/kubectl/pkg/cmd/exec/` and 
    `staging/src/k8s.io/kubectl/pkg/polymorphichelpers/` that
    verify the behaviour.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**
  - None
* **What specific metrics should inform a rollback?**
  - None
* **Were upgrade and rollback tested?**
  - None, or with adding and deleting the annotation from a pod to test it.
* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?** Even if applying deprecation policies, they may still surprise some users. No.
  - None

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**
  - A cluster-admin can checking which pods have the annotation.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - N/A, since it's just an annotation used for client-side hint.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  - N/A.

* **Are there any missing metrics that would be useful to have to improve observability of this feature? **
  - No.


### Dependencies
* **Does this feature depend on any specific services running in the cluster? **
  - No, since it's just an annotation used for client-side hint.

### Scalability
* **Will enabling / using this feature result in any new API calls?**
  - No, since it's just an annotation used for client-side hint.

* **Will enabling / using this feature result in introducing new API types?**
  - No.

* **Will enabling / using this feature result in any new calls to the cloud
provider?**
  - No.

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  - No.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  - No.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  - No.

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**
  - Same with original behavior.

* **What are other known failure modes?**
  - Same with original behavior.

* **What steps should be taken if SLOs are not being met to determine the problem?**
  - Use `-c`/`--container` flag to skip this feature.

## Implementation History

WIP in https://github.com/kubernetes/kubernetes/pull/97099 in 1.21.

## Drawbacks

It is not generic for other clients like go-client for kubernetes.

## Alternatives

Use `-c` or `--container` option in kubectl commands.
