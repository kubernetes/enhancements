# Provide fsgroup of pod to CSI driver on mount

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Currently for most volume plugins kubelet applies fsgroup ownership and permission based changes by recursively `chown`ing and `chmod`ing the files and directories inside a volume. For certain CSI drivers this may not be possible because `chown` and `chmod` are unix primitives and underlying CSI driver
may not support them. This enhancement proposes providing the CSI driver with fsgroup as an explicit field so as CSI driver can apply this on mount time.

## Motivation

Since some CSI drivers(AzureFile for example) - don't support chmod/chown - we propose that fsgroup of pod to be provided to the CSI driver on `NodeStageVolume`
and `NodePublishVolume` CSI RPC calls. This will allow the CSI driver to apply `fsgroup` as a mount option on `NodeStageVolume` or `NodePublishVolume` and kubelet can be freed
from responsibility of applying recursive ownership and permission change.

This feature hence becomes a prerequisite of CSI migration of Azure file driver and removal of Azure Cloud Provider.

### Goals

- Allow CSI driver to mount volumes with provided fsgroup.

### Non-Goals

- We are not supplying `fsGroup` as a generic ownerhip and permission handle to the CSI driver. We do not expect CSI drivers to `chown` or `chmod` files.

## Proposal

We are updating CSI specs by adding additional field called `volume_mount_group` to `NodeStageVolume` and `NodePublishVolume` RPC calls. The CSI proposal is available at - https://github.com/container-storage-interface/spec/pull/468 .

The CSI spec change is deliberately trying to avoid asking drivers to use supplied `fsGroup` as a generic handle for ownership and permissions. The reason being - Kubernetes may expect ownership and permissions to be in a way that is very platform/OS specific. We do not think CSI driver is right place to enforce all kind of different permissions expected by Kubernetes. The full scope of that discussion is out of scope for this enhancement and interested folks can follow along on - https://github.com/container-storage-interface/spec/issues/449


### Risks and Mitigations

I am not aware of any associated risks. If a driver can not support using `fsgroup` as a mount option, it can always use `FileFSGroupPolicy` and let kubelet handle the ownership and permissions.

## Design Details

We are proposing that when kubelet determines a CSI driver has `VOLUME_MOUNT_GROUP` node capability, the kubelet will use proposed CSI field `volume_mount_group` to pass pod's `fsGroup` to the CSI driver. Kubelet will expect that driver will use
this field for mounting volume with given `fsGroup` and no further permission/ownerhip change will be necessary.

It should be noted that if a CSI driver advertises `VOLUME_MOUNT_GROUP` node capability then value defined in `CSIDriver.Spec.FSGroupPolicy` will be ignored and kubelet will always use `fsGroup` as a mount option.

### Test Plan

Unit test:
1. Test that whenever supported pod's `fsGroup` should be passed to CSI driver via `volume_mount_group` field.

For alpha feature:
1. Update Azure File CSI driver to support supplying `fsGroup` via `NodeStageVolume` and `NodePublishVolume`.
1. Run manual tests against azurefile CSI driver.

For beta:
1. E2E tests that verify volume readability/writability using azurefile CSI driver.
2. E2E tests using CSI mock driver.

We already have quite a few e2e tests that verify generic fsgroup functionality for existing drivers - https://github.com/kubernetes/kubernetes/blob/master/test/e2e/storage/testsuites/fsgroupchangepolicy.go . This should give us a reasonable
confidence that we won't break any existing drivers.


### Graduation Criteria

#### Alpha -> Beta Graduation

- Since this feature is a must-have for azurefile CSI migration, we will perform testing of the driver.
- Currently CSI spec change is being introduced as alpha change and we will work to move the API change in CSI spec to stable.

#### Beta -> GA Graduation

- CSI spec change should be stable.
- Tested via e2e and manually using azurefile CSI driver.

### Upgrade / Downgrade Strategy

Currently there is no way to make a volume readable/writable using azurefile and `fsGroup` unless
pod was running as root.

When feature-gate is disabled, kubelet will no longer pass `fsGroup`  to CSI drivers and such volumes will not be readable/writable by the Pod. This feature is currently broken anyways.


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

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

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

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: MountWithFSGroup
    - Components depending on the feature gate:
      - Kubelet
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  Enabling this feature-gate could result in `CSIDriver.Spec.FSGroupPolicy` to be ignored for a driver that now has
  `VOLUME_MOUNT_GROUP` capability. Enabling this feature will cause volume to be mounted with `fsGroup`
  of the pod rather than kubelet performing permission/ownership change. This should result in quicker
  pod startup but still may surprise some users. This will be covered via release notes.

  We expect that once a CSI driver accepts provided `fsGroup` via `volume_mount_group` and mount
  operation is successful - the permissions should be correct on the volume. A user may write additional
  healthcheck to determine the permissions if necessary.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes - feature gate can be disabled once enabled and kubelet will fallback to its current behavior, which will
  result in any CSI driver that requires using `fsGroup` as a mount option to not use the mount option. In which case
  mounted volumes will be unwritable by any pods other than those running as root.

* **What happens if we reenable the feature if it was previously rolled back?**
  It will cause kubelet to use `volume_mount_group` field of CSI whenver applicable as discussed in above design. The pods running on affected
  nods have to restarted for feature to take affect though.

* **Are there any tests for feature enablement/disablement?**
  Not yet.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**
  One of the ways the rollout could fail is CSI driver defines `VOLUME_MOUNT_GROUP` capability but somehow does not implement correctly.
  This change should not affect running workloads (i.e running Pods).

  Rolling out the feature however may cause group permissions to be applied correctly which weren't applied before.

* **What specific metrics should inform a rollback?**
  If after enabling this feature a spike in `storage_operation_status_count{operation_name="volume_mount", status="fail-unknown"}` metric is observed
  then cluster admin should look into identifying root cause and rolling back the feature.

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
  We are going to split the metric that captures mount and permission timings. The full details are available in - https://github.com/kubernetes/kubernetes/issues/98667

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

* **Does this feature depend on any specific services running in the cluster?**
  This feature depends on presence of `fsGroup` field in Pod and CSI driver having `VOLUME_MOUNT_GROUP`
  capability.

  - [CSI driver]
    - CSI driver must have `VOLUME_MOUNT_GROUP` capability:
      - If this capability is not available in CSI driver then kubelet will try to use default mechanism of applying `fsGroup` to volume (which is basically `chown` and `chomid`). If underlying driver however does not support applying group permissions via `chown` and `chmod` then Pods will not run correctly.
      - Workloads may not run correctly or volume has to be mounted with permissions `777`.


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

- Feb 2 2021: KEP merged into kubernetes/enhancements repo

## Drawbacks

- N/A

## Alternatives

- An alternative to supplying fsgroup of the pod to the CSI driver is to mount volumes with 777 permissions or run pods. Both of these alternatives are not
  great and not backward compatible.

## Infrastructure Needed (Optional)

- Azure Cloudprovider access
