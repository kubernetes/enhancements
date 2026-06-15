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
  - [Edge cases](#edge-cases)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
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

2. Introduce an optional `User` field to atomic write volume types child items API objects.
This allows users to define file owner UID per item that takes precedence over the volume level `DefaultUser`
field.

3. When writing files into the atomic write volumes, Kubelet configures their file owner UID according to
the `DefaultUser` and `User` fields.

### User Stories (Optional)

#### Story 1: define owner UID of mounted volume files

As a Kubernetes user, I want to run software that make use of files mounted from Kubernetes volumes
and define owner UID of the mounted files.

### Constraints

1. Do not support Windows pods.

This won't be implemented for Windows pods, since Windows doesn't support setting file ownership for
virtualized container accounts.

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

The following example demonstrates how the file owner UIDs are determined by the new user fields.

```yaml
volumes:
- name: volA
  configMap:
    defaultUser: 1000
    name: cm1
    items:
    - key: foo // Owner=defaultUser
      path: foo
    - key: bar // Owner=user
      path: bar
      user: 1001
- name: volB
  secret: // Owner=defaultUser
    defaultUser: 1000
    secretName: secret1
- name: volC
  secret:
    secretName: secret2
    items:
    - key: moo // Owner=root
      path: moo
    - key: baa // Owner=user
      path: baa
      user: 1000
```

### Changes to API Specs

The `DefaultUser` and `User` fields are optional and default to null. Both fields use the same valid range
as existing UID fields (e.g. `runAsUser`).

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

Moreover, this proposal covers atomic write volumes only. All atomic write volumes are defined
within a single pod only. There are no concerns regarding different UID mappings across multiple pods.

4. Projected service account tokens file ownership heuristic

[KEP-2451](kep-2451) introduced a heuristic - when all containers in a pod have the same `runAsUser`,
kubelet ensures that the files of the projected service account token volumes are
owned by that user and the permission mode set to `0600`.

The same heuristic is also implemented by projected cluster trust bundles ([KEP-3257](kep-3257)) and
pod certificates ([KEP-4317](kep-4317)).

While this heruistic continues to function, the new `defaultUser` and `user` fields provide higher
specificity and take precedence over the `runAsUser` value.

The file owner UID is evaluated per file in the following order of increasing precedence (where later
rules override earlier ones):

1. `runAsUser` UID, if all containers in the pod have the same `runAsUser`.
2. `defaultUser` UID, if the projected volume defined one.
3. `user` UID, if the volume source defined one.

The following example illustrates a mixed application of the existing heuristic and the new user fields:

```yaml
volumes:
- name: volA
  projected:
    sources:
    - serviceAccountToken: // Owner=runAsUser
        path: tokenA
    - serviceAccountToken: // Owner=user
        path: tokenB
        user: 1001
- name: volB
  projected:
    defaultUser: 1001
    sources:
    - serviceAccountToken: // Owner=defaultUser
        path: tokenA
    - serviceAccountToken: // Owner=user
        path: tokenB
        user: 1002
```

[kep-2451]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/2451-service-account-token-volumes
[kep-3257]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-auth/3257-cluster-trust-bundles
[kep-4317]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-auth/4317-pod-certificates

### Edge cases

1. Specify owner UID as root explicitly.

Configuring `defaultUser: 0` and `user: 0` are valid and ensures that kubelet explicitly enforces
root file ownership.

Although root (UID 0) is the default file owner of atomic write volumes, some projected sources
implement a heuristic to use the `runAsUser` UID.

Specifying 0 guarantees root ownership regardless of this heuristic or other mechanisms introduced
in the future.

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

No.

This feature is more effectively validated with unit tests and e2e tests.
There are existing, mature e2e testing tools for volume mounts and file ownership verification.

##### e2e tests

Extend the existing volume end-to-end tests.

Create a `agnhost` test pod with the volume definition under test. Make use of `mounttest` utility to verify
ownership of the files.

__SIG Storage__

- [ConfigMap](https://github.com/kubernetes/kubernetes/blob/v1.36.1/test/e2e/common/storage/configmap_volume.go): [SIG Storage](https://testgrid.k8s.io/sig-storage-kubernetes#kind-master-alpha-beta&include-filter-by-regex=ConfigMap), [triage search](https://storage.googleapis.com/k8s-triage/index.html?sig=storage&test=ConfigMap)
- [Downward API volume](https://github.com/kubernetes/kubernetes/blob/v1.36.1/test/e2e/common/storage/downwardapi_volume.go): [SIG Storage](https://testgrid.k8s.io/sig-storage-kubernetes#kind-master-alpha-beta&include-filter-by-regex=Downward%20API%20volume), [triage search](https://testgrid.k8s.io/sig-storage-kubernetes#kind-master-alpha-beta&include-filter-by-regex=ConfigMap)
- [Projected configMap](https://github.com/kubernetes/kubernetes/blob/v1.36.1/test/e2e/common/storage/projected_configmap.go): [SIG Storage](https://testgrid.k8s.io/sig-storage-kubernetes#kind-master-alpha-beta&include-filter-by-regex=Projected%20configMap), [triage search](https://storage.googleapis.com/k8s-triage/index.html?sig=storage&test=Projected%20configMap)
- [Projected downwardAPI](https://github.com/kubernetes/kubernetes/blob/v1.36.1/test/e2e/common/storage/projected_downwardapi.go): [SIG Storage](https://testgrid.k8s.io/sig-storage-kubernetes#kind-master-alpha-beta&include-filter-by-regex=Projected%20downwardAPI), [triage search](https://storage.googleapis.com/k8s-triage/index.html?sig=storage&test=Projected%20downwardAPI)
- [Projected secret](https://github.com/kubernetes/kubernetes/blob/v1.36.1/test/e2e/common/storage/projected_secret.go): [SIG Storage](https://testgrid.k8s.io/sig-storage-kubernetes#kind-master-alpha-beta&include-filter-by-regex=Projected%20secret), [triage search](https://storage.googleapis.com/k8s-triage/index.html?sig=storage&test=Projected%20secret)
- [Secrets](https://github.com/kubernetes/kubernetes/blob/v1.36.1/test/e2e/common/storage/secrets_volume.go): [SIG Storage](https://testgrid.k8s.io/sig-storage-kubernetes#kind-master-alpha-beta&include-filter-by-regex=Secrets), [triage search](https://storage.googleapis.com/k8s-triage/index.html?sig=storage&test=Secrets)

__SIG Auth__

- [ServiceAccounts](https://github.com/kubernetes/kubernetes/blob/v1.36.1/test/e2e/auth/service_accounts.go): SIG Auth, [triage search](https://storage.googleapis.com/k8s-triage/index.html?sig=auth&test=ServiceAccounts)
- [ClusterTrustBundle](https://github.com/kubernetes/kubernetes/blob/v1.36.1/test/e2e/auth/projected_clustertrustbundle.go): SIG Auth, [triage search](https://storage.googleapis.com/k8s-triage/index.html?sig=auth&test=ClusterTrustBundle)
- [Projected PodCertificate](https://github.com/kubernetes/kubernetes/blob/v1.36.1/test/e2e/auth/projected_podcertificate.go): SIG Auth, [triage search](https://storage.googleapis.com/k8s-triage/index.html?sig=auth&test=Projected%20PodCertificate)

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

TBD

#### GA

TBD

### Upgrade / Downgrade Strategy

__Upgrade__

No changes in the cluster upgrade process.

__Downgrade__

1. If the feature flag has never been enabled, no impacts.

2. If the target Kubernetes version supports the feature, no impacts.

3. If the feature flag was ever enabled, and the target Kubernetes version does not support the feature.

    - Cluster operators should audit the `defaultUser` and `user` fields usages before downgrading.
      - If there are no usages of the new fields in any pods, no impacts.
      - If there are usages of the fields.
        - Existing pods temporarily maintain their configured volume data file owners. However, the data
        files will be rewritten and the file owner UIDs will be reverted to legacy defaults when the volume
        data is updated, the pod is recreated, or the pod is rescheduled onto another node.
        There are no events or warnings.
        - The unrecognized fields are silently dropped from API responses by the apiserver, although being
        persisted in the existing etcd pod specs.

### Version Skew Strategy

1. New kubelet, old apiserver

The user fields are not known to apiserver and dropped. The pod spec will be persisted
without those fields.

2. New apiserver, old kubelet

apiserver admits the pods configured with the new fields. kubelet ignores the fields and configures volume
data file owner without considering them.

After a kubelet upgrade, since the data file has already been configured, the file owners will not be updated.
The correct file ownership will only be applied only after a data file is updated, the pod is recreated, or
the pod is rescheduled onto another node.

Therefore, it is recommended to upgrade all the nodes before turning on the feature or using the new fields.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: AtomicWriteVolumeUserFields
  - Components depending on the feature gate:
    - kube-apiserver
    - kubelet

__Use with projected pod certificates__

Pod certificate request is currently in beta and disabled by default.

Therefore, using the new user fields with projected pod certificates also require turning on the
`PodCertificateRequest` feature gate and the `certificates.k8s.io/v1beta1/podcertificaterequests` API.

__Use with projected cluster trust bundles__

Cluster trust bundle is currently in beta and disabled by default.

Therefore, using the new user fields with projected cluster trust bundles also require turning on the
`ClusterTrustBundle` feature gate, the `ClusterTrustBundleProjection` feature gate and the
`certificates.k8s.io/v1beta1/podcertificaterequests` API.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

__Existing usages__

Existing data files in atomic write volumes that were previously configured with the user fields
will retain their assigned file owner.

However, any new or updated data files written by kubelet will ignore these user fields.

If a pod is recreated or rescheduled onto a different node, the data files will be recreated and therefore
kubelet will determine their file owners without referring to the user fields.

__New usages__

apiserver will drop the user fields from newly created pods, as well as from updates to existing pods
that did not already have them defined.

kubelet will ignore these fields when determining file ownership of the volume data files.

###### What happens if we reenable the feature if it was previously rolled back?

__Existing usages__

Existing data files in atomic write volumes that were previously configured with the user fields
will retain their assigned file owner.

However, any new or updated data files written by kubelet will reflect the ownership according to
the user fields.

If a pod is recreated or rescheduled onto a different node, the data files will be recreated and therefore
reflect the ownership according to the user fields.

__New usages__

apiserver will accept the user fields from newly created pods and from updates to existing pods.

kubelet will make use of the user fields to determine file ownership of the volume data files.

###### Are there any tests for feature enablement/disablement?

Yes. Unit testing of user fields retention/dropping based on feature gate enablement/disablement.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

apiserver with the feature enabled will admit pods containing the new user fields.

However, those pods may subsequently be scheduled to nodes where the feature is disabled or unsupported.
This will result in a silent issue, as the volume file ownership will not be properly configured.

To prevent such issues, it is recommended to upgrade all node kubelets before enabling the feature gate.

###### What specific metrics should inform a rollback?

An abnormal increase in the baseline `storage_operation_duration_seconds` metric for the supported volume
types indicates that the feature is degrading storage performance and a rollback is advised.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Manual testing for upgrade/rollback will be done prior to Beta. Steps taken for manual tests will be updated here.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

TBD

###### How can someone using this feature know that it is working for their instance?

TBD

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
  - Components exposing the metric: kubelet

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

The added complexity is minimal, since the feature is able to reuse the existing `FileProjection.FsUser`
mechanism introduced by [KEP-1205][kep-1205-file-permission].

[kep-1205-file-permission]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-auth/1205-bound-service-account-tokens#file-permission

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

This proposal introduces no changes to the existing behavior.

* Existing volume data files remain intact and unaffected in the file system.
* New volume specs or data file updates cannot propagate to kubelet and will not result in file changes.

###### What are other known failure modes?

TBD

###### What steps should be taken if SLOs are not being met to determine the problem?

TBD

## Implementation History

* 2026-02-24 Initial proposal to SIG
* 2026-02-27 Initial KEP draft

## Drawbacks

1. This design does not support persistent volumes

    Persistent volumes may be shared across multiple pods and have an independent lifecycle.

    A design that covers both ephemeral and persistent volumes would introduces significant complexity
    regarding user namespaces, varying underlying storage capabilities and diverse CSI drivers implementation.

    Therefore, a simpler design that focus on ephemeral volumes is more feasible and maintainable.

2. This design does not support all ephemeral volume types

    Unlike atomic write volumes, where individual data files are managed by kubelet, `emptyDir`
    provides an empty directory at the mount point. Therefore, any ownership configurations involve the
    mount point directory only.

    Other ephemeral volume types, i.e. image, CSI ephemeral and general ephemeral, each present different
    challenges and variances. These distinctions are significant enough to warrant a separate proposal.

## Alternatives

1. An idea of a `podSecurityContext.fsUser` has been proposed. However, I believe this KEP is preferrable
because of the following reasons.

    1. `podSecurityContext.fsUser` is a pod level construct.
    
       It doesn't support running multiple containers as different users and requiring different file owners
       per volumes or files.

    2. `podSecurityContext.fsUser` is a pod level construct and implies supporting all the volume types.

       This may also imply passing `fsUser` to CSI drivers, since Kubernetes is
       [currently passing][csi-fsgroup] `fsGroup` to CSI drivers.

    3. Its interaction with `fsGroupChangePolicy` is problematic.
    
       For example, users may reasonably expect
       `fsUser` to follow the same behavior of `fsGroupChangePolicy`. Adding `fsUser` may also entail a new
       `fsUserChangePolicy` feature.

2. Extend the current `runAsUser` heuristic of projected `serviceAccountToken`, `clusterTrustBundle` and `podCertificate`.

    I.e. if all the pods have the same `runAsUser`, use that as the owner UID of the files created.
    There are some potential drawbacks.

    1. Breaking change of the current default behavior.
    
       The files are currently owned by root. Extending the heuristic changes the owner UID to `runAsUser`.

       We can make this an opt-in feature by introducing an optional
       `podSecurityContext.setAtomicVolumeOwnerFromRunAsUser`.

       If we implement this for other volume types in the future, we will also need one such field for
       each volume type.

    2. Do not support containers having different `runAsUser`.

       The current implemetation requires all containers having the same `runAsUser`.

       We can potentially extend it to look up the container `securityContext.runAsUser` and
       apply chown to volumes that is only mounted to one container.

       However, this is likely more complicated to implement and maintain. It is also harder for the users
       to reason about.

    3. It may be confusing that some volume types follow this heuristic but some don't.

       Users may reasonably assume that other volume types follow the same heuristic. User fields are
       clear and explicit.

[csi-fsgroup]: https://kubernetes.io/blog/2022/12/23/kubernetes-12-06-fsgroup-on-mount/

## Infrastructure Needed (Optional)

No.
