# KEP-6480: CSI ControllerUnpublishVolume Reason

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
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubernetes calls `ControllerUnpublishVolume` for a few different reasons, and today the CSI driver gets the exact same request in every case. [CSI spec PR #613](https://github.com/container-storage-interface/spec/pull/613) adds a `reason` field so the CO can tell the driver why it made the call. This KEP proposes that external-attacher fill in that field. It works out the reason from the Node the volume was attached to, so no Kubernetes API, kube-controller-manager or kubelet changes are needed.

## Motivation

There are three cases where Kubernetes calls `ControllerUnpublishVolume`:

1. The ordinary path in the CSI spec, where kubelet has finished `NodeUnpublishVolume` and `NodeUnstageVolume`.
2. The attach/detach controller gave up waiting for kubelet - it does this when the node is NotReady and the volume is still in use 6 minutes after its pod was deleted (and also in v1.38+ when Node object is deleted). This violates the CSI spec, which requires node cleanup to finish first.
3. An operator took the node out of service with the `node.kubernetes.io/out-of-service` taint ([KEP-2268](https://kep.k8s.io/2268)), which tells Kubernetes the node is shut down, and the attach/detach controller detached without waiting for kubelet.

Cases 2 and 3 both arrive out of sequence, but they're meaningifully different - in case 2 nobody knows whether the node is still running and writing to the volume. In case 3 the operator has said it isn't. The driver can't tell any of these apart today, so it has to handle every call the same way.

This mostly matters for SPs that want to protect themselves when a call comes out of sequence. For example, on a backend where a normal detach can hang until it's forced, and forcing it can corrupt data if the node is still writing, the driver can force the detach in case 3 and never in case 2. Another driver might choose to refuse the detach in case 2. SPs that don't care can ignore the field.

### Goals

- Tell the CSI driver which of the three cases a `ControllerUnpublishVolume` call is.

### Non-Goals

- Changing when Kubernetes detaches a volume.
- Deciding what a driver should do with each reason.
- A real force detach or fencing mechanism for nodes that are still running but unreachable ([CSI spec PR #477](https://github.com/container-storage-interface/spec/pull/477)). This KEP doesn't get in the way of adding one later.

## Proposal

CSI spec PR #613 adds this enum, one value per case (spec comments shortened):

```protobuf
message ControllerUnpublishVolumeRequest {
  enum Reason {
    UNKNOWN = 0;
    NODE_CLEANUP_SUCCEEDED = 1; // all NodeUnstage/NodeUnpublish calls succeeded
    NODE_CLEANUP_TIMEOUT = 2;   // the CO stopped waiting for them
    NODE_OUT_OF_SERVICE = 3;    // the node is removed, powered down or rebooting
  }
  ...
  Reason reason = 4 [(alpha_field) = true];
}
```

When external-attacher's new `ControllerUnpublishReason` feature gate is enabled, it reads the Node right before it calls `ControllerUnpublishVolume` and picks the reason like this:

```go
func reason(node *v1.Node, volume v1.UniqueVolumeName) Reason {
	switch {
	case node != nil && hasOutOfServiceTaint(node):
		return NODE_OUT_OF_SERVICE
	case node == nil || slices.Contains(node.Status.VolumesInUse, volume):
		return NODE_CLEANUP_TIMEOUT
	default:
		return NODE_CLEANUP_SUCCEEDED
	}
}
```

kubelet keeps a volume in `status.volumesInUse` until it has unmounted it, and the attach/detach controller [checks the same list](https://github.com/kubernetes/kubernetes/blob/d0d1a17193d42dab5e893d21266bc5f8de16735f/pkg/volume/util/operationexecutor/operation_generator.go#L1511-L1534) before an ordinary detach. So if the volume is still listed when external-attacher calls `ControllerUnpublishVolume`, Kubernetes didn't wait for kubelet, which is case 2.

- A deleted Node is case 2, not case 3, because someone can delete a Node while its machine is still running.
- If external-attacher can't read the Node, it leaves the field unset.
- The Node is read on every call, so if an operator takes the node out of service while a detach is stuck, the next retry reports case 3.

### Risks and Mitigations

The out-of-service taint is now passed to the driver as a statement that the node is shut down, so tainting a node that's still running gives the driver wrong information. The [non-graceful node shutdown docs](https://kubernetes.io/docs/concepts/cluster-administration/node-shutdown/#non-graceful-node-shutdown) already say to only taint a node that's shut down.

## Design Details

external-attacher gets a new alpha feature gate, `ControllerUnpublishReason`, off by default, and its ClusterRole gets `get` on `nodes`.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates

The e2e tests need kubernetes/kubernetes to vendor a CSI spec release with `reason` and to bump the csi-attacher image used by csimock.

##### Unit tests

- `github.com/kubernetes-csi/external-attacher/v4/pkg/controller`: 2026-10-28 - 79.2%
- `github.com/kubernetes-csi/external-attacher/v4/pkg/attacher`: 2026-10-28 - 46.7%

New unit tests cover each Node state above, a failed Node GET, and the gate being off.

##### Integration tests

None, since there are no Kubernetes component changes.

##### e2e tests

New tests in `test/e2e/storage/csimock`:

- Deleting a pod sends `NODE_CLEANUP_SUCCEEDED`.
- With the mock driver failing `NodeUnpublishVolume` so the volume stays in use, deleting the `VolumeAttachment` by hand sends `NODE_CLEANUP_TIMEOUT`.
- `[Serial]`: tainting the node out of service and force deleting the pod sends `NODE_OUT_OF_SERVICE`.

### Graduation Criteria

#### Alpha

- A CSI spec release includes `reason`, and external-attacher vendors it.
- external-attacher sets `reason` behind the `ControllerUnpublishReason` gate, with unit tests.
- external-attacher has a metric that counts `ControllerUnpublishVolume` calls by reason.
- The csimock e2e tests run in CI.

#### Beta

- At least one CSI driver uses the field.
- The gate is on by default.

#### GA

- The CSI spec no longer marks `reason` as alpha.
- Two releases in beta without issues that need a design change.

#### Deprecation

N/A

### Upgrade / Downgrade Strategy

Enabling the gate needs the new `get nodes` permission, and without it the call goes out without a reason. Nothing is persisted, so downgrading or disabling the gate just stops setting the field.

### Version Skew Strategy

- Drivers built against an older CSI spec ignore the field.
- Older external-attacher versions leave it unset, which drivers already have to handle since the field is optional.
- The reason only depends on Node fields that every supported Kubernetes version sets.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ControllerUnpublishReason`
  - Components depending on the feature gate: external-attacher

###### Does enabling the feature change any default behavior?

external-attacher GETs the Node before each `ControllerUnpublishVolume` call and sets `reason`. Drivers that don't use the field see no difference.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by disabling the gate and restarting external-attacher.

###### What happens if we reenable the feature if it was previously rolled back?

external-attacher starts setting `reason` again on the next `ControllerUnpublishVolume` call.

###### Are there any tests for feature enablement/disablement?

Unit tests with the gate on and off.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Missing RBAC only means no reason is sent. A wrong reason only affects how the driver handles a volume Kubernetes is already detaching.

###### What specific metrics should inform a rollback?

More `ControllerUnpublishVolume` errors in `csi_sidecar_operations_seconds` after enabling the gate.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be tested by hand before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

external-attacher runs with `--feature-gates=ControllerUnpublishReason=true`.

###### How can someone using this feature know that it is working for their instance?

- [X] Other (treat as last resort)
  - Details: external-attacher logs the reason it sends with each `ControllerUnpublishVolume` call.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Enabling the gate shouldn't raise the `ControllerUnpublishVolume` error rate.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name: `csi_sidecar_operations_seconds`
  - Aggregation method: `method_name="/csi.v1.Controller/ControllerUnpublishVolume"`, by `grpc_status_code`
  - Components exposing the metric: external-attacher

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No. external-attacher will count `ControllerUnpublishVolume` calls by reason starting in alpha.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

One Node GET per `ControllerUnpublishVolume` call, retries included.

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

The same as today, and calls go out without a reason while the Node can't be read.

###### What are other known failure modes?

- external-attacher can't read Nodes.
  - Detection: Errors in the external-attacher logs.
  - Mitigations: Grant `get` on `nodes`, or disable the gate. Detaches keep working either way.
  - Diagnostics: The log message has the Node name and the API error.
  - Testing: Unit tests.

###### What steps should be taken if SLOs are not being met to determine the problem?

Compare the reasons and Node errors in the external-attacher logs with the driver's logs for the same volume.

## Implementation History

- 2026-09-24: CSI spec PR [#613](https://github.com/container-storage-interface/spec/pull/613) opened, approved on 2026-10-23.
- 2026-10-29: Initial KEP.

## Drawbacks

external-attacher has to work out the reason from the Node instead of being told by the attach/detach controller, which made the decision.

## Alternatives

The attach/detach controller could record the reason on the `VolumeAttachment` before deleting it, and external-attacher could pass it along. That needs a new `VolumeAttachment` field and new kube-controller-manager RBAC, and it only works on clusters running a new kube-controller-manager, while reading the Node works on any supported version.

It also wouldn't tell the driver much more since a/d controller decides from the same Node fields. The only edge case it would get right that this proposal doesn't is a kubelet restart that drops a still-mounted volume from `status.volumesInUse`. And because it's written once when the detach starts, it would miss an out-of-service taint added while the detach is stuck, which is exactly when the driver needs `NODE_OUT_OF_SERVICE`.
