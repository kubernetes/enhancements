# KEP-5607: Allow HostNetwork Pods to Use User Namespaces

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

This KEP proposes introducing a new feature gate to allow Pods to have both `hostNetwork` enabled and user namespaces enabled (by setting `hostUsers: false`).

## Motivation

The primary motivation is to enhance the security of Kubernetes control plane components. Many control plane components, such as the `kube-apiserver` and `kube-controller-manager` often run as static Pods and are configured with `hostNetwork: true` to bind to node ports or interact directly with the host's network stack.

Currently, a validation rule in the kube-apiserver prevents the combination of `hostNetwork: true` and `hostUsers: false`. This KEP aims to remove that barrier.

### Goals

* Introduce a new, separate alpha feature gate: `UserNamespacesHostNetworkSupport`.

* When this feature gate is enabled, modify the Pod validation logic to allow Pod specs where `spec.hostNetwork` is true and `spec.hostUsers` is false.

### Non-Goals

Including this functionality as part of the `UserNamespacesSupport` feature gate. As `UserNamespacesSupport` is nearing GA, it would be unwise to add a new, unstable feature with external dependencies.

## Proposal

We propose the introduction of a new feature gate named `UserNamespacesHostNetworkSupport`.

When this feature gate is disabled (the default state), the kube-apiserver will maintain the current validation behavior, rejecting any Pod spec that includes both `spec.hostNetwork: true` and `spec.hostUsers: false`.

When the `UserNamespacesHostNetworkSupport` feature gate is enabled, we will relax this validation check. 
The kube-apiserver will accept such a Pod spec and pass it on to the kubelet. 
At this point, the responsibility for successfully creating and running the Pod shifts to the container runtime. 
If the low-level container runtime (e.g., containerd/runc) does not support this combination, the pod will remain stuck in the `ContainerCreating` state and report an exception event, which is the expected behavior.

This change will primarily involve modifying the Pod validation function in pkg/apis/core/validation/validation.go to account for the state of the new feature gate.

### User Stories (Optional)

#### Story 1
As a cluster administrator, I want to enable user namespaces for my control plane static Pods (e.g., kube-apiserver, kube-controller-manager) to follow the principle of least privilege and reduce the attack surface. These Pods need to use hostNetwork to interact correctly with the cluster network. By enabling the new feature gate, I can add a critical layer of security isolation to these vital components without changing their networking model.


### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

If either the container runtime or the underlying container runtime does not support this feature, the container will fail to be created. To mitigate this issue, we will keep this feature in the alpha stage until mainstream container runtimes (containerd/runc) and mainstream underlying container runtimes (runc/crun) both support it, before promoting it to beta.

Users might upgrade the container runtime to a newer version on some nodes first, but pods could still be scheduled onto nodes that do not support this feature. In such cases, users can leverage [Node Declared Features](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5328-node-declared-features) to avoid this problem.


## Design Details

The core design change is very simple: in the apiserver's Pod validation logic, locate the code block that prevents the `hostNetwork: true` and `hostUsers: false` combination, and wrap it in a conditional that only executes the validation if the `UserNamespacesHostNetworkSupport` feature gate is disabled.

And add a parameter to `PodValidationOptions` so that if the `UserNamespacesHostNetworkSupport` feature gate is disabled, and the pod has already used the combination of `hostNetwork: true` and `hostUsers: false`, then we should allow updates the pod.

### Test Plan

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `pkg/apis/core/validation`: `2025-10-03` - `85.1%`

##### Integration tests

##### e2e tests

- Add e2e tests to ensure that pods with the combination of `hostNetwork: true` and `hostUsers: false` can run properly.

### Graduation Criteria

#### Alpha

- The `UserNamespacesHostNetworkSupport` feature gate is implemented and disabled by default.

#### Beta

- Mainstream container runtimes and low-level container runtimes (e.g., containerd/CRI-O, runc/crun) have released generally available versions that support the concurrent use of `hostNetwork` and user namespaces.
- Add e2e tests to ensure feature availability.
- Document the limitations of combining user namespaces and `hostNetwork` (e.g., CAP_NET_RAW, CAP_NET_ADMIN, CAP_NET_BIND_SERVICE remain restricted).

#### GA

- The feature has been stable in Beta for at least 2 Kubernetes releases.
- Multiple major container runtimes support the feature.


### Upgrade / Downgrade Strategy

Upgrade: After upgrading to a version that supports this KEP, the `UserNamespacesHostNetworkSupport` feature gate can be enabled at any time.

Downgrade: If downgraded to a version that does not support this KEP, kube-apiserver will revert to strict validation. Pods that were already running in this configuration will continue to run with this configuration.
If we were supposed to disable the feature, all pods using that configuration should be manually purged.

### Version Skew Strategy

A newer kube-apiserver with this feature enabled will accept such a Pod.

