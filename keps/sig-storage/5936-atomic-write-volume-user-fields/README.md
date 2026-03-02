# KEP-5936: Add user fields to atomic write volumes

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: define owner UID of mounted volume files](#story-1-define-owner-uid-of-mounted-volume-files)
  - [Constraints](#constraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Changes to API Specs](#changes-to-api-specs)
  - [Interactions with existing features](#interactions-with-existing-features)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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

This KEP proposes adding optional `DefaultUser` and `User` fields to atomic write volumes, defining
owner UID of the written files. Atomic write volumes include ConfigMap, Secret, DownwardAPI
and Projected volumes.

This enables running software with strict file ownership requirements as non-root users,
and mounting files from atomic write volumes.

## Motivation

This KEP resolves a long-standing and recurring ask to configure atomic write volume files with proper
ownerships. There have been several issue tickets since 2014, each with a lot of comments and reactions,
demonstrating the strong requirements from the Kubernetes user base.

Many popular software requires strict file ownerships, such as MongoDB replica set
[key files][mongodb-key-files] and SSHD [host keys][sshd-host-keys]. The existing atomic write volume
implementation creates files owned by root and therefore not satisfying such ownership requirements.
A known workaround involves running an initContainers as root to perform chown.

However, this workaround is not possible in clusters implementing the [restricted][restricted-policy] pod
security standards policy that follow the current pod hardening best practices. The policy requires pods to
run as non-root and drop all capabilities, therefore rending this workaround impossible.  Moreover, even in
less hardened clusters, the workaround creates unnecessary friction and maintenance overhead for the users.

[mongodb-key-files]: https://www.mongodb.com/docs/manual/tutorial/enforce-keyfile-access-control-in-existing-replica-set/#enforce-keyfile-access-control-on-existing-replica-set
[sshd-host-keys]: https://man.openbsd.org/sshd_config#HostKey
[restricted-policy]: https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted

### Goals

1. Allow users to optionally define the desired file owner UID of atomic write volume files.

### Non-Goals

1. Define file owner GIDs. This is already covered by `PodSecurityContext.FsGroup` and
`PodSecurityContext.SupplementalGroups`.

2. Implement the feature for Windows pods.

3. Implement the feature for other volume types.

## Proposal

1. Introduce an optional `DefaultUser` field to atomic write volume API objects. Atomic write
volumes include ConfigMap, Secret, DownwardAPI and Projected volumes.

2. Introduce an optional `User` field to atomic write volume types child items API objects,
allowing users to define file owner UID per item.

3. When creating the atomic volume root directory, Kubelet configures its file owner UID according
to the `DefaultUser` field.

4. When writing files into the volumes, Kubelet configures their file owner UID according to
the `DefaultUser` and `User` fields.

### User Stories (Optional)

#### Story 1: define owner UID of mounted volume files

As a Kubernetes user, I want to run software that make use of files mounted from Kubernetes volumes
and define owner UID of the mounted files.

### Constraints

1. Do not support Windows pods.

This won't be implemented for Windows pods, since Windows doesn't support setting file ownership for virtualized
container accounts.

However, this is a common limitation of many related fields, such as `runAsUser`, `runAsGroup`, `fsGroup`,
`supplementalGroups`, `defaultMode` and `mode`.

2. Owner UIDs are defined statically.

To mount the same source to multiple containers under different UIDs,
it is necessary to define multiple volumes of the same source and different owner UIDs.

### Risks and Mitigations

Risks are minimal.

The new fields are optional. When the fields are defined, they affect an ephemeral volume only.
When the fields are not defined, the current default behavior is not changed.

## Design Details

### Changes to API Specs

```go
type SecretVolumeSource struct {
+       DefaultUser *int64
}

type ConfigMapVolumeSource struct {
+       DefaultUser *int64
}

type ProjectedVolumeSource struct {
+       DefaultUser *int64
}

type DownwardAPIVolumeSource struct {
+       DefaultUser *int64
}

type KeyToPath struct {
+       User *int64
}

type DownwardAPIVolumeFile struct {
+       User *int64
}

type ServiceAccountTokenProjection struct {
+       User *int64
}

type ClusterTrustBundleProjection struct {
+       User *int64
}

type PodCertificateProjection struct {
+       User *int64
}
```

### Interactions with existing features

1. `fsGroup` and `supplementalGroups`

The new UID fields do not interfere with or changes the behavior of `fsGroup` or `supplementalGroups`.

2. `runAsUser` and `runAsGroup`

The new UID fields do not interfere with or changes the behavior of `runAsUser` or `runAsGroup`.

3. User namespaces

The new UID fields do not interfere with or changes the behavior of user namespaces.

The owner UID always refer to the user inside the container.

Moreover, this proposal covers atomic write volumes only. All atomic write volumes are defined within a single pod only.
There are no concerns regarding different UID mappings across multiple pods.

4. Projected service account token volumes file ownership heuristic

[KEP-2451](kep-2451) introduced a heuristic - when all containers in a pod have the same `runAsUser`,
kubelet ensures that the files of the projected service account token volumes are
owned by that user and the permission mode set to `0600`.

If no owner UIDs are defined for the volume and the projection, this heuristic will continue to apply.

If any owner UIDs are defined for the volume or the projection, the explicitly defined UIDs have a
higher precedence and the heuristic will not be applied.

[kep-2451]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/2451-service-account-token-volumes

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No.

##### Unit tests

- `k8s.io/kubernetes/pkg/apis/core/validation/validation.go`: `2026-02-28` - `85.3`
- `k8s.io/kubernetes/pkg/volume/configmap`: `2026-02-28` - `76.4`
- `k8s.io/kubernetes/pkg/volume/downwardapi`: `2026-02-28` - `51.1`
- `k8s.io/kubernetes/pkg/volume/projected`: `2026-02-28` - `70`
- `k8s.io/kubernetes/pkg/volume/secret`: `2026-02-28` - `67.3`
- `k8s.io/kubernetes/pkg/volume/util/atomic_writer`: `2026-02-28` - `72.6`

##### Integration tests

##### e2e tests

Extend the existing volume end-to-end tests.

Create a `agnhost` test pod with the volume definition under test.
Make use of `mounttest` utility to verify file ownership of the files.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

### Upgrade / Downgrade Strategy

### Version Skew Strategy

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: AtomicWriteVolumeUserFields
  - Components depending on the feature gate:
    - kube-apiserver
    - kubelet

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

Existing pods are not affected. Only new pods will be affected.

###### What happens if we reenable the feature if it was previously rolled back?

Existing pods are not affected. Only new pods will be affected.

###### Are there any tests for feature enablement/disablement?

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No changes to kubelet SLOs.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `storage_operation_duration_seconds` (existing metric)
  - Aggregation method: filter by `volume_plugin` = one of
    `kubernetes.io/configmap`, `kubernetes.io/downward-api`, `kubernetes.io/projected` or `kubernetes.io/secret`
  - Components exposing the metric: kube-apiserver

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes.

The new optional `DefaultUser` and `User` fields of atomic write volume API objects have integer values of 64-bit.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Not applicable. Volume files of pods are not affected if the API server and/or etcd is unavailable.

###### What are other known failure modes?

Not applicable.

###### What steps should be taken if SLOs are not being met to determine the problem?

Not applicable.

## Implementation History

## Drawbacks

1. Additional complexity in the atomic write volume modules.

   However, the added complexity is minimal, since the feature is able to reuse the internal `FileProjection.FsUser`
   mechanism introduced by [KEP-1205][kep-1205-file-permission].

2. New fields in the volume API types.

   The fields are optional. Moreover, they do not alter the current default behavior if not defined.

[kep-1205-file-permission]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-auth/1205-bound-service-account-tokens#file-permission

## Alternatives

1. An idea of a `podSecurityContext.fsUser` has been proposed. However, I believe this KEP is preferrable
because of the following reasons.

    1. `podSecurityContext.fsUser` is a pod level construct.
    
       It doesn't support running multiple containers as different users and requiring different file owners per volumes or files.

    2. `podSecurityContext.fsUser` is a pod level construct and implies supporting all the volume types.

       This may also imply passing `fsUser` to CSI drivers, since Kubernetes is [currently passing][csi-fsgroup]
       `fsGroup` to CSI drivers.

    3. Its interaction with `fsGroupChangePolicy` is problematic.
    
       For example, users may reasonably expect
       `fsUser` to follow the same behavior of `fsGroupChangePolicy`. Adding `fsUser` may also entail a new
       `fsUserChangePolicy` feature.

2. Extend the current `runAsUser` heuristic of projected `serviceAccountToken`, `clusterTrustBundle` and `podCertificate`.
I.e. if all the pods have the same `runAsUser`, use that as the owner UID of the files created. There are some potential drawbacks.

    1. Breaking change of the current default behavior.
    
       The files are currently owned by root. Extending the heuristic changes the owner UID to `runAsUser`.

       We can make this an opt-in feature by introducing an optional `podSecurityContext.setAtomicVolumeOwnerFromRunAsUser`.

       If we implement this for other volume types in the future, we will also need one such field for each volume type.

    2. Do not support containers having different `runAsUser`.

       The current implemetation requires all containers having the same `runAsUser`.

       We can potentially extend it to look up the container `securityContext.runAsUser` and apply chown to volumes that is only mounted to one container.

       However, this is likely more complicated to implement and maintain. It is also harder for the users to reason about.

    3. It may be confusing that some volume types follow this heuristic but some don't.

       Users may reasonably assume that other volume types follow the same heuristic. User fields are clear and explicit.

[csi-fsgroup]: https://kubernetes.io/blog/2022/12/23/kubernetes-12-06-fsgroup-on-mount/

## Infrastructure Needed (Optional)

No.
