# Recovery from volume expansion failure

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation](#implementation)
    - [On using pvc.Spec.AllocatedResources](#on-using-pvcspecallocatedresources)
    - [User flow stories](#user-flow-stories)
        - [Case 1 (controller+node expandable):](#case-1-controllernode-expandable)
        - [Case 2 (controller+node expandable):](#case-2-controllernode-expandable)
        - [Case 3](#case-3)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
  - [Test Plan](#test-plan)
  - [Monitoring](#monitoring)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Why not use pvc.Status.Capacity for tracking quota?](#why-not-use-pvcstatuscapacity-for-tracking-quota)
  - [Fix Quota when resize succeeds with reduced size](#fix-quota-when-resize-succeeds-with-reduced-size)
  - [Allow admins to manually fix PVCs which are stuck in resizing condition](#allow-admins-to-manually-fix-pvcs-which-are-stuck-in-resizing-condition)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


## Summary

A user may expand a PersistentVolumeClaim(PVC) to a size which may not be supported by
underlying storage provider. In which case - typically expansion controller forever tries to
expand the volume and keeps failing. We want to make it easier for users to recover from
expansion failures, so that user can retry volume expansion with values that may succeed.

To enable recovery, this proposal relaxes API validation for PVC objects to allow reduction in
requested size. This KEP also adds a new field called `pvc.Spec.AllocatedResources` to allow resource quota
to be tracked accurately if user reduces size of their PVCs.


## Motivation

Sometimes a user may expand a PVC to a size which may not be supported by
underlying storage provider. Lets say PVC size was 10GB and user expanded
it to 500GB but underlying storage provider can not support 500GB volume
for some reason. We want to make it easier for user to retry re-sizing by
requesting more reasonable value say - 100GB.

Problem in allowing users to reduce size is - a malicious user can use
this feature to abuse the quota system. They can do so by rapidly
expanding and then shrinking the PVC. In general - both CSI
and in-tree volume plugins have been designed to never perform actual
shrink on underlying volume, but it can fool the quota system in believing
the user is using less storage than he/she actually is. To solve this
problem - we propose that although users are allowed to reduce size of their
PVC (as long as requested size >= `pvc.Status.Capacity`), quota calculation
will only use previously requested higher value. This is covered in more
detail in [Implementation](#implementation).


### Goals

- Allow users to cancel previously issued volume expansion requests, assuming they are not yet successful or have failed.
- Allow users to retry expansion request with smaller value than original requested size in `pvc.Spec.Resources`, assuming previous requests are not yet successful or have failed.

### Non-Goals

- As part of this KEP we do not intend to actually support shrinking of underlying volume.

## Proposal

As part of this proposal, we are mainly proposing three changes:

1. Relax API validation on PVC update so as reducing `pvc.Spec.Resoures` is allowed, as long as requested size is `>=pvc.Status.Capacity`.
2. Add a new field in PVC API called - `pvc.Spec.AllocatedResources` for the purpose of tracking used storage size. This field is only allowed to increase.
3. Update quota code to use `max(pvc.Spec.Resources, pvc.Spec.AllocatedResources)` when evaluating usage for PVC.

### Implementation

We propose that by relaxing validation on PVC update to allow users to reduce `pvc.Spec.Resources`, it becomes possible to cancel previously issued expansion requests or retry expansion with a lower value if previous request has not been successful. In general - we know that volume plugins are designed to never perform actual shrinking of the volume, for both in-tree and CSI volumes. Moreover if a previously issued expansion has been successful and user
reduces the PVC request size, for both CSI and in-tree plugins they are designed to return a successful response with NO-OP. So, reducing requested size will be a safe operation and will never result in data loss or actual shrinking of volume.

We however do have a problem with quota calculation because if a previously issued expansion is successful but is not recorded(or partially recorded) in api-server and user reduces requested size of the PVC, then quota controller will assume it as actual shrinking of volume and reduce used storage size by the user(incorrectly). Since we know actual size of the volume only after performing expansion(either on node or controller), allowing quota to be reduced on PVC size reduction will allow an user to abuse the quota system.

To solve aforementioned problem - we propose that, a new field will be added to PVC, called `pvc.Spec.AllocatedResources`. This field is only allowed to increase and will be set by the api-server to value in `pvc.Spec.Resources` as long as `pvc.Spec.Resources > pvc.Spec.AllocatedResources`.  Quota controller code will be updated to use `max(pvc.Spec.Resources, pvc.Spec.AllocatedResources)` when calculating usage of the PVC under questions.

#### On using pvc.Spec.AllocatedResources

We chose to use `pvc.Spec.AllocatedResources` for storing maximum of user requested pvc capacity because once user requests a new size in `pvc.Spec.Resources` even if she cancels the operation later on, resize-controller may have already started working on reconciling newly requested size.

AllocatedResources is not volume size but more like whatever user has requested and towards which resize-controller was working to reconcile. It is possible that user has requested smaller size since then but that does not changes the fact that resize-controller has already tried to expand to AllocatedResources and might have partially succeeded. So AllocatedResources is maximum user requested size for this volume and does not reflect actual volume size of the PV and hence is not recorded in `pvc.Status.Capacity`.

This also falls inline with how in-place update of Pod is handled(https://github.com/kubernetes/enhancements/pull/1342/files) and allocated resources is recorded in `pod.Spec.Containers[i].ResourcesAllocated`.

#### User flow stories

###### Case 1 (controller+node expandable):

- User increases 10Gi PVC to 100Gi by changing - `pvc.spec.resources.requests["storage"] = "100Gi"`.
- API Server sets `pvc.spec.allocatedResources` to 100Gi via pvc storage strategy.
- Quota controller uses `max(pvc.Spec.AllocatedResources, pvc.Spec.Resources)` and adds `90Gi` to used quota.
- Expansion to 100Gi fails and hence `pv.Spec.Capacity` and `pvc.Status.Capacity `stays at 10Gi.
- User requests size to 20Gi.
- Quota controller sees no change in storage usage by the PVC because `pvc.Spec.AllocatedResources` is `100Gi`.
- Expansion succeeds and `pvc.Status.Capacity` and `pv.Spec.Capacity` report new size as `20Gi`.
- `pvc.Spec.AllocatedResources` however keeps reporting `100Gi`.


###### Case 2 (controller+node expandable):

- User increases 10Gi PVC to 100Gi by changing - `pvc.spec.resources.requests["storage"] = "100Gi"`
- Api Server sets `pvc.spec.allocatedResources` to 100Gi via pvc storage strategy.
- Quota controller uses `max(pvc.Spec.AllocatedResources, pvc.Spec.Resources)` and adds `90Gi` to used quota.
- Expansion to 100Gi fails and hence `pv.Spec.Capacity` and `pvc.Status.Capacity `stays at 10Gi.
- User requests size to 20Gi.
- Quota controller sees no change in storage usage by the PVC because `pvc.Spec.AllocatedResources` is `100Gi`.
- Expansion succeeds and `pvc.Status.Capacity` and `pv.Spec.Capacity` report new size as `20Gi`.
- `pvc.Spec.AllocatedResources` however keeps reporting `100Gi`.
- User expands PVC to 120Gi
- Api Server sets `pvc.spec.allocatedResources` to 120Gi.
- Quota controller uses `max(pvc.Spec.AllocatedResources, pvc.Spec.Resources)` and adds `20Gi` to used quota.
- Expansion to 120Gi fails and hence `pv.Spec.Capacity` and `pvc.Status.Capacity `stays at 20Gi.

###### Case 3

- User increases 10Gi PVC to 100Gi by changing `pvc.spec.resources.requests["storage"] = "100Gi"`
- API server sets `pvc.spec.allocatedResources` to 100Gi via PVC storage strategy.
- Quota controller uses `max(pvc.Spec.AllocatedResources, pvc.Spec.Resources)` and adds 90Gi to used quota. (100Gi is the total space taken by the PVC)
- Volume plugin starts slowly expanding to 100Gi. `pv.Spec.Capacity` and `pvc.Status.Capacity` stays at 10Gi until the resize is finished.
- While the storage backend is re-sizing the volume, user requests size 20Gi by changing `pvc.spec.resources.requests["storage"] = "20Gi"`
- Quota controller sees no change in storage usage by the PVC because `pvc.Spec.AllocatedResources` is 100Gi.
- Expansion succeeds and pvc.Status.Capacity and pv.Spec.Capacity report new size as 100Gi, as that's what the volume plugin did.

### Risks and Mitigations

- One risk as mentioned above is, if expansion failed and user retried expansion(successfully) with smaller value, the quota code will keep reporting higher value. In practice though - this should be acceptable since such expansion failures should be rare and admin can unblock the user by increasing the quota or rebuilding PVC if needed. We will emit events on PV and PVC to alerts admins and users.

## Graduation Criteria

* *Alpha* in 1.19 behind `RecoverExpansionFailure` feature gate with set to a default of `false`. The limitation about quota should be clearly documented.
* *Beta* in 1.20: Since this feature is part of general `ExpandPersistentVolumes` feature which is in beta, we are going to move this to beta with confirmed production usage.
* GA in 1.21 along with `ExpandPersistentvolumes` feature. The list of issues for volume expansion going GA can be found at - https://github.com/orgs/kubernetes-csi/projects/12.

### Test Plan

* Basic unit tests for storage strategy of PVC and quota system.
* E2e tests using mock driver to cause failure on expansion and recovery.
* Also verify quota usage when this happens.

### Monitoring

* We will add events that are added to both PV and PVC objects for user reducing PVC size.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `RecoverExpansionFailure`
    - Components depending on the feature gate: kube-apiserver, kube-controller-manager, csi-external-resizer
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  Allow users to reduce size of pvc in `pvc.spec.resources`. In general this was not permitted before
  so it should not break any of existing automation. This means that if `pvc.Spec.AllocatedResources` is available it will be
  used for calculating quota.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  Yes the feature gate can be disabled once enabled. The quota code will use new field to calculate quota if available. This will allow quota code to use
  `pvc.Spec.AllocatedResources` even if feature gate is disabled.

* **What happens if we reenable the feature if it was previously rolled back?**
  This KEP proposes a new field in pvc and it will be used if available for quota calculation. So when feature is rolled back, it will be possible to reduce pvc capacity and quota will use `pvc.Spec.AllocatedResources`.

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling and disabling feature
  gates. However, unit tests in each component dealing with managing data created
  with and without the feature are necessary. At the very least, think about
  conversion tests if API types are being modified.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  This change should not impact existing workloads and requires user interaction via reducing pvc capacity.

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metrics. Operations against Kubernetes API (e.g.
  checking if there are objects with field X set) may be last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At the high-level this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide a comprehensive guidance, but at the very
  high level (they needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**
  Describe the metrics themselves and the reason they weren't added (e.g. cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  Think about both cluster-level services (e.g. metrics-server) as well
  as node-level agents (e.g. specific version of CRI). Focus on external or
  optional services that are needed. For example, if this feature depends on
  a cloud provider API, or upon an external software-defined storage or network
  control plane.

  For each of the fill in the following, thinking both about running user workloads
  and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high error rates on the feature:


### Scalability

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirms the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  None

* **Will enabling / using this feature result in introducing new API types?**
  None

* **Will enabling / using this feature result in any new calls to cloud
  provider?**
  None

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**
  Since this feature adds a new field to PersistentVolumeClaim object it will increase size of the object.
  Describe them providing:
  - API type(s): PersistentVolumeClaim
  - Estimated increase in size: 40B

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data send and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits][].

### Troubleshooting

Troubleshooting section serves the `Playbook` role as of now. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now we leave it here though.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
  For each of them fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without loogging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debugging the issue?
      Not required until feature graduated to Beta.
    - Testing: Are there any tests for failure mode? If not describe why.

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos


## Implementation History

- 2020-01-27 Initial KEP pull request submitted

## Drawbacks

- One drawback is while this KEP allows an user to reduce requested PVC size, it does not automatically reduce quota.


## Alternatives

There were several alternatives considered before proposing this KEP.

### Why not use pvc.Status.Capacity for tracking quota?


`pvc.Status.Capacity` can't be used for tracking quota because pvc.Status.Capacity is calculated after binding happens, which could be when pod is started. This would allow an user to overcommit because quota won't reflect accurate value until PVC is bound to a PV.

### Fix Quota when resize succeeds with reduced size

The first thing we considered was doing the *right* thing and ensuring that quota always reflects correct volume size. So if user increases a 10GiB PVC to 1TB and that fails but user retries expansion by reducing the size to 100GiB and it succeeds then quota will report 100GiB rather than 1TB.

There are several problems with this approach though:
- Currently Kubernetes always calculates quota based on user requested size rather than actual volume size. So if a 10GiB PVC is bound to 100GiB PV, then used quota is 10GiB and hence special care must be taken when reconciling the quota with actual volume size because we could break existing conventions around quota.
- The second problem is - using actual volume size for computing quota results in too many problems because there are multiple sources of truth. For example - `ControllerExpandVolume` RPC call of CSI driver may report different size than `NodeExpandVolume` RPC call. So we must fix all existing CSI drivers to report sizes that are consistent. For example - `NodeExpandVolume` must return a size greater or equal to what `ControllerExpandVolume` returns, otherwise it could result in infinte expansion loop on node. The other problem is - `NodeExpandVolume` RPC call in CSI may not report size at all and hence CSI spec must be fixed.

### Allow admins to manually fix PVCs which are stuck in resizing condition

We also considered if it is worth leaving this problem as it is and allow admins to fix it manually since the problem that this KEP fixes is not very frequent(there have not been any reports from end-users about this bug). But there are couple of problems with this:
- A workflow that depends on restarting the pods after resizing is successful - can't complete if resizing is forever stuck. This is the case with current design of statefulset expansion KEP.
- If a storage type only supports node expansion and it keeps failing then PVC could become unusable and prevent its usage.

## Infrastructure Needed [optional]
