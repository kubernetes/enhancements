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
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
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
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Currently for most volume plugins kubelet applies fsgroup ownership and permission-based changes by recursively `chown`ing and `chmod`ing the files and directories inside a volume. For certain CSI drivers this may not be possible because `chown` and `chmod` are unix primitives and underlying CSI driver
may not support them. This enhancement proposes providing the CSI driver with fsgroup as an explicit field so as CSI driver can apply this on mount time.

## Motivation

Since some CSI drivers(AzureFile for example) - don't support chmod/chown - we propose that fsgroup of pod to be provided to the CSI driver on `NodeStageVolume`
and `NodePublishVolume` CSI RPC calls. This will allow the CSI driver to apply `fsgroup` as a mount option on `NodeStageVolume` or `NodePublishVolume` and kubelet can be freed
from responsibility of applying recursive ownership and permission change.

This feature hence becomes a prerequisite of CSI migration of Azure file driver and removal of Azure Cloud Provider.

### Goals

- Allow CSI driver to mount volumes with provided fsgroup.

### Non-Goals

- We are not supplying `fsGroup` as a generic ownership and permission handle to the CSI driver. We do not expect CSI drivers to `chown` or `chmod` files.

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

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

N/A.

##### Unit tests

The new coded added should have complete test coverage, even though the module does not:

- `pkg/volume/csi`: `<Thu 06 Oct 2022>` - `76.2`

##### Integration tests

No integration tests are required. This feature is better tested with e2e tests.

##### e2e tests

We already have quite a few e2e tests that verify generic fsgroup functionality for existing drivers - https://github.com/kubernetes/kubernetes/blob/master/test/e2e/storage/testsuites/fsgroupchangepolicy.go. This should give us a reasonable
confidence that we won't break any existing drivers.


