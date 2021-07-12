# Allow VolumeSnapshot resources to be transferred between namespaces

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
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
  - [Alternative 1](#alternative-1)
  - [Alternative 2](#alternative-2)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
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

Currently, VolumeSnapshot resources are tied to a specific namespace. This KEP proposes a method for transferring
these resources between namespaces, allowing users to conveniently move these resources without manual creation
of each independent object (VolumeSnapshot, VolumeSnapshotContent).

This extends the [API for PVC Namespace Transfer](https://github.com/kubernetes/enhancements/pull/2110/files) to also apply for VolumeSnapshots.

## Motivation

Allow Kubernetes users to quickly and conveniently transfer VolumeSnapshots across namespaces.

### Goals 

- Extend the API proposed for PVC Namespace Transfer to apply for VolumeSnapshots

### Non-Goals

## Proposal

We are going to allow users to transfer VolumeSnapshot resources between namespaces, provided
that both the source and target namespace have a user that has agreed to the transfer.

The source and target VolumeSnapshot will point to the same VolumeSnapshotContent, which
references the backing snapshot on the storage backend.

Once the transfer is completed, the source VolumeSnapshot should be deleted, so that only
a single VolumeSnapshot stays mapped to a single VolumeSnapshotContent.

### User Stories

Users should be able to transfer VolumeSnapshots that reference the same VolumeSnapshotContent
across namespaces. This allows users to quickly share snapshots, utilizing data for testing
or backups in separate namespaces.

#### Story 1

The `prod` namespace contains a PVC that holds user information. VolumeSnapshots are taken of 
this PVC on a regular basis.

An existing VolumeSnapshot can be transferred to a test namespace. The user initiates the request
by creating a `StorageTransferRequest` in the source namespace, which will generate a token
on the created resource. 

Once created, a `StorageTransferAccept` resource in the target namespace will be matches, as outlined
in [Design Details](#design-details). Once matching resources are found, a PVC can be created from 
the snapshot in the new namespace for testing purposes.  

### Risks and Mitigations

The following scenarios must be considered and tested as appropriate:

- What to do with secrets are required to access the backing storage provider?

The CSI external-provisioner sidecar allows CSI drivers to specify secrets for performing operations
on the created volumes as defined in [StorageClass Secrets](https://kubernetes-csi.github.io/docs/secrets-and-credentials-storage-class.html).

We must allow the user to specify new secrets, in the event that they do not exist in the target namespace.
These can be provided in the `StorageTransferRequest`, as outlined above.

- What should be done when the PVC is currently in use?

The current approach of using a Finalizer and deleting the source PVC should address this, as it will 
wait for all running Pods to stop using the PVC before the transfer continues. 

This process can take some time, and we need to ensure users who request a transfer have the ability
to check the status of the transfer.

- Can a transfer be performed when the PVC is having an operation performed on it?

This issue is less of a concern for `VolumeSnapshots`, as operations are typically not performed on the
snapshot, but for PVCs it is possible the user could be resizing or snapshotting the PVC during
the transfer request.

The current approach of using a Finalizer and deleting the source PVC should address this concern.
For instance, the external-resizer can resume the operation on a PVC after the transfer is completed.
We should ensure that other sidecars can continue once the transfer is completed as well.

- As long as we reference the existing VolumeSnapshotContent, there should be no issues with encrypted VolumeSnapshotContents. The secret reference is stored on the VolumeSnapshotContent and VolumeSnapshotClass, not the VolumeSnapshot itself. If we take a different approach, such as re-creating the VolumeSnapshotContent, then we will need to ensure these credentials are propagated to the new content.
- The VolumeSnapshot must be `readyToUse` before the transfer begins.
- Any owners of the destination namespace must accept the transfer. By using the StorageTransfer API, this ensures an owner in the destination namespace is aware of the transfer.

## Design Details

This continues the discussion from [API for PVC Namespace Transfer](https://github.com/kubernetes/enhancements/pull/2110/) 
in regards to the StorageTransfer API.

A `StorageTransferRequest` resource will be created in the source namespace:

```yaml
apiVersion: storage.k8s.io/v1alpha1
kind: StorageTransferRequest
metadata:
  name: request-name
  namespace: source-namespace
spec:
  source:
    name: foo
    kind: VolumeSnapshot
  secrets:
    - name: secret-name
      namespace: secret-namespace
      type: secret-type # such as `csi.storage.k8s.io/provisioner-secret-name`
  acceptName: accept-name
  token: xyji6OwynW # This will be populated automatically by the controller
  targetName: bar
```

`StorageTransferAccept` resource will be created in the target namespace:

```yaml
apiVersion: storage.k8s.io/v1alpha1
kind: StorageTransferAccept
metadata:
  name: accept-name
  namespace: target-namespace
spec:
  sourceNamespace: source-namespace
  requestToken: xyji6OwynW
  requestName: request-name
```

 When matching `StorageTransferRequest` and `StorageTransferAccept` resources are detected, the 
 transfer process is initiated. We have the following matching criteria for the `StorageTransferRequest` 
 named `r` and the `StorageTransferAccept` `a`:
 - `r.metadata.name == a.spec.requestName`
 - `r.metadata.namespace == a.spec.sourceNamespace`
 - `r.spec.acceptName == a.metadata.name`
 - `r.spec.token == a.spec.requestToken`

Once matching is complete, a controller will begin the transfer process. For PVCs, this performs the following:
- Adds a Finalizer to the PVC source-namespace\foo
- Set the reclaim policy of the associated PersistentVolume Retain (if not already).
- Delete the PVC source-namespace\foo.
- Wait for all Pods to stop using the PVC source-namespace\foo
- Bind The PersistentVolume to target-namespace\bar.
- Create the PVC target-namespace\bar by copying the spec of source-namespace\foo and setting spec.volumeName appropriately.
- Reset the PersistentVolume reclaim policy
- Remove the Finalizer from the PVC source-namespace\foo

For VolumeSnapshots, this performs the following:
- Creates a VolumeSnapshot in the target namespace.
- Binds the VolumeSnapshotContent to the newly created VolumeSnapshot. If this was a dynamically provisioned VolumeSnapshotContent, then it adjusts the VolumeSnapshotContent.Spec.Source to point to the newly created
VolumeSnapshot.
- Deletes the source VolumeSnapshot.

The external-snapshotter will need to be updated to remove immutability of the following fields:
- VolumeSnapshotContent.VolumeSnapshotRef
- VolumeSnapshotContent.Spec.Source

In addition, the external-snapshotter will require updates to ensure that the VolumeSnapshot resource
in the target namespace can bind to a dynamically provisioned VolumeSnapshotContent. 

### Test Plan

E2E test for PVC:
1. Create a PVC containing data.
2. Wait for the PVC to be bound.
3. Transfer the PVC to a new namespace.
4. Confirm that the data matches what was initially created on the PVC
5. Confirm deletion of `StorageTransfer*` resources.

E2E test for VolumeSnapshot:
1. Create a PVC containing data.
2. Create a VolumeSnapshot
3. Wait for the VolumeSnapshot to be `readyToUse`.
4. Transfer the VolumeSnapshot to a new namespace.
5. Create a new PVC from the VolumeSnapshot.
6. Confirm that the data matches what was initially created on the PVC.
7. Confirm deletion of `StorageTransfer*` resources.

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include 
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

### Upgrade / Downgrade Strategy

- As an optional feature, there should be no issues during upgrade.
- When downgrading we'll need to handle existing StorageTransfer resources.

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

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Also set `disable-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?).

* **What happens if we reenable the feature if it was previously rolled back?**

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling or disabling feature
  gates. However, unit tests in each component dealing with managing data, created
  with and without the feature, are necessary. At the very least, think about
  conversion tests if API types are being modified.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

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
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

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

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
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

### Alternative 1

Instead of using the StorageTransfer API, we could utilize an annotation on the VolumeSnapshot
and configure the external-snapshotter to create a new VolumeSnapshot that references the
same VolumeSnapshotContent. This approach requires one less controller (the one that watches
for StorageTransfer resources), but invalidates the user accepting the transfer. As users may
not have authority in the target namespace, this approach is problematic.

### Alternative 2

Continuing the thought in the above alternative, the external-snapshotter could monitor for
multiple VolumeSnapshot resources that reference the same backing VolumeSnapshotContent. If
two snapshots exist, we could an annotation to each and delete the source resource once the
target VolumeSnapshot is marked `readyToUse`.

This is still a viable possibility; however, if the StorageTransfer API is used for transferring
PVCs, then we should be consistent with other storage types and use this API for snapshots.

## Infrastructure Needed (Optional)

None