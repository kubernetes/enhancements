<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [x] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-5381: Mutable PersistentVolume Node Affinity

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Handling race condition](#handling-race-condition)
  - [Extending CSI specification](#extending-csi-specification)
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
  - [New GRPC](#new-grpc)
  - [User Specified Topology Requirement](#user-specified-topology-requirement)
  - [Support for SPs that don't Know Attached Nodes](#support-for-sps-that-dont-know-attached-nodes)
  - [Confirming the Persisted Topology](#confirming-the-persisted-topology)
  - [Attaching Before Binding via NominatedNodeName](#attaching-before-binding-via-nominatednodename)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
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

This KEP proposes to make `PersistentVolume.spec.nodeAffinity` field mutable,
making it possible to change the affinity with VolumeAttributesClass.
This allows user to migrate data or enabling features without
interrupting workloads.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Currently, `PersistentVolume.spec.nodeAffinity` is set at creation time and cannot be changed.
But user may modify the volume to
taking advantage of new features provided by the storage provider,
or accommodate to the changes of business requirements.
These modification can be expressed by `VolumeAttributesClass` in Kubernetes.
But sometimes, A modification to volume comes with change to its accessibility, such as:
1. migration of data from one zone to regional storage;
2. enabling features that is not supported by all the client nodes.

In these scenarios, the `nodeAffinity` becomes inaccurate,
causing the scheduler to make decisions based on outdated information.
This results in pods:
* being scheduled to nodes that cannot access the volume, getting stuck in a `ContainerCreating` state;
* or being rejected from nodes that actually can access the volume, getting stuck in a `Pending` state.

By making `PersistentVolume.spec.nodeAffinity` field mutable,
we give storage providers a chance to propagate latest accessibility requirement to the scheduler,
improving the reliability of stateful pod scheduling.

### Goals

- Make `PersistentVolume.spec.nodeAffinity` field mutable.
- Enable CSI drivers to return a new accessibility requirement on ControllerModifyVolume.

### Non-Goals

- Modifying the core scheduling logic of Kubernetes.
- Implementing cloud provider-specific solutions within Kubernetes core.
- Re-scheduling running pods with volumes being modified,
  or directly interrupting workloads.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

1. Change APIServer validation to allow `PersistentVolume.spec.nodeAffinity` to be mutable.
2. Ensure scheduler will re-schedule pending pods that using a PV that has been updated (already implemented).
3. When a Pod is scheduled to a node that does not match volume node affinity, kubelet should fail the Pod.
4. Change CSI Specification to allow `ControllerModifyVolume` to return a new accessibility requirement.
5. Change external-resizer to set `PersistentVolume.spec.nodeAffinity` to the new accessibility requirement if returned by CSI driver.

### User Stories (Optional)

#### Story 1

As the owner of a stateful workload, I want to take advantage of the new
regional storage provided by the storage provider,
to improve the availability of my application.
I need a way to tell scheduler that the volume is now accessible in every zone,
so that the pod can be scheduled to another zone when the current zone is down.

In this case, the old affinity would be:
```yaml
required:
  nodeSelectorTerms:
  - matchExpressions:
    - key: topology.kubernetes.io/zone
      operator: In
      values:
      - cn-beijing-g
```
Or in the view of CSI accessibility requirement:
```json
[{"topology.kubernetes.io/zone": "cn-beijing-g"}]
```

The workflow:
1. User create a `VolumeAttributesClass`:
   ```yaml
   apiVersion: storage.k8s.io/v1beta1
   kind: VolumeAttributesClass
   metadata:
     name: regional
   driverName: csi.provider.com
   parameters:
     type: regional
   ```
2. User modify the `volumeAttributesClassName` in the PVC to `regional`
3. external-resizer initiate ControllerModifyVolume with `allow_topology_updates` set to true, `mutable_parameters` set to `{"type": "regional"}`
4. CSI driver blocks until the modification finished, then return with `accessible_topology` set to
   ```json
   [{"topology.kubernetes.io/region": "cn-beijing"}]
   ```
5. external-resizer sets `PersistentVolume.spec.nodeAffinity` to
   ```yaml
   required:
   nodeSelectorTerms:
   - matchExpressions:
     - key: topology.kubernetes.io/region
       operator: In
       values:
       - cn-beijing
   ```
   then update the PV status to indicate the modification is successful.


#### Story 2

As a cluster operator, I'm conducting an upgrade to new storage category provided by our storage provider.
However, once upgraded, the volume cannot be attached to some legacy nodes in the cluster.
I need a way to convey this new requirement to the scheduler,
so that my pod will not getting stuck in a `ContainerCreating` state.

In this case, the old affinity would be:
```yaml
required:
  nodeSelectorTerms:
  - matchExpressions:
    - key: provider.com/disktype.cloud_ssd
      operator: In
      values:
      - available
```
Or in the view of CSI accessibility requirement:
```json
[{"provider.com/disktype.cloud_ssd": "available"}]
```

Type A node only supports cloud_ssd, while Type B node supports both cloud_ssd and cloud_essd.
We will only allow the modification if the volume is attached to type B nodes.
And I want to make sure the Pods using new cloud_essd volume not to be scheduled to type A nodes.

In this case, it takes long to modify the volume, the new topology is not strictly less restrictive,
and SP wants to minimize the time window of the race condition:

The workflow:
1. User create a `VolumeAttributesClass`:
   ```yaml
   apiVersion: storage.k8s.io/v1beta1
   kind: VolumeAttributesClass
   metadata:
     name: essd
   driverName: csi.provider.com
   parameters:
     type: cloud_essd
   ```
2. User modify the `volumeAttributesClassName` in the PVC to `essd`
3. external-resizer initiate ControllerModifyVolume with `allow_topology_updates` set to true, `mutable_parameters` set to `{"type": "cloud_essd"}`
4. CSI driver returns with `in_progress` set to true, and `accessible_topology` set to
   ```json
   [{"provider.com/disktype.cloud_essd": "available"}]
   ```
5. external-resizer sets `PersistentVolume.spec.nodeAffinity` to
   ```yaml
   required:
     nodeSelectorTerms:
     - matchExpressions:
       - key: provider.com/disktype.cloud_essd
         operator: In
         values:
         - available
   ```
   but the PV status is not updated yet.
   From now on, the new Pod will be scheduled to nodes with `provider.com/disktype.cloud_essd: available`,
   maybe they will stuck in `ContainerCreating` state until the modification finishes.
6. external-resizer go back to step 3, retries until `in_progress` is set to false.
7. external-resizer update the PV status to indicate the modification is successful.


### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

It is storage provider's responsibility to ensure that the running workload is not interrupted while the data is being moved.

Whoever modifies the `PersistentVolume.spec.nodeAffinity` field should ensure that
no running Pods on nodes with incompatible labels are using the PV.
Kubernetes will not verify this. It is expensive and racy.

If the incompatibility does happen (i.e. someone updated nodeAffinity, making running Pods violate the new nodeAffinity),
we don't guarantee that those Pods will continue to run without any issue.
However, we try our best not to interrupt them:
- For volumes that not yet present in the Node.status.volumesAttached field,
  we fail the Pods that use them, since we are sure the Pods have never been running.
  (see [Handling race condition](#handling-race-condition) below)
- We will not detach the volume. So if the volume is actually accessible (depends on the storage provider), the Pod can continue to run.
- For CSI drivers with `requiresRepublish` set to true, we will stop calling NodePublishVolume periodically. and an event is emitted.
- For CSI drivers with `requiresRepublish` set to false, an event is emitted on kubelet restart. Otherwise the pod should continue to run.
It is not re-evaluated when the pod is already running.

Note that `Pod.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution` is similar.
Currently if the node labels change and the Pod nodeAffinity becomes incompatible,
the pod will continue to run until kubelet restarts, which will fail the pod.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

User may likely rollout workload and PV nodeAffinity changes at the same time.
This may trigger a race condition where the workload pods are scheduled to a node matchs the old nodeAffinity,
but the volume cannot be used on the node.

To mitigate this risk, we let kubelet to fail the mis-scheduled pods.
Hopefully, workload controller will create a replacement pod for it.

If the user is running an incompatible scheduler which does not respect PV nodeAffinity,
we may ended up in an endless loop of creating then failing pods.
This should be fine since we already have many cases like this.
We mitigate this by adding an note in the release note.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Handling race condition

There is a race condition between volume modification and pod scheduling:
1. User modifies the volume from storage provider.
2. A new Pod is created and scheduler schedules it with the old affinity.
3. external-resizer (or the operator) sets the new affinity to the PV.
4. KCM/external-attacher attaches the volume to the node, and find the affinity mismatch.

If this happens, the pod will be stuck in a `ContainerCreating` state.
Kubelet should detect this condition and reject the pod.
Hopefully some other controllers (StatefulSet controller) will re-create the pod and it will be scheduled to the correct node.

Specifically, kubelet should reject the pod (setting pod phase to 'Failed')
if the volume is not present in the `node.status.volumesAttached` list and the volume nodeAffinity does not match the current node
in `waitForVolumeAttach()`.

We check `volumesAttached` to ensure the Pods have never been running, to avoid interrupting running Pods.
We don't check the VolumeAttachment to make this also work for non-CSI volumes.

### Extending CSI specification

We will extend CSI specification to add:
```protobuf
// Specifies a capability of the controller service.
message ControllerServiceCapability {
  message RPC {
    enum Type {
      ...
      // TODO
      MODIFY_VOLUME_TOPOLOGY = 16 [(alpha_enum_value) = true];
    }

    Type type = 1;
  }

  oneof type {
    // RPC that the controller supports.
    RPC rpc = 1;
  }
}

message ControllerModifyVolumeRequest {
  option (alpha_message) = true;
  ...

  // Indicates whether the CO allows the SP to update the topology
  // as a part of the modification.
  bool allow_topology_updates = 4;
}

message ControllerModifyVolumeResponse {
  option (alpha_message) = true;

  // Specifies where (regions, zones, racks, etc.) the modified
  // volume is accessible from.
  // A plugin that returns this field MUST also set the
  // VOLUME_ACCESSIBILITY_CONSTRAINTS plugin capability.
  // An SP MAY specify multiple topologies to indicate the volume is
  // accessible from multiple locations.
  // COs MAY use this information along with the topology information
  // returned by NodeGetInfo to ensure that a given volume is accessible
  // from a given node when scheduling workloads.
  // This field is OPTIONAL. It is effective and replaces the
  // accessible_topology returned by CreateVolume if the plugin has
  // MODIFY_VOLUME_TOPOLOGY controller capability.
  // If it is not specified, the CO MAY assume
  // the volume is equally accessible from all nodes in the cluster and
  // MAY schedule workloads referencing the volume on any available
  // node.
  //
  // SP MUST only set this field if allow_topology_updates is set
  // in the request. SP SHOULD fail the request if it needs to update
  // topology but is not allowed by CO.
  repeated Topology accessible_topology = 1;

  // Indicates whether the modification is still in progress.
  // For a long-running modification, an SP SHOULD return with
  // in_progress set to true rather than blocking until the RPC times out.
  // SPs MAY also set in_progress to update the accessible_topology
  // before the modification finishes to reduce possible race conditions.
  // COs SHOULD retry the request if in_progress is set to true,
  // until in_progress is set to false.
  bool in_progress = 2;
}
```

When this new field is set, external-resizer will set `PersistentVolume.spec.nodeAffinity` accordingly, before it updates the PV status.

`in_progress` serves two purposes:
1. It lets the SP publish `accessible_topology` as early as it knows it, without waiting for the whole
   modification to finish, so external-resizer can update `PersistentVolume.spec.nodeAffinity` promptly and
   keep the scheduler's view as fresh as possible.
2. It gives external-resizer an explicit "still working" signal for long-running modifications. Without it, a
   slow modification would either block the RPC until it times out, or be indistinguishable from a stuck or
   failed call, causing external-resizer to surface spurious warnings. With `in_progress` set to true,
   external-resizer knows to simply retry later.

This mirrors the async pattern already in the CSI spec: `CreateSnapshot` reports progress via
`ready_to_use`, and the CO re-invokes the idempotent RPC until the snapshot is ready. We use the inverse
polarity (`in_progress`, whose proto3 default of `false` means "complete") so that a plain
`ControllerModifyVolume` from an SP that never sets the field is treated as already finished.

When anything unexpected happens (race between multiple resizer instances, crashes) and we lost track of the latest topology.
external-resizer will invoke `ControllerModifyVolume` again with the desired `mutable_parameters` to fetch the latest topology.

A new error condition of `ControllerModifyVolume` is added to CSI spec:

| Condition | gRPC Code | Description | Recovery Behavior |
|-----------|-----------|-------------|-------------------|
| Topology conflict | 9 FAILED_PRECONDITION | Indicates that the CO has requested a modification that would make the volume inaccessible to some already attached nodes. | Caller MAY detach the volume from the nodes that are in conflict and retry. |

But this KEP does not cover the automatic correction. Kubernetes should only retry with exponential backoff.


### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `pkg/apis/core/validation`: 2025-09-30 - 85.1%
- `pkg/kubelet/volumemanager`: 2025-09-30 - 72.1%
- `pkg/kubelet/volumemanager/reconciler`: 2025-09-30 - 82.7%

- Will test kubelet volume manager correctly fails the pods with mismatch volume node affinity
- Will test kubelet volume manager will not fail the pods with volumes already attached
- Will test that API validation allows volume node affinity update if the feature gate is enabled

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

- Test modifying PV `nodeAffinity` will trigger reschedule of pending pods.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

- Test a mis-scheduled pod will be failed and re-created on another node.
- Test modifying VolumeAttributesClass can properly update PV `nodeAffinity`.

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved 

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->


#### Alpha

- v1.35: `PersistentVolume.spec.nodeAffinity` made mutable behind the `MutablePVNodeAffinity` feature gate
- v1.38:
  - kubelet fails pods mis-scheduled onto nodes incompatible with the volume's nodeAffinity
  - CSI `ControllerModifyVolume` topology integration, with external-resizer setting the updated nodeAffinity
  - Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Additional tests are in Testgrid and linked in KEP
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved 

#### GA

- 2 examples of real-world usage
- Allowing 2 releases for feedback
- All issues and gaps identified as feedback during beta are resolved

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

APIServer and kubelet can be update / downgraded independently.

Upgrade external-resizer after APIServer to take advantage of the new feature if desired.
Otherwise, admin can also utilize the new feature manually with kubectl.

Downgrade/Reconfigure external-resizer before APIServer to avoid updating PV nodeAffinity being rejected.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

This feature involves changes to the kubelet, and APIServer. But they are not strongly coupled.

An n-3 kubelet will not able to fail the mis-scheduled pods. The mis-scheduled pods will stuck at ContainerCreating status.
If the kubelet is upgraded afterwards, it will properly fail those pods.
User can also manually delete the pods if they don't want to upgrade kubelet soon.
If user does not actually update the PV nodeAffinity, there will be no such mis-scheduled pods and everything should be fine.

kube-scheduler is not directly affected.
It just read the latest PV nodeAffinity for scheduling decision regardless of whether it's being updated or not.

An old external-resizer should work fine with new APIServer, since it will not update PV nodeAffinity.

external-resizer and the CSI driver can be upgraded in any order; all combinations work reasonably:
- An old external-resizer never sets `allow_topology_updates`, so the CSI driver does not change the topology.
  A driver that needs a topology change SHOULD reject the modification rather than silently breaking scheduling.
- A new external-resizer with an old CSI driver: the driver does not advertise the `MODIFY_VOLUME_TOPOLOGY`
  capability and ignores the unknown `allow_topology_updates` field, so it behaves as a plain `ControllerModifyVolume`
  with no topology change.
- A new external-resizer with a new CSI driver gets the full functionality.

The dangerous skew is a new external-resizer and new CSI driver against an **old APIServer** that does
not yet allow `PersistentVolume.spec.nodeAffinity` to be mutated.
The CSI driver may already have changed the volume's accessibility, but external-resizer's PATCH of the
PV nodeAffinity is rejected, leaving a stale nodeAffinity.
Only accessibility-*reducing* modifications are affected: a stale (broader) nodeAffinity can let the
scheduler place pods on nodes the volume no longer serves.
A widening modification only leaves a stale, more-restrictive nodeAffinity, which is safe.

To prevent this, external-resizer SHOULD dry-run (`?dryRun=All`) a nodeAffinity update before initiating a
topology-changing modification. If the dry-run is rejected, it does not set `allow_topology_updates`, so the
SP rejects the request before changing the volume. This catches the common case where the APIServer feature
gate is simply not enabled yet.

The recommended ordering remains: enable the feature gate on all APIServers before enabling it on
external-resizer, and reverse on downgrade.
Under an in-progress HA APIServer rollout, the dry-run and the real write may hit different APIServers,
leaving a small window where an accessibility-reducing modification completes but the affinity is not yet
persisted, and pods may get stuck on nodes the volume no longer serves.
This is bounded and self-healing: once the gate is enabled on all APIServers, external-resizer persists the
narrower affinity, and kubelet then fails and reschedules the affected pods.

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

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: MutablePVNodeAffinity
  - Components depending on the feature gate: kubelet, kube-apiserver, external-resizer

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

PV `spec.nodeAffinity` becomes mutable.

If a pod being scheduled to a node that is incompatible with the PV's nodeAffinity, the pod will fail.
Previously, it will be stuck at `ContainerCreating` status.

This should be rare before enabling this feature, since we don't allow PV nodeAffinity to be updated,
nor CSI driver can change the topology reported from NodeGetInfo.
So this is only possible if the user edited the node labels manually, or is running an incompatible scheduler.
Existing workflow will unlikely be affected by this behavior change.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes. Once disabled, PV node affinity cannot be updated any more.
Already updated PVs will still keep the updated node affinity.

###### What happens if we reenable the feature if it was previously rolled back?

Nothing special.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

Will add unit test to verify the validation and kubelet behavior when the feature gate is enabled or disabled.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

High value of `kubelet_admission_rejections_total{reason="VolumeNodeAffinity"}`

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
No

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
Unfortunately, no metrics records the update of a specific field.
Operator should check APIServer audit log.

Operator may also use the storage controller specific metrics.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

1. nodeAffinity can now be updated for existing volumes
2. pods that cannot be run due volume that can't be attached are now being failed by kubelet

As the consequences, if a Pod is previously stuck due to out-of-date PV nodeAffinity,
now user can update the PV to correct the nodeAffinity, and see the Pod entering Running state eventually.
For Pods stuck in ContainerCreating due to storage provider unable to attach the volume to the scheduled node,
The Pod will be rejected by kubelet and re-created at the correct node.
For Pods stuck in Pending due to no suitable node available,
scheduler will retry scheduling the Pod according to the updated nodeAffinity.

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
Count of PV nodeAffinity field update. We have so many fields, it is not reasonable to add a metric for each field or specific to this field.

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
No. But an external storage controller can depend on this feature.

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
No if unused.
One PATCH PV request from external storage controller or human operator per affinity update.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
No if unused.
Depends on the external storage controller implementation to make API calls to actually migrate the data in the volume.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
No.

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
Slightly increased CPU usage to check node affinity in kubelet.

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

Nothing changed.

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
- endless loop of pod failure and recreation
  - Detection: rapidly increasing `kubelet_admission_rejections_total{reason="VolumeNodeAffinity"}`
  - Mitigations: scale down the workload to zero. These pods should already not work
  - Diagnostics: scheduler logs to see why PV node affinity is ignored
  - Testing: No. This should not happen in a conformant cluster.

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
- 2025-09: targeting alpha in v1.35
- 2025-09-30: prototype implemented
- v1.35: alpha; `PersistentVolume.spec.nodeAffinity` made mutable
- 2026-07: proposing CSI spec changes; targeting a second alpha in v1.38 for kubelet enforcement and CSI integration

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### New GRPC

Instead of adding new fields to CSI GRPC `ControllerModifyVolume`, we could add a new GRPC `ControllerModifyVolumeTopology` (Other candidate names: `ControllerMigrateVolume`):

```protobuf
rpc ControllerModifyVolumeTopology (ControllerModifyVolumeTopologyRequest)
  returns (ControllerModifyVolumeTopologyResponse) {
    option (alpha_method) = true;
  }

message ControllerModifyVolumeTopologyRequest {
  option (alpha_message) = true;
  string volume_id = 1;
  map<string, string> secret = 2 [(csi_secret) = true];
  map<string, string> mutable_parameters = 3;
}

message ControllerModifyVolumeTopologyResponse {
  option (alpha_message) = true;
  repeated Topology accessible_topology = 1;
  bool in_progress = 2;
}
```

The workflow of this new GRPC is essentially the same as the current `ControllerModifyVolume` GRPC, but it allows SPs to mutate the accessible
topologies of volumes by default.

SPs with the `MODIFY_VOLUME_TOPOLOGY` controller capability should implement both this new GRPC and `ControllerModifyVolume`.
New COs that support modify volume topology (i.e. external-resizer) should only call the new GRPC when modifying volumes.
Old COs can continue to call `ControllerModifyVolume`. SPs should reject such requests if topology will be changed.

Comparison between these two approaches:
| Criteria | [PR 592](https://github.com/container-storage-interface/spec/pull/592) (Extended GRPC) | [PR 593](https://github.com/container-storage-interface/spec/pull/593) (New GRPC) |
| -------- | ---------------------- | ----------------- |
| Maintenance Difficulty | ✅ Low | ⚠️ High, need to also modify ControllerModifyVolumeTopology when making changes to ControllerModifyVolume |
| Implementation Complexity | ✅ Low | ⚠️ High, SPs will have to implement a new GRPC if they want to support topology modification even if they have implemented ControllerModifyVolume |
| Side Effects | ⚠️ Will impede the GA process of K8s VAC  | ✅ No influence on other features |

### User Specified Topology Requirement

Currently we don't support user specified topology requirement.
We've considered a design:
* Add `accessibility_requirements` in `ModifyVolumeRequest`, like that in `CreateVolumeRequest`
* Add `allowedTopologies` in `VolumeAttributesClass`, like that in `StorageClass`

We determine this lacks vaild use cases.

In most cases, SP can determine the desired topology of a volume from the `mutable_parameters`, or from the currently attached nodes.
An exception could be: modifying a volume from regional to zonal, and it is not attached to any node.
In this case, SP may need more information from the CO to determine the desired zone.
But we don't want user to create a separate VAC for each zone.
Instead, it maybe easier for user to just attach it to a node, so that SP can determine the desired zone.

For the other case (User Story 2), the topology (provider.com/disktype.cloud_essd) is actually not intended for user as an API.
User just want to modify the disk type, and we implement the underlying limitations as topology.
So we don't want to let user to specify this topology key directly.

Besides, this facing a lot of unresolved questions:
* How to merge `allowedTopologies` from `VolumeAttributesClass`, `StorageClass`?
* Should we use `allowedTopologies` from `StorageClass` if it is not specified in `VolumeAttributesClass`?
* Should we consider the topology of the currently attached nodes?
* Should we consider the topology of all the nodes in the cluster?

We may consider this again with vaild use cases.

### Support for SPs that don't Know Attached Nodes

Maybe there are some SPs that don't know the currently attached nodes,
so they cannot determine whether the topology change will break any workload.

Some kind of storage does not have persistent connection between client and server, such as object storage like S3.
But as network attached storage, they can be accessed wherever the network can reach.
So these SPs typically do not use the topology feature at all.

So, we decide that for an SP to support this feature,
they are required to properly detect potential breaking for existing workloads.

That said, the candidate design looks:
Add a new `dry_run` parameter to the `ControllerModifyVolumeRequest`.
CO first call `ControllerModifyVolume` with `dry_run=true` to fetch the new topology,
determine if the new topology is compatible with the existing workloads,
then decide whether to proceed the modification with `dry_run=false`.

Another way to get the new topology is further extending the "User Specified Topology Requirement" section,
Making it required for user to explicitly specify the new topology in the VAC and
remove `accessible_topology` from `ControllerModifyVolumeResponse`.
That is to say, SP must accept the new topology specified by user or it should reject the request.
The workflow will become:
1. User create a `VolumeAttributesClass`:
   ```yaml
   apiVersion: storage.k8s.io/v1beta1
   kind: VolumeAttributesClass
   metadata:
     name: regional
   driverName: csi.provider.com
   parameters:
     type: regional
   allowedTopologies:
   - matchLabelExpressions:
     - key: topology.kubernetes.io/region
       values:
       - cn-beijing
   ```
2. User modify the `volumeAttributesClassName` in the PVC to `regional`
3. external-resizer verifies all the nodes that all the nodes with this volume attached satisfy the `allowedTopologies`
4. external-resizer sets `PersistentVolume.spec.nodeAffinity` to
   ```yaml
   required:
   nodeSelectorTerms:
   - matchExpressions:
     - key: topology.kubernetes.io/region
       operator: In
       values:
       - cn-beijing
   ```
5. external-resizer initiate ControllerModifyVolume with `allow_topology_updates` set to true, `mutable_parameters` set to `{"type": "regional"}`
6. CSI driver blocks until the modification finished
7. external-resizer then update the PV status to indicate the modification is successful.

Besides the reasons mentioned above, we also facing a critical drawback for this design:
Topology can have many orthogonal aspects, such as above mentioned zone/region and disk type.
If SP cannot return the topology, user will need to be aware of all aspects of topology used by SP.
And SP will not able to extend the topology in the future, since VAC is immutable.

Note that the above designs are also racy.
We may still break some workloads that just started after the check but before the modification.

### Confirming the Persisted Topology

To make the new external-resizer/CSI + old APIServer skew (see [Version Skew Strategy](#version-skew-strategy))
fully safe, we considered adding a request field `current_accessible_topology`: the accessible topology the CO
currently advertises for the volume, derived from `PersistentVolume.spec.nodeAffinity`.
The SP would reduce a volume's accessibility only once this field shows the CO has already persisted the
narrower topology (persist-before-reduce). Because "persisted" is an etcd fact, this is robust even to an
in-progress HA APIServer rollout.

If the field is message-typed (so absent means a topology-unaware CO), it could also replace
`allow_topology_updates` — its presence is the capability signal — and subsume the `dry_run` preview above,
since the first call returns the target topology without reducing accessibility.

We defer it because:
- It does not eliminate the scheduler race: it confirms the write reached etcd, not that the scheduler's
  informer has observed the new PV. That race is already handled by kubelet failing mis-scheduled pods.
- Feature-enablement detection is adequately covered by a dry-run PATCH to the APIServer in the common case.
- Its only unique benefit — HA-skew-proof persist-before-reduce — applies only to accessibility-reducing
  modifications, only during an APIServer upgrade window, and only for SPs that can defer the reduction;
  the consequence without it is bounded and self-healing.
- It adds protocol surface and complexity: a `nodeAffinity` ↔ `Topology` round-trip in the CO, and a
  semantic comparison in the SP.

If real-world usage shows this skew matters, we can adopt it.

### Attaching Before Binding via NominatedNodeName

[KEP-5278](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5278-nominated-node-name-for-expectation)
(beta and on by default since v1.35, GA targeted for v1.37) has the scheduler set
`Pod.status.nominatedNodeName` at the beginning of a binding cycle to express its intended placement.
It writes the field only when the binding cycle will actually wait — a `Permit` plugin returned `Wait`, or a
`PreBind` plugin has work to do (decided by a new `PreBindPreFlight()` hook that lets a plugin return `Skip`) —
to limit the extra load on kube-apiserver. Today only the scheduler writes the field; other components consume
it read-only.

We could build on this to eliminate the race condition entirely, without the kubelet change. That needs two new
pieces on top of KEP-5278:
1. A new `VolumeAttachment` PreBind plugin that, for a pod with volumes to attach, returns non-`Skip` from
   `PreBindPreFlight()` (which is what makes the scheduler publish `nominatedNodeName`) and then blocks the
   binding cycle until the volume is attached to the chosen node.
2. Teaching the attach/detach controller to act on `nominatedNodeName` (not just `spec.nodeName`), so it
   creates the `VolumeAttachment` before the pod is bound.

The flow becomes: scheduler picks a node and sets `nominatedNodeName` → the attach/detach controller attaches
the volume to that node → the PreBind plugin waits for the attachment to succeed → scheduler binds. If the
attachment fails (the volume is not accessible on that node under the current topology), the plugin fails the
binding cycle; the scheduler clears `nominatedNodeName`, picks another node, and repeats.

Because attachment is verified against the volume's *actual* accessibility before the pod is bound, a pod is
never bound to a node where its volume cannot attach — regardless of any concurrent nodeAffinity update. This
removes the need for kubelet to fail mis-scheduled pods, and avoids the pod-failure/recreation churn.

We do not choose this because:
- It moves volume attachment into the scheduling critical path (attachment happens after binding today) and
  changes both the scheduler and the attach/detach controller — a larger and more cross-cutting change than the
  kubelet-side check. The controller must also detach from a nominated node it later abandons.
- Via the PreBind gate it adds an extra API write (`nominatedNodeName`) and an attach-wait to the binding cycle
  of every pod that needs a volume attachment — an always-on cost paid to guard a race that should be rare.
- It depends on KEP-5278 (only recently beta) plus a not-yet-designed extension of it.

The kubelet-side check costs nothing in the common case and only acts when the rare race actually occurs, so we
prefer it. This remains a clean option to revisit if the NominatedNodeName infrastructure matures or the
kubelet approach proves insufficient.

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