- [[sig-storage] CSI mock volume Delegate FSGroup to CSI driver [LinuxOnly] should pass FSGroup to CSI driver if it is set in pod and driver supports VOLUME_MOUNT_GROUP [Suite:openshift/conformance/parallel] [Suite:k8s]](https://storage.googleapis.com/k8s-triage/index.html?test=CSI%20mock%20volume%20Delegate%20FSGroup%20to%20CSI%20driver%20%5BLinuxOnly%5D%20should%20pass%20FSGroup%20to%20CSI%20driver%20if%20it%20is%20set%20in%20pod%20and%20driver%20supports%20VOLUME_MOUNT_GROUP)
- [[sig-storage] In-tree Volumes [Driver: azure-file] [Testpattern: Dynamic PV (default fs)] fsgroupchangepolicy (Always)[LinuxOnly], pod created with an initial fsgroup, new pod fsgroup applied to volume contents [Suite:openshift/conformance/parallel] [Suite:k8s]](https://storage.googleapis.com/k8s-triage/index.html?test=%5Bsig-storage%5D%20In-tree%20Volumes%20%5BDriver%3A%20azure-file%5D%20%5BTestpattern%3A%20Dynamic%20PV%20(default%20fs)%5D%20fsgroupchangepolicy%20(Always)%5BLinuxOnly%5D%2C%20pod%20created%20with%20an%20initial%20fsgroup%2C%20new%20pod%20fsgroup%20applied%20to%20volume%20contents%20)
- [[sig-storage] In-tree Volumes [Driver: azure-file] [Testpattern: Dynamic PV (default fs)] fsgroupchangepolicy (Always)[LinuxOnly], pod created with an initial fsgroup, volume contents ownership changed via chgrp in first pod, new pod with different fsgroup applied to the volume contents [Suite:openshift/conformance/parallel] [Suite:k8s]](https://storage.googleapis.com/k8s-triage/index.html?test=%5Bsig-storage%5D%20In-tree%20Volumes%20%5BDriver%3A%20azure-file%5D%20%5BTestpattern%3A%20Dynamic%20PV%20(default%20fs)%5D%20fsgroupchangepolicy%20(Always)%5BLinuxOnly%5D%2C%20pod%20created%20with%20an%20initial%20fsgroup%2C%20volume%20contents%20ownership%20changed%20via%20chgrp%20in%20first%20pod%2C%20new%20pod%20with%20different%20fsgroup%20applied%20to%20the%20volume%20contents)
- [[sig-storage] In-tree Volumes [Driver: azure-file] [Testpattern: Dynamic PV (default fs)] fsgroupchangepolicy (Always)[LinuxOnly], pod created with an initial fsgroup, volume contents ownership changed via chgrp in first pod, new pod with same fsgroup applied to the volume contents [Suite:openshift/conformance/parallel] [Suite:k8s]](https://storage.googleapis.com/k8s-triage/index.html?test=%5Bsig-storage%5D%20In-tree%20Volumes%20%5BDriver%3A%20azure-file%5D%20%5BTestpattern%3A%20Dynamic%20PV%20(default%20fs)%5D%20fsgroupchangepolicy%20(Always)%5BLinuxOnly%5D%2C%20pod%20created%20with%20an%20initial%20fsgroup%2C%20volume%20contents%20ownership%20changed%20via%20chgrp%20in%20first%20pod%2C%20new%20pod%20with%20same%20fsgroup%20applied%20to%20the%20volume%20contents%20)
- [[sig-storage] In-tree Volumes [Driver: azure-file] [Testpattern: Dynamic PV (default fs)] fsgroupchangepolicy (OnRootMismatch)[LinuxOnly], pod created with an initial fsgroup, new pod fsgroup applied to volume contents [Suite:openshift/conformance/parallel] [Suite:k8s]](https://storage.googleapis.com/k8s-triage/index.html?test=%5Bsig-storage%5D%20In-tree%20Volumes%20%5BDriver%3A%20azure-file%5D%20%5BTestpattern%3A%20Dynamic%20PV%20(default%20fs)%5D%20fsgroupchangepolicy%20(OnRootMismatch)%5BLinuxOnly%5D%2C%20pod%20created%20with%20an%20initial%20fsgroup%2C%20new%20pod%20fsgroup%20applied%20to%20volume%20contents%20)
- [[sig-storage] In-tree Volumes [Driver: azure-file] [Testpattern: Dynamic PV (default fs)] fsgroupchangepolicy (OnRootMismatch)[LinuxOnly], pod created with an initial fsgroup, volume contents ownership changed via chgrp in first pod, new pod with different fsgroup applied to the volume contents [Suite:openshift/conformance/parallel] [Suite:k8s]](https://storage.googleapis.com/k8s-triage/index.html?test=%5Bsig-storage%5D%20In-tree%20Volumes%20%5BDriver%3A%20azure-file%5D%20%5BTestpattern%3A%20Dynamic%20PV%20(default%20fs)%5D%20fsgroupchangepolicy%20(OnRootMismatch)%5BLinuxOnly%5D%2C%20pod%20created%20with%20an%20initial%20fsgroup%2C%20volume%20contents%20ownership%20changed%20via%20chgrp%20in%20first%20pod%2C%20new%20pod%20with%20different%20fsgroup%20applied%20to%20the%20volume%20contents%20)
- [[sig-storage] In-tree Volumes [Driver: azure-file] [Testpattern: Dynamic PV (default fs)] fsgroupchangepolicy (OnRootMismatch)[LinuxOnly], pod created with an initial fsgroup, volume contents ownership changed via chgrp in first pod, new pod with same fsgroup skips ownership changes to the volume contents [Suite:openshift/conformance/parallel] [Suite:k8s]](https://storage.googleapis.com/k8s-triage/index.html?test=%5Bsig-storage%5D%20In-tree%20Volumes%20%5BDriver%3A%20azure-file%5D%20%5BTestpattern%3A%20Dynamic%20PV%20(default%20fs)%5D%20fsgroupchangepolicy%20(OnRootMismatch)%5BLinuxOnly%5D%2C%20pod%20created%20with%20an%20initial%20fsgroup%2C%20volume%20contents%20ownership%20changed%20via%20chgrp%20in%20first%20pod%2C%20new%20pod%20with%20same%20fsgroup%20skips%20ownership%20changes%20to%20the%20volume%20contents)
- [[sig-storage] In-tree Volumes [Driver: azure-file] [Testpattern: Dynamic PV (default fs)] provisioning should provision storage with mount options [Suite:openshift/conformance/parallel] [Suite:k8s]](https://storage.googleapis.com/k8s-triage/index.html?test=%5Bsig-storage%5D%20In-tree%20Volumes%20%5BDriver%3A%20azure-file%5D%20%5BTestpattern%3A%20Dynamic%20PV%20(default%20fs)%5D%20provisioning%20should%20provision%20storage%20with%20mount%20options%20)


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
    - Feature gate name: DelegateFSGroupToCSIDriver
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
  The feature is in use if the feature gate DelegateFSGroupToCSIDriver is enabled in kubelet, and the CSI driver supports the `VOLUME_MOUNT_GROUP` node service capability.
  
  We have considered introducing a new metric with a label that identifies which fsgroup logic is used (https://github.com/kubernetes/kubernetes/issues/98667), but because this feature is small and simple enough, the benefit of such a label would be marginal.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [x] Metrics
    - Metric name: csi_operations_seconds
    - [Optional] Aggregation method: filter by `method_name=NodeStageVolume|NodePublishVolume`, `driver_name` (CSI driver name), `grpc_status_code`.
    - Components exposing the metric: kubelet
  - [ ] Other (treat as last resort)
    - Details:
    
  The `csi_operations_seconds` metrics reports a latency histogram of kubelet-initiated CSI gRPC calls by gRPC status code. Filtering by `NodeStageVolume` and `NodePublishVolume` will give us latency data for the respective gRPC calls which include FSGroup operations for drivers with `VOLUME_MOUNT_GROUP` capability, but analyzing driver logs is necessary to further isolate the problem to this feature.
  
  An SLI isn't necessary for kubelet logic since it just passes the FSGroup parameter to the CSI driver.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  
  For a particular CSI driver, per-day percentage of gRPC calls with `method_name=NodeStageVolume|NodePublishVolume` returning error status codes (as defined by the CSI spec) <= 1%.
  
  Latency SLO would be specific to each driver.

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**

  No

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
    
  No.

* **Will enabling / using this feature result in introducing new API types?**
  
  No.

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

  No.

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  
  No.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

  This feature does not have any impact on scalability.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].
  
  Not in Kubernetes components. CSI drivers may vary in their implementation and may increase resource usage.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

  This feature is part of the volume mount path in kubelet, and does not add extra communication with the API server, so this does not introduce new failure modes in the presence of API server or etcd downtime.
  
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
    
  In addition to existing k8s volume and CSI failure modes:
    
  - Driver fails to apply FSGroup (due to a driver error).
    - Detection: SLI above, in conjunction with the `DelegateFSGroupToCSIDriver` feature gate and `VOLUME_MOUNT_GROUP` node service capability in the CSI driver to determine if this feature is being used.
    - Mitigations: Revert the CSI driver version to one without the issue, or avoid specifying an FSGroup in the pod's security context, if possible.
    - Diagnostics: Depends on the driver. Generally look for FSGroup-related messages in `NodeStageVolume` and `NodePublishVolume` logs.
    - Testing: Will add an e2e test with a test driver (csi-driver-host-path) simulating a FSGroup failure.


* **What steps should be taken if SLOs are not being met to determine the problem?**

The CSI driver log should be inspected to look for `NodeStageVolume` and/or `NodePublishVolume` errors.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- Feb 2 2021: KEP merged into kubernetes/enhancements repo
- Sep 26 2022: Use latest template

## Drawbacks

- N/A

## Alternatives

- An alternative to supplying fsgroup of the pod to the CSI driver is to mount volumes with 777 permissions or run pods. Both of these alternatives are not
  great and not backward compatible.

## Infrastructure Needed (Optional)

- Azure Cloudprovider access
