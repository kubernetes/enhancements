# KEP-2639: Secret/ConfigMap Protection

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

This KEP proposes a feature to protect Secrets/ConfigMaps while they are in use. Currently, users can delete a secret that is being used by other resources, like Pods, PVs, and VolumeSnapshots. This may have negative impact on the resources using the secret and it may result in data loss.

For example, one of the worst case scenarios is deletion of a controller-publish Secret or a provisioner Secret for CSI volumes while it is still in-use.
If it happens, a CSI driver for the volume can't controller-unpublish or delete the volume due to the lack of the Secret,
as a result, the volume remains published on the controller or the volume can't be deleted.
This issue will easily happen if a PVC for the volume and a Secret for the volume exist in the same namespace, and a user requests to delete the namespace, which starts deletion of all the resources in the namespace.
When the Secret is deleted before the PVC is deleted, this issue happens.
In addition, even if Secrets exist in a separate namespace, an admin or a user of the namespace may still mistakenly delete the Secrets.

Also, ConfigMaps can be deleted while they are in use, which may lead to an unexpected behavior in applications.

Similar features for protecting PV and PVC already exist as [pv-protection](https://github.com/kubernetes/enhancements/issues/499) and [pvc-protection](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/postpone-pvc-deletion-if-used-in-a-pod.md).


## Motivation

This feature aims to protect Secrets/ConfigMaps from being deleted while they are in-use.
Secrets can be used by below ways:
- From Pod:
  - [Mounted as data volumes](https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-files-from-a-pod)
  - [Exposed as environment variables](https://kubernetes.io/docs/concepts/configuration/secret/#using-secrets-as-environment-variables)
  - [Generic ephemeral volumes
](https://kubernetes.io/docs/concepts/storage/ephemeral-volumes/#generic-ephemeral-volumes) (can be handled as CSI PV below)
- From Deployment:
  - Specified through [DeploymentSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#deploymentspec-v1-apps) -> [PodTemplateSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#podtemplatespec-v1-core)
- From Replicaset:
  - Specified through [ReplicaSetSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#replicasetspec-v1-apps) -> [PodTemplateSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#podtemplatespec-v1-core) (Excluded from the scope, because [Replicaset should be managed through Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/replicaset/#when-to-use-a-replicaset))
- From StatefulSet:
  - Specified through [StatefulSetSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#statefulsetspec-v1-apps) -> [PodTemplateSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#podtemplatespec-v1-core)
- From DaemonSet:
  - Specified through [DaemonSetSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#daemonsetspec-v1-apps) -> [PodTemplateSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#podtemplatespec-v1-core)
- From CronJob:
  - Specified through [JobTemplateSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#cronjob-v1-batch) -> [JobSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#jobspec-v1-batch) -> [PodTemplateSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#podtemplatespec-v1-core)
- From Job:
  - Specified through [JobSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#jobspec-v1-batch) -> [PodTemplateSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#podtemplatespec-v1-core) (Excluded from the scope, because existence of a Job doesn't always mean creation of Pod in the future. Also see [here](https://kubernetes.io/docs/concepts/workloads/controllers/job/#clean-up-finished-jobs-automatically))
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

ConfigMaps can be used by below ways:
- From Pod, Deployment, Replicaset, StatefulSet, DaemonSet, CronJob, Job in the same way as Secrets

    Note that:
	- "From PV" case doesn't exist for ConfigMap
	- The same resources, or Replicaset and Job, are also out of scope for ConfigMap

### Goals

- Protect Secrets/ConfigMaps from being deleted while they are in use.
- Provide ways to dynamically enable/disable above feature per Secret/Configmap

### Non-Goals

- Protect important Secrets/ConfigMaps that aren't in use from being deleted
- Protect Secrets/ConfigMaps from being __updated__ while they are in use (Immutable Secrets/ConfigMaps will solve it)
- Protect resources other than Secrets/ConfigMaps from being deleted.
- Provide a generic mechanism that prevents a resource from being deleted when it shouldn't be deleted (Scope of [KEP-2839](https://github.com/kubernetes/enhancements/pull/2840))

## Proposal

New controllers to protect Secrets/ConfigMaps are introduced.

### User Stories

#### Story 1

A user creates Secrets/ConfigMaps and a pod using them. Then, the user mistakenly delete them while the pod is using them.
They are protected until the pod using them is deleted.

#### Story 2

A user creates a volume that uses a certain secret in the same namespace. Then, the user delete the namespace.
The secret is protected until the volume using the secret is deleted and the deletion of the volume succeeds.

#### Story 3

A user really would like to delete a secret despite that it is used by other resources.
The user force delete the secret while it is used by other resources, and the secret isn't protected and is actually deleted.
An example of such use cases is to update an immutable secret, by deleting and recreating.

### Notes/Constraints/Caveats (Optional)

- Compatibility:
  - There might be many existing scripts that don't care the order of deletion. Therefore, such scripts might stuck on Secret/ConfigMap deletion, if the deletion of the resources using the Secrets/ConfigMaps are done later.
- Usability:
  - Use of the Secret/ConfigMap in other resource will not be obvious to users. Therefore, users might not easily understand why the Secret/ConfigMap is not deleted.
  - Users might need to force delete the Secret/ConfigMap on deletion or would like to avoid protection for certain Secrets/ConfigMaps that already exist or that are newly created.

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
On the other hand, provisioner secrets that are defined by using templates with PVC information,
like "${pvc.name}", would be deleted in above case, due to the failure in finding a reference to the secret,
because the reference to provisioner secret is not stored in PV and needs to be resolved on deletion time with PVC information that is already deleted.
This behavior is inconsistent with other secrets, like node publish secret whose reference is stored in PV.

To avoid this kind of case, an [issue](https://github.com/kubernetes-csi/external-provisioner/issues/654) for adding information on secrets in PersistentVolume as annotations is in progress. This change will allow protecting provisioner secrets from deletion even if PVC or SC for PV is deleted.

## Design Details

This feature can be implemented by using Lien which is being discussed in [KEP-2839](https://github.com/kubernetes/enhancements/pull/2840).
A newly introduced Secret protection controller and a newly introduced ConfigMap protection controller update the `Liens` field of Secrets and ConfigMaps when it finds a reference to them from other resources that the controllers are watching.

Note that that Secret protection controller needs to handle `VolumeSnapshotContent`, which is out-of-tree resource.
Due to the restriction of out-of-tree resources, it needs to be handled by an external controller.
Therefore, it is designed to make the same external controller handle both in-tree resources and out-of-tree resources, instead of creating both an in-tree controller and an external controller, separately.
Also, ConfigMap protection controller is implemented as an external controller.

To minimize the risk of unintentionally blocking the deletion, this feature is opt-in.
Users need to add below to enable this feature per resource:
- `kubernetes.io/enable-secret-protection` annotation to Secret
- `kubernetes.io/enable-configmap-protection` annotation to ConfigMap

Basic flows of Secret protection controller and ConfigMap protection controller are as follows:

- Secret protection controller:
  - The controller watches `Pod`, `PersistentVolume`, `VolumeSnapshotContent`, `Deployment`, `StatefulSet`, `DaemonSet`, and `CronJob`
    - Create/Update events:
      - For all the `Secret`s referenced by the resource:
        1. Update the in-memory dependency graph (If the resource is using a Secret, add a dependency to the Secret that the resource is using the Secret. Also, the graph holds previous dependencies from the resource, so that the controller can remove a dependency to the Secret when the controller detects that the resource is no longer using the Secret).
        2. If number of dependency is changed:
           - Number becomes 0: Add the key for the Secret to delete-lien-queue
           - Number becomes 1+ from 0: Add the key for the Secret to add-lien-queue
           - Otherwise: Do nothing
    - Delete event:
      - For all the `Secret`s referenced by the resource:
        1. Update the in-memory dependency graph (If the resource is using a Secret, delete a dependency to the Secret that the resource is using it)
        2. If number of dependency is changed:
           - Number becomes 0: Add the key for the Secret to delete-lien-queue
           - Otherwise: Do nothing
 - The controller watches `Secret`
    - Create/Update events:
      - Check if the `Secret` have `kubernetes.io/enable-secret-protection: "yes"` annotation:
        - If yes:
          1. Check API server if the resources marked as using the secret in the in-memory dependency graph are still using the secret, and update the graph
          2. Check if there is no using resources in the graph any more
             - If yes: Add the key for the secret to delete-lien-queue
             - Otherwise: Add the key for the secret to add-lien-queue
        - Otherwise: Add the key for the secret to delete-lien-queue
  - The controller gets a key for a `Secret` from add-lien-queue:
    1. If the `Secret` doesn't have `kubernetes.io/enable-secret-protection: "yes"` annotation, do nothing
    2. Check API server if the `Secret` is actually used by one of the resources marked in the dependency graph, and if used, add `kubernetes.io/secret-protection` lien to the `Secret`
  - The controller gets a key for a `Secret` from delete-lien-queue:
    1. If the `Secret` doesn't have `kubernetes.io/enable-secret-protection: "yes"` annotation, delete `kubernetes.io/secret-protection` lien from the `Secret`
    2. Check API server if the `Secret` is actually not used by any of the resources, and if not used, delete `kubernetes.io/secret-protection` lien from the `Secret`

- ConfigMap protection controller can be implemented in almost the same way, except:
  - It handles ConfigMap instead of Secret
  - PVs aren't watched
  - `kubernetes.io/enable-configmap-protection: "yes"` annotation is checked
  - `kubernetes.io/configmap-protection` lien is added

Prototype implementation can be found [here](https://github.com/mkimuram/secret-protection/commits/lien).

For users to force delete the Secret, users need to either:
1. Remove `kubernetes.io/enable-secret-protection: "yes"` annotation
    ```
    kubectl annotate secret secret-to-be-deleted kubernetes.io/enable-secret-protection-
    ```

2. Remove the Liens, by below command:
    ```
    kubectl patch secret secret-to-be-deleted -p '{"metadata":{"liens":[]}}' --type=merge
    ```

For users to force delete the ConfigMap, users need to either:
1. Remove `kubernetes.io/enable-configmap-protection: "yes"` annotation
    ```
    kubectl annotate configmap configmap-to-be-deleted kubernetes.io/enable-configmap-protection-
    ```

2. Remove the Liens, by below command:
    ```
    kubectl patch configmap configmap-to-be-deleted -p '{"metadata":{"liens":[]}}' --type=merge
    ```

Annotation will be more user friendly than directly deleting liens.
However, annotation can't be used in the case that controllers are once deployed and undeployed later.
On the other hand, directly deleting liens might not work, because liens may be added by the controllers again, even after users manually remove the liens.
Therefore, directly deleting liens should be used after the controllers are undelpoyed.
For this case, it may be needed to provide a script to delete all the related liens on all the Secrets/ConfigMaps.

Note that above examples of removing liens remove all the liens from the resource.
So, when running these commands, we need to care that other controllers or users may have been add another Liens for other purposes (For the case _to force delete_, it won't matter because the user really would like to delete the resource anyway).


### Test Plan

- For Alpha, unit tests and e2e tests verifying that a Secret/ConfigMap used by other resources is protected by this feature are added.
  - unit tests:
    - Lien feature enabled:
      - Verify immediate deletion of a secret that is not used
      - Verify immediate deletion of a secret that is used but doesn't have annotation
      - Verify that secret used by a Pod as volume is not removed immediately
      - Verify that secret used by a Pod as EnvVar is not removed immediately
      - Verify that secret used by a Deployment, a StatefulSet, a Daemonset, and a CronJob as volume has proper Liens
      - Verify that secret used by a Deployment, a StatefulSet, a Daemonset, and a CronJob as EnvVar has proper Liens
      - Verify that secret used by a CSI PV as controllerPublishSecret is not removed immediately
      - Verify that secret used by a CSI PV as nodeStageSecret is not removed immediately
      - Verify that secret used by a CSI PV as nodePublishSecret is not removed immediately
      - Verify that secret used by a CSI PV as controllerExpandSecret is not removed immediately
      - Verify that secret used by a CSI PV as provisionerSecret is not removed immediately
      - Verify that secret used by a CSI PV as provisionerSecret via template is not removed immediately
      - Verify that secret used by a VolumeSnapshot is not removed immediately
      - Verify that secret used by a VolumeSnapshot as snapshotterSecret via template is not removed immediately
      - Verify immediate deletion of a ConfigMap that is not used
      - Verify immediate deletion of a ConfigMap that is used but doesn't have annotation
      - Verify that ConfigMap used by a Pod as volume is not removed immediately
      - Verify that ConfigMap used by a Pod as EnvVar is not removed immediately
      - Verify that ConfigMap used by a Deployment, a StatefulSet, a Daemonset, and a CronJob as volume has proper Liens
      - Verify that ConfigMap used by a Deployment, a StatefulSet, a Daemonset, and a CronJob as EnvVar has proper Liens
    - Lien feature disabled:
      - Verify immediate deletion of a secret that is not used
      - Verify immediate deletion of a secret that is used by a Pod
      - Verify immediate deletion of a ConfigMap that is not used
      - Verify immediate deletion of a ConfigMap that is used by a Pod
      - Verify immediate deletion of a secret that is not used
      - Verify immediate deletion of a secret used by a VolumeSnapshot
      - Verify immediate deletion of a secret used by a VolumeSnapshot as snapshotterSecret via template
  - e2e tests:
    - secret-protection-controller deployed:
      - Verify immediate deletion of a secret that is not used
      - Verify immediate deletion of a secret that is used but doesn't have annotation
      - Verify that secret used by a Pod as volume is not removed immediately
      - Verify that secret used by a Pod as EnvVar is not removed immediately
      - Verify that secret used by a Deployment, a StatefulSet, a Daemonset, and a CronJob as volume has proper Liens
      - Verify that secret used by a Deployment, a StatefulSet, a Daemonset, and a CronJob as EnvVar has proper Liens
      - Verify that secret used by a CSI PV as controllerPublishSecret is not removed immediately
      - Verify that secret used by a CSI PV as nodeStageSecret is not removed immediately
      - Verify that secret used by a CSI PV as nodePublishSecret is not removed immediately
      - Verify that secret used by a CSI PV as controllerExpandSecret is not removed immediately
      - Verify that secret used by a CSI PV as provisionerSecret is not removed immediately
      - Verify that secret used by a CSI PV as provisionerSecret via template is not removed immediately
      - Verify that secret used by a VolumeSnapshot is not removed immediately
      - Verify that secret used by a VolumeSnapshot as snapshotterSecret via template is not removed immediately
    - secret-protection-controller not deployed:
      - Verify immediate deletion of a secret that is used by Pod, PV, and VolumeSnapshot
    - configmap-protection-controller deployed:
      - Verify immediate deletion of a ConfigMap that is not used
      - Verify immediate deletion of a ConfigMap that is used but doesn't have annotation
      - Verify that ConfigMap used by a Pod as volume is not removed immediately
      - Verify that ConfigMap used by a Pod as EnvVar is not removed immediately
      - Verify that ConfigMap used by a Deployment, a StatefulSet, a Daemonset, and a CronJob as volume has proper Liens
      - Verify that ConfigMap used by a Deployment, a StatefulSet, a Daemonset, and a CronJob as EnvVar has proper Liens
    - config-protection-controller not deployed:
      - Verify immediate deletion of a ConfigMap that is used by Pod

- For Beta, scalability tests are added to exercise this feature.

- For GA, the introduced e2e tests will be promoted to conformance.

### Graduation Criteria
#### Alpha -> Beta Graduation

- Review the dependencies covered by the controllers are enough
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
- Upgrade: 
  - This feature requires that lien(InUseProtection feature gate) is enabled in the cluster.
  - To enable Secret protection, deploy secret-protection-controller.
  - To enable ConfigMap protection, deploy configmap-protection-controller.
- Downgrade:
  - To disable Secret protection, undeploy secret-protection-controller.
  - To disable ConfigMap protection, undeploy configmap-protection-controller.
  - If the cluster is downgraded to the version that doesn't support lien, this feature should be disabled, or controllers should be undeployed.

### Version Skew Strategy

- As for dependency on features, this feature depends on lien (InUseProtection feature gate), therefore, if the lien feature is disabled, this feature should also be disabled,
- As for resources, CSI Volume and CSI Snapshot are involved, changes in the API/CRD of these resources especially for their Secret/ConfigMap fields might cause issue. Howerver, this should be compatibility issue for these API/CRDs.

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

- [x] Other
  - Describe the mechanism:
    - Prerequisite: InUseProtection feature gate is enabled.
    - To enable:
      - Deploy secret-protection-controller to enable Secret protection
      - Deploy configmap-protection-controller to enable Configmap protection
    - To disable:
      - Undeploy secret-protection-controller to disable Secret protection
      - Undeploy configmap-protection-controller to disable Configmap protection
  - Will enabling / disabling the feature require downtime of the control
    plane? No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled). No.

###### Does enabling the feature change any default behavior?

Yes. Secrets/ConfigMaps aren't deleted until the resources using them are deleted.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. By undeploying the controllers.

###### What happens if we reenable the feature if it was previously rolled back?

Secrets/ConfigMaps aren't deleted until the resources using them are deleted, again.

###### Are there any tests for feature enablement/disablement?

Yes.

- Unit tests cover scenarios for lien feature (InUseProtection feature gate) disabled case.
- E2e tests for secret-protection-controller and configmap-protection-controller cover scenarios where the controller is deployed and undeployed.

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

There will be Secrets/ConfigMaps which have liens prefixed with `kubernetes.io/secret-protection` or `kubernetes.io/configmap-protection`.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: secret_protection
    - Aggregation method: prometheus
    - Components exposing the metric: secret-protection-controller
  - Metric name: configmap_protection
    - Aggregation method: prometheus
    - Components exposing the metric: configmap-protection-controller

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

- API call type: Update Secret/ConfigMap, Watch/List Secret/ConfigMap/Pod/PV/VolumeSnapshot/Deployment/StatefulSet/DaemonSet/CronJob, and Get PVC/SC
    (List is called only before deleting lien to make sure that there are no resource using Secret/ConfigMap. For other cases, watch or list from cache is used).
- estimated throughput: TBD
- originating component: secret-protection-controller and configmap-protection-controller
- API calls are triggered by changes of Secret, ConfigMap, Pod, PV, VolumeSnapshot, Deployment, StatefulSet, DaemonSet, and CronJob

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API type(s): Secret/ConfigMap
- Estimated increase in size: the size of Liens field per Secret/ConfigMap
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
  - Size: Increase in size of the Secret/ConfigMap is very limited.
  - Number of calls: Rate limit is set for the number of API calls.
Disk/IO:
  - No disk/IO are done through other than API calls and log outputs
CPU/RAM:
  - It works as common controller pattern. Therefore, number of resouces to process and the logic on how to detect in-use Secret/ConfigMap should only be needed to be checked.

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

- Manually adding/deleting finalizer to/from Secrets/ConfigMaps that are not deleted automatically in certain life cycle
- Implement in the same way as [pv-protection](https://github.com/kubernetes/enhancements/issues/499) and [pvc-protection](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/postpone-pvc-deletion-if-used-in-a-pod.md)
- Introduce a new kind of reference, like usedReference, and leave addition/deletion of it to users
  (Similar to ownerReference, but just block deletion and won't try to delete referenced resources through GC, like deleting child on parent's deletion).

Above ways will force users to do some additional works to protect Secrets/ConfigMaps.
