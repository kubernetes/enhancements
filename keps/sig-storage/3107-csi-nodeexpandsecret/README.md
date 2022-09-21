# NodeExpandSecret for CSI Driver

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User stories](#user-stories)
    - [story 1](#story-1)
    - [story 2](#story-2)
    - [story 3](#story-3)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

## Summary

This KEP proposes a way to add NodeExpandSecret to the CSI persistent
volume source and thus enabling the csi client to send it out as part of
the nodeExpandVolume request to the csi drivers for making use of it
in the various Node Operations.

## Motivation

### Goals

- Introduce `secretRef` in CSI Persistent Volume Source.
- Allow CSI driver to get/refer `secretRef` sent
  from kubelet as part of `NodeExpandVolume` operation.
- To support per-PVC secrets for volume resizing, similar to CSI attach and
  detach - this proposal expands `CSIPersistentVolumeSource` object to
  contain `NodeExpandSecretRef`.

### Non-Goals

- Other CSI calls e.g. `NodeStageVolume` will not have the secretRef
  in the request, this is limited to `NodeExpandVolume` operation.

## Proposal

Currently, the CSI drivers dont have a method to make use of secretRef
at time of Node operation (ex: nodeExpansion) as the subjected csi request does
not carry a secret or credentials in the request. Even-though
Kubernetes CSI have implemented similar mechanism for Controller side operations,
ie secretRef field available in the csi PV source and making use of it while
controllerExpand request has been sent to the CSI driver,  similar field 
is missing in the nodeExpandVolume request.

### User stories

#### story 1
- At times, the CSI driver need to check the actual size of the backend volume/image
  before proceeding on FS resize to avoid false positive returns on fs resize operation.

#### story 2
- Encrypted device with LUKs, which need the passphrase in order to resize
  the device on the node.

#### story 3
- For various validations at time of node expansion the CSI driver has to be connected
  to the backend storage cluster, if the secretRef is part of the nodeExpandVolume request
  the CSI driver can make use of the same and connect to the storage cluster 
  to perform the cluster operations.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

## Design Details

```go
- pkg/apis/core/types.go
..
type CSIPersistentVolumeSource struct {
    .....
    // nodeExpandSecretRef is a reference to secret object containing sensitive
    // information to pass to the CSI driver to complete CSI node expansion
    NodeExpandSecretRef *SecretReference
}
```
The above field NodeExpandSecretRef is optional:

To enable, NodeExpandSecretRef a new feature gate (CSINodeExpandSecret) has to be
introduced.

When the feature gate is enabled, the secretRef field will be added to the
NodeExpandVolume request.

Secrets will be fetched from StorageClass with parameters `csi.storage.k8s.io/node-expand-secret-name`
and `csi.storage.k8s.io/node-expand-secret-namespace`. Resizing secrets will support
same templating rules as attach and detach as documented
- https://kubernetes-csi.github.io/docs/secrets-and-credentials.html#controller-publishunpublish-secret .

CSI volumes that require secrets for online expansion will have NodeExpandSecretRef
field set. If not set NodeExpandVolume CSI RPC call will be made without secret.
Existing validation of PersistentVolume object will be relaxed to allow setting of
NodeExpandSecretRef for the first time so as CSI volume expansion can be supported
for existing PVs.

CSI Spec 1.5 has added below field to facilitate and enable COs to make use of the
same as part of the NodeExpandSecret

```
message NodeExpandVolumeRequest {
  ...
  // Secrets required by plugin to complete node expand volume request.
  // This field is OPTIONAL. Refer to the `Secrets Requirements`
  // section on how to use this field.
  map<string, string> secrets = 6
    [(csi_secret) = true, (alpha_field) = true];
}
```
The same field will be used by Kubernetes to fill secretRef in the
NodeExpandVolume request.

### Test Plan
[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

N/A

##### Unit tests

- Unit tests around all the added logic in kubelet.
- Unit tests around all the added logic in kube-apiserver.

The Unit tests for this feature is available [here](https://github.com/kubernetes/kubernetes/blob/master/pkg/api/persistentvolume/util_test.go#L31)
,[here](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/validation/validation_test.go)
and [here](https://github.com/kubernetes/kubernetes/blob/master/pkg/volume/csi/expander_test.go#L36)

##### Integration tests

N/A

##### e2e tests

- E2E tests around nodeExpandVolume to make sure the field value has passed
  and can be used, however these tests require external dependency like external-provisioner
  [sidecar](https://github.com/kubernetes-csi/external-provisioner/). Once added this
  support to mentioned sidecar, the e2e tests will be added and validated using
  example CSI driver [tests](https://github.com/kubernetes/kubernetes/blob/master/test/e2e/storage/drivers/csi-test/driver/driver.go).
- E2E test PR is available [here](https://github.com/kubernetes/kubernetes/pull/115451)
  and this test has been enabled in the [testgrid](https://k8s-testgrid.appspot.com/presubmits-kubernetes-nonblocking#pull-kubernetes-e2e-gce-cos-alpha-features)

### Graduation Criteria

#### Alpha

- Implemented the feature.
- implementation of unit tests.

#### Beta

- Deployed the feature in production and went through at least minor k8s
  version.
- Feedback from users.
- Implementation of e2e tests.

#### GA

#### Deprecation

### Upgrade / Downgrade Strategy

1. Upgrading a Kubernetes cluster with this feature flag enabled:
- in this upgraded cluster, a CSI driver should receive secrets as
part of NodeExpansion RPC call from CO side and should be able to
make use of it while expanding volumes on node.

2. Downgrading a Kubernetes cluster with feature disabled:
- in this downgraded cluster, a CSI driver will not receive secrets
as part of the NodeExpansion RPC call from CO side.
 
### Version Skew Strategy

The proposal requires changes to kubelet and kube api server feature
flag set. If any of the components are not upgraded to a version
supporting this feature, then the feature will not work as expected.
From an end user perspective, the existing behaviour will continue, ie,
there will be no facility to get the secrets as part of the node expansion
RPC call from CO side to the CSI driver.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

- **How can this feature be enabled / disabled in a live cluster?**

  - Feature gate name: CSINodeExpandSecret
  - Components depending on the feature gate: kubelet, kube-apiserver
  - Will enabling / disabling the feature require downtime of the control
    plane? no.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? yes.

- **Does enabling the feature change any default behavior?** no.

- **Can the feature be disabled once it has been enabled (i.e. can we roll
  back the enablement)?** yes, if rollback of feature gate happened with the
  field `NodeExpandRequest` set, it will exist, but be ignored.

- **What happens if we reenable the feature if it was previously rolled
  back?** nothing, as long as the new fields in `NodeExpandRequest` is not used.

- **Are there any tests for feature enablement/disablement?** yes, unit tests
  will cover this.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A failed scenario of rollout or rollback dont have any impact on running workloads.
The CSI drivers use the feature based on the availability of Secrets in NodeExpansion
call which is controlled by the Kubernetes feature flag set.

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

`csi_kubelet_operations_seconds` metric available
[here](https://github.com/kubernetes/kubernetes/blob/6b55f097bb2140381a58312aeede37fc76a0762e/pkg/volume/util/metrics.go#L66)
covers CSI NodeExpand operation which can be used for this purpose.

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->
###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

manual testing will be performed on upgrade and rollback.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

An operator can query for api server and kubelet flags in the cluster
for `CSINodeExpandSecret` flag and if it exist then the feature is
in use.


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
- [x] Other (treat as last resort)
  - Details: to make use of this feature in a cluster a StorageClass instance  has
to carry below entries in the parameter list.

  ```
  csi.storage.k8s.io/node-expand-secret-name
  csi.storage.k8s.io/node-expand-secret-namespace
 ```

  The subjected CSI PV object should have `nodeExpandSecretRef` field filled with the
  details given in the StorageClass.

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
  - Metric name: `csiOperationsLatencyMetric` can be used by an operator to determine
the health of the service.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

This feature depends on the cluster having CSI drivers and sidecars that use CSI
spec v1.5.0 at minimum.

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
- [CSI drivers and sidecars]
  - Usage description:
    - Impact of its outage on the feature: Inability to perform CSI storage
      operations with NodeExpandVolume RPC call where the CSI driver require
      credentials to complete this specific operation.
    - Impact of its degraded performance or high-error rates on the feature:
      Increase in latency performing CSI storage operations (due to repeated
      retries)

### Scalability

- **Will enabling / using this feature result in any new API calls?**
  no.
- **Will enabling / using this feature result in introducing new API types?**
  no.

- **Will enabling / using this feature result in any new calls to the cloud
  provider?** no.

- **Will enabling / using this feature result in increasing size or count of
  the existing API objects?**
  yes, this adds a new field to the API so it changes the size.

- **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs]?** no.

- **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?** no.

- **Can enabling / using this feature result in resource exhaustion of som
  node resources (PIDs, sockets, inodes, etc.)?** no.

### Troubleshooting

If the CSI driver does not receive the secrets as part of nodeExpansion
request, below things have to be checked in a cluster.

- make sure StorageClass has `csi.storage.k8s.io/node-expand-secret-name`
  and `csi.storage.k8s.io/node-expand-secret-namespace` parameters set
  with proper value.

- make sure `CSINodeExpandSecret` feature gate has been enabled for
  `kubelet` and `kube-apiserver` configuration in the cluster.

## Implementation History

- 18/01/2022: Implementation started

## Drawbacks

## Alternatives

1. Instead of fetching secretRef from the nodeExpansion request, CSI drivers
can store those somewhere in the cluster and make use of it while doing nodeExpansion,
however this is really a hacky way and not the CSI driver authors want.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
---
