# KEP-2639: Secret Protection

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
    - [Removing a Deprecated Flag](#removing-a-deprecated-flag)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [ ] (R) Graduation criteria is in place
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

This KEP proposes a feature to protect secrets while it is in use. Currently, user can delete a Secret that is being used by other resources, like Pods, PVs, and VolumeSnapshots. This may have negative impact on the resouces using the Secret and it may result in data loss.

Similar features for protecting PV and PVC already exist as [pv-protection](https://github.com/kubernetes/enhancements/issues/499) and [pvc-protection](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/postpone-pvc-deletion-if-used-in-a-pod.md).


## Motivation

This feature aims to protect secrets from being deleted while they are in-use.
Secrets can be used by below ways:
- From Pod:
  - [Mounted as data volumes](https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-files-from-a-pod)
  - [Exposed as environment variables](https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-environment-variables)
  - [Generic ephemeral volumes
](https://kubernetes.io/docs/concepts/storage/ephemeral-volumes/#generic-ephemeral-volumes) (can be handled as CSI PV below)
- From PV:
  - [CSI](https://kubernetes-csi.github.io/docs/secrets-and-credentials-storage-class.html):
    - provisioner secret
    - controller publish secret
    - node stage secret
    - node publish secret
    - controller expand secret
  - non-CSI:
    - dependent on each storage driver and will be deprecated soon (Out of scope)
- [From VolumeSnapshot](https://kubernetes-csi.github.io/docs/secrets-and-credentials-volume-snapshot-class.html):
  - snapshotter secret

### Goals

- Protect secrets from being deleted while they are in use.

### Non-Goals

- Protect important secrets that aren't in use from being deleted
- Protect resources other than secrets from being deleted.

## Proposal

A new controller to protect secret is introduced.

### User Stories

#### Story 1

A user creates a secret and a pod using the secret. Then, the user mistakenly delete the secret while the pod is using it.
The secret is protected until the pod using the secret is deleted.

#### Story 2

A user creates a volume that uses a certain secret in the same namespace. Then, the user delete the namespace.
The secret is protected until the volume using the secret is deleted and the deletion of the volume succeeds.

#### Story 3

A user really would like to delete a secret despite that it is used by other resources.
The user force delete the secret while it is used by other resources, and the secret isn't protected and is actually deleted.

### Notes/Constraints/Caveats (Optional)

- Compatibility:
  - There might be many existing scripts that don't care the order of deletion. Therefore, such scripts might stuck on secret deletion, if the deletion of the resources using the secrets are done later.
- Usability:
  - Use of the secret in other resource will not be obvious to users. Therefore, users might not easily understand why the secret is not deleted.
  - Users might need to force delete the secret on deletion or would like to avoid protection for certain secrets that already exist or that are newly created.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->
There is a corner case when Reclaim Policy is Retain and PV is referencing secrets in a certain user namespace.
In such a case, when PVC is deleted, the PV referencing secrets remains as a resource not managed by the namespace, as a result, deletion of the secrets is blocked.
However, the user of the namespace won't notice why it is blocked and won't find a way to delete the secrets other than manually deleting finalizer.
In addition, to make things worse, provisioner secrets that are defined by using template with PVC information,
like "${pvc.name}", would be deleted in above case, due to the failure in finding a reference to the secret,
because the reference to provisioner secret is not stored in PV and needs to be resolved on deletion time with PVC information that is already deleted.
This behavior is inconsistent with other secrets, like node publish secret whose reference is stored in PV.

## Design Details

This feature can be implemented in the same way as `pv-protection/pvc-protection`.
Protection logics for in-tree resources and out-of-tree resources are separeted and independently work.
It is due to the restriction that CRDs can't be handled from in-tree controller.

- In-tree resources(`Pod` and `PersistentVolume`):
  - The deletion is blocked by using newly introduced `kubernetes.io/secret-protection` finalizer,
  - The `kubernetes.io/secret-protection` finalizer will be always added on creation of the secret by using admission controller for in-tree resources,
  - After the secret is requested to be deleted (`deletionTimestamp` is set), the `kubernetes.io/secret-protection` finalizer will be deleted by newly introduced in-tree `secret-protection-controller` by checking whether the secret is in-use, on every change(Create/Update/Delete) events for secrets and related resources.
  - If the `SecretInUseProtection` feature gate is disalbed, finalizer is added on creation, but deleted regardless of whether it is in-use after `deletionTimestamp` is set. This will allow users to avoid manually deleting finalizers on the downgrading by disabling the feature.
- Out-of-tree resources(`VolumeSnapshot`):
  - The deletion is blocked by using newly introduced `snapshot.storage.kubernetes.io/secret-protection` finalizer,
  - The `snapshot.storage.kubernetes.io/secret-protection` finalizer will be always added on creation of the secret by using admission controller for `VolumeSnapshot`,
  - After the secret is requested to be deleted (`deletionTimestamp` is set), the `snapshot.storage.kubernetes.io/secret-protection` finalizer will be deleted by newly introduced out-of-tree `secret-protection-volumesnapshot-controller` by checking whether the secret is in-use, on every change(Create/Update/Delete) events for secrets and related resources.

Feature gate `SecretInUseProtection` is used only for in-tree controller. Out-of-tree controller will be always enabled when the `secret-protection-volumesnapshot-controller` is deployed.

For users to force delete the secret, users need to do either:
1. manually delete the finalizer, by below command:
    ```
    kubectl patch secret/secret-to-be-deleted -p '{"metadata":{"finalizers":[]}}' --type=merge
    ```
2. add `secret.kubernetes.io/skip-secret-protection: "yes"` annotation to opt-out this feature per secret

Annotation will be more user friendly than directly deleting finalizer. However, it can't be used in the case that this feature is once enabled and disabled later by deleting the controller. For this case, it may be needed to provide a script to delete the finalizer on all the secrets.

### Test Plan

- For Alpha, unit tests and e2e tests verifying that a secret used by other resources is protected by this feature are added.
  - unit tests:
    - SecretInUseProtection enabled:
      - Verify immediate deletion of a secret that is not used
      - Verify that secret used by a Pod is not removed immediately
      - Verify that secret used by a CSI PV as controllerPublishSecret is not removed immediately
      - Verify that secret used by a CSI PV as nodeStageSecret is not removed immediately
      - Verify that secret used by a CSI PV as nodePublishSecret is not removed immediately
      - Verify that secret used by a CSI PV as controllerExpandSecret is not removed immediately
      - Verify that secret used by a CSI PV as provisionerSecret is not removed immediately
      - Verify that secret used by a CSI PV as provisionerSecret via template is not removed immediately
      - Verify that secret used by a VolumeSnapshot is not removed immediately
      - Verify that secret used by a VolumeSnapshot as snapshotterSecret via template is not removed immediately
    - SecretInUseProtection disabled:
      - Verify immediate deletion of a secret that is not used
      - Verify immediate deletion of a secret that is used by a Pod
      - Verify immediate deletion of a secret with finalizer that is used by a Pod
  - e2e tests:
    - Verify immediate deletion of a secret that is not used
    - Verify that secret used by a Pod is not removed immediately
    - Verify that secret used by a CSI PV as controllerPublishSecret is not removed immediately
    - Verify that secret used by a CSI PV as nodeStageSecret is not removed immediately
    - Verify that secret used by a CSI PV as nodePublishSecret is not removed immediately
    - Verify that secret used by a CSI PV as controllerExpandSecret is not removed immediately
    - Verify that secret used by a CSI PV as provisionerSecret is not removed immediately
    - Verify that secret used by a CSI PV as provisionerSecret via template is not removed immediately
    - Verify that secret used by a VolumeSnapshot is not removed immediately
    - Verify that secret used by a VolumeSnapshot as snapshotterSecret via template is not removed immediately
- For Beta, scalability tests are added to exercise this feature.
- For GA, the introduced e2e tests will be promoted to conformance.

### Graduation Criteria
#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys of users
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- Allowing time for feedback

#### Removing a Deprecated Flag

N/A

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->
- Upgrade: Secret that doesn't have `kubernetes.io/secret-protection` finalizer and/or `snapshot.storage.kubernetes.io/secret-protection` finalizer will be added the finalizers on Update/Delete events, therefore no additional user operation will be needed.
- Downgrade:
  - Feature disabled case:
    - If the `secret-protection-controller` exists and the feature is disabled, `kubernetes.io/secret-protection` finalizer will always be deleted, therefore no additional user operation will be needed,
    - If the `secret-protection-volumesnapshot-controller` exists and the feature is disabled, `snapshot.storage.kubernetes.io/secret-protection` finalizer will always be deleted, therefore no additional user operation will be needed,
  - Downgraded to no controller case:
    - If no `secret-protection-controller` exists but `kubernetes.io/secret-protection` finalizer is added to the secrets, no one removes the finalizer. Therefore, user needs to remove the `kubernetes.io/secret-protection` finalizer from all the secrets manually.
    - If no `secret-protection-volumesnapshot-controller` exists but `snapshot.storage.kubernetes.io/secret-protection` finalizer is added to the secrets, no one removes the finalizer. Therefore, user needs to remove the `kubernetes.io/secret-protection` finalizer from all the secrets manually.

### Version Skew Strategy

- As for the difference between in-tree components and out-of-tree components, they work independently, so they won't affect each other,
- As for in-tree components, the protection logic involves only in-tree admission controller and in-tree secret-protection-controller, so version skew won't happen unless these components are available with different versions,
- As for out-of-tree components, the protection logic involves only out-of-tree admission controller and out-of-tree secret-protection-volumesnapshot-controller, so version skew won't happen unless these components are available with different versions,
- As for resources, CSI Volume and CSI Snapshot are involved, changes in the API/CRD of these resources especially for their secret fields might cause issue. Howerver, this should be compatibility issue for these API/CRDs.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

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
###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: SecretInUseProtection
  - Components depending on the feature gate:
    - kube-controller-manager
    - secret-protection-controller (part of kube-controller-manager)
    - storageobjectinuseprotection admission plugin (part of kube-controller-manager)

Secret protection for volume snapshot will be enabled when those relevant out-of-tree controllers are deployed, but no feature gate is needed.

###### Does enabling the feature change any default behavior?

Yes. Secrets aren't deleted until the resources using them are deleted.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by disabling the feature gate. After the feature gate is disabled, users need to manually delete the `kubernetes.io/secret-protection` finalizer and the `snapshot.storage.kubernetes.io/secret-protection` finalizer to make secret deleted properly.

###### What happens if we reenable the feature if it was previously rolled back?

Secrets aren't deleted until the resources using them are deleted, again.

###### Are there any tests for feature enablement/disablement?

Yes, unit tests for the secret-protection-controller and secret-protection-volumesnapshot-controller cover scenarios where the feature is disabled or enabled.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

There will be secrets which have `kubernetes.io/secret-protection` finalizer and `snapshot.storage.kubernetes.io/secret-protection` finalizer.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: secret_protection_controller
    - Aggregation method: prometheus
    - Components exposing the metric: secret-protection-controller
  - Metric name: secret_protection_volumesnapshot_controller
    - Aggregation method: prometheus
    - Components exposing the metric: secret-protection-volumesnapshot-controller

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

<!--
At a high level, this usually will be in the form of "high percentile of SLI
per day <= X". It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code
-->

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
###### Will enabling / using this feature result in any new API calls?

- API call type: Update Secret, List Pod/PV/VolumeSnapshot, and Get PVC/SC
- estimated throughput: TBD
- originating component: secret-protection-controller and secret-protection-volumesnapshot-controller
- API calls are triggered by changes of secrets, Pod, PV, VolumeSnapshot

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API type(s): Secret
- Estimated increase in size: the size of `kubernetes.io/secret-protection` finalizer, `snapshot.storage.kubernetes.io/secret-protection` finalizer, and `secret.kubernetes.io/skip-secret-protection` annotation per secret
- Estimated amount of new objects: N/A

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

API:
  - Size: Increase in size of the Secret is very limited.
  - Number of calls: Rate limit is set for the number of API calls.
Disk/IO:
  - No disk/IO are done through other than API calls and log outputs
CPU/RAM:
  - It works as common controller pattern. Therefore, number of resouces to process and the logic on how to detect in-use secret should only be needed to be checked.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

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

- Manually adding/deleting finalizer to/from secrets that are not deleted automatically in certain life cycle
- Introduce a new kind of reference, like usedReference, and leave addition/deletion of it to users
  (Similar to ownerReference, but just block deletion and won't try to delete referenced resources through GC, like deleting child on parent's deletion).

Above ways will force users to do some additional works to protect secrets. Also, they are inconsistent with pv-protection/pvc-protection concepts.