An older kubelet will still get the Pod definition from the kube-apiserver. 
It will attempt to create the Pod, if the container runtime version is too old and doesn't support this combination, the pod will be stuck in the ContainerCreating state.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `UserNamespacesHostNetworkSupport`
  - Components depending on the feature gate: `kube-apiserver`
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?
No. The behavior only changes when a user explicitly sets both `hostNetwork: true` and `hostUsers: false` in a Pod spec. 
The behavior of all existing Pods is unaffected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. It can be disabled by setting the feature gate to false and restarting the kube-apiserver.
This restores the old validation logic. 
When disabled, Pods that were running in this mode have to be manually purged. Otherwise, they will continue
to run in that mode (hostNetwork: true, hostUsers: false) even though it's technically disabled.

###### What happens if we reenable the feature if it was previously rolled back?
The kube-apiserver will once again begin to accept the combination of `hostNetwork: true` and `hostUsers: false`.
This is a stateless change, and reenabling is safe.

###### Are there any tests for feature enablement/disablement?

During the alpha stage, unit tests for enabling and disabling the toggle functionality will be added to the validation code. Manual testing will also be conducted during the beta stage, and the testing process will be documented here.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The [Version Skew Strategy](#version-skew-strategy) section covers this point.

###### What specific metrics should inform a rollback?

If a pod is stuck in the ContainerCreating state and returns events similar to the following, it indicates that the container runtime does not yet support this combination, and we should roll back this feature:
```
Failed to create pod sandbox: rpc error: code = Unknown desc = failed to start sandbox "0db019a96c2a28eaacb0d8a795bbbc48c8a3823d9b8e5099948f1d99e826238d": failed to generate sandbox container spec: failed to pin user namespace: failed to open netns(): open : no such file or directory
```

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be validated via manual testing. 

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

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?
No.

###### Will enabling / using this feature result in introducing new API types?
No.

###### Will enabling / using this feature result in any new calls to the cloud provider?
No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?
No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?
No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?
No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?
No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?
No impact to the running workloads

###### What are other known failure modes?
- Detection: Please refer to the content in the "What specific metrics should inform a rollback?" section.
- Mitigations: Users should roll back this feature and discontinue using the combination of `hostNetwork: true` and `hostUsers: false`.
- Diagnostics: If the following log appears in kubelet, it indicates an abnormality with this feature, and users need to roll back. The default log level is sufficient to obtain the following log:
```
  E1014 20:30:39.550653 2823108 pod_workers.go:1324] "Error syncing pod, skipping" err="failed to \"CreatePodSandbox\" for \"data-writer-pod_default(7607f6d7-91e1-4dbd-b957-c0d7b101de2e)\" with CreatePodSandboxError: \"Failed to create sandbox for pod \\\"data-writer-pod_default(7607f6d7-91e1-4dbd-b957-c0d7b101de2e)\\\": rpc error: code = Unknown desc = failed to start sandbox \\\"f47b48e3c415105d25fb316cf224c0e57b146b340c09d6847b2dfcf3b49c923c\\\": failed to generate sandbox container spec: failed to pin user namespace: failed to open netns(): open : no such file or directory\"" pod="default/data-writer-pod" podUID="7607f6d7-91e1-4dbd-b957-c0d7b101de2e"
```
- Testing: Failure mode tests have been run locally. We cannot add this test to the e2e test suite because once container runtime support is introduced, it will exit the failure mode, causing the test to fail.
```
opt kubectl get pods
NAME              READY   STATUS              RESTARTS   AGE
data-writer-pod   0/1     ContainerCreating   0          8m4s
➜  opt kubectl get event
LAST SEEN   TYPE      REASON                   OBJECT                MESSAGE
8m37s       Normal    Starting                 node/127.0.0.1
8m37s       Normal    RegisteredNode           node/127.0.0.1        Node 127.0.0.1 event: Registered Node 127.0.0.1 in Controller
8m7s        Normal    Scheduled                pod/data-writer-pod   Successfully assigned default/data-writer-pod to 127.0.0.1
8m7s        Warning   FailedCreatePodSandBox   pod/data-writer-pod   Failed to create pod sandbox: rpc error: code = Unknown desc = failed to start sandbox "f47b48e3c415105d25fb316cf224c0e57b146b340c09d6847b2dfcf3b49c923c": failed to generate sandbox container spec: failed to pin user namespace: failed to open netns(): open : no such file or directory
```


###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

* 2025-10-03: Initial proposal

## Drawbacks

There are no known drawbacks at this time.


## Alternatives

Add this feature to the existing `UserNamespacesSupport` feature gate:

  * This was ruled out because the `UserNamespacesSupport` feature is approaching GA, and its functionality should be stable.
Adding a new, externally-dependent, and immature behavior to a nearly-GA feature would introduce unnecessary risk and delays. Keeping the two feature gates separate is cleaner and safer.

Do not implement this feature:
  * Users can use `hostPort` as an alternative to `hostNetwork`, but this may cause some disruption to the existing user environment, as certain privileged containers require direct interaction with the host network stack. Moreover, `hostPort` requires pre-configured CNI; otherwise, the pod will fail to start. This limitation is precisely why Kubernetes control plane components continue to rely on `hostNetwork`.

## Infrastructure Needed (Optional)

No new infrastructure needed.