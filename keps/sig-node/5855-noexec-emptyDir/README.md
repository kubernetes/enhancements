# KEP-5855: Allow bind mount options (noexec, nodev, nosuid) on volumeMounts

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
  - [API Changes](#api-changes)
  - [Implementation mechanism](#implementation-mechanism)
  - [CRI changes](#cri-changes)
  - [Container runtime changes (CRI-O, containerd)](#container-runtime-changes-cri-o-containerd)
  - [Cluster-wide Enforcement](#cluster-wide-enforcement)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
      - [Manual validation](#manual-validation)
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
    - [Mount options on EmptyDirVolumeSource](#mount-options-on-emptydirvolumesource)
    - [ACL](#acl)
    - [SELinux](#selinux)
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

This KEP introduces a `bindMountOptions` field on `volumeMounts` in the pod spec so users can apply security-related bind mount flags such as `noexec`, `nodev`, and `nosuid` to any volume type. Today, volume mounts inside containers do not carry `noexec`, `nosuid`, or `nodev` flags by default, which can undermine security for containers that rely on read-only root filesystems or need to meet hardening standards. By adding configurable (opt-in) `bindMountOptions` to volume mounts, users can restrict a mount to be writable for data only (e.g. `noexec`), block setuid/setgid binaries (`nosuid`), and prevent use of device files (`nodev`), in any combination, in a native and declarative way. Because the field is on `volumeMounts`, it applies to all volume types (emptyDir, PersistentVolumes, CSI, projected, etc.) and provides per-container granularity, i.e., different containers in the same pod can mount the same volume with different options.

## Motivation

The primary goal of this KEP is to increase the security of Kubernetes workloads by allowing security-related bind mount options on volume mounts. Currently, volumes are bind-mounted into containers without `noexec`, `nosuid`, or `nodev` flags. This default can undermine security, for example, with `noexec` missing, an attacker can use any writable volume (emptyDir, PersistentVolume, projected, etc.) to download, `chmod +x`, and execute arbitrary binaries even when the container has a read-only root filesystem (`readOnlyRootFilesystem: true`). Supporting `noexec`, `nodev`, and `nosuid` gives users a native way to harden volume mounts to match security benchmarks and policy.

The gap is most visible with emptyDir volumes, which are the most common writable volume type and have been the subject of multiple security findings:

-   [Issue #48912](https://github.com/kubernetes/kubernetes/issues/48912):
    Recognized security gap - the inability to set mount options on `emptyDir` was flagged in an audit but remains unresolved as of 2026.

-   [Issue #119627](https://github.com/kubernetes/kubernetes/issues/119627):
    Kubernetes 1.24 Security Audit (Finding NCC-E003660-7HM) - external auditors specifically noted that the inability to mount `emptyDir` with `noexec` represents a security failure.

However, the same gap applies to all volume types. PersistentVolumes have a `mountOptions` field, but those options are filesystem-level flags applied by the CSI driver at the node - they do not reliably translate into bind mount flags inside the container. There is currently no mechanism to set `noexec`, `nosuid`, or `nodev` on the bind mount that the container runtime creates for any volume type.

### Goals

- Add a `bindMountOptions` field to `VolumeMount` in the pod spec, supporting `noexec`, `nodev`, and `nosuid` flags.
- Pass bind options through the CRI `Mount` message to container runtimes.
- Support all volume types (emptyDir, PersistentVolume, CSI, projected, configMap, secret, etc.).
- Provide per-container granularity: different containers can mount the same volume with different options.

### Non-Goals

- Extending the allowed set of bind options beyond `noexec`, `nodev`, and `nosuid` in this KEP. Adding a new permitted value is an API and validation change and would require a separate KEP.
- Supporting this feature on platforms that don't support Unix-style mount flags.
- Replacing or modifying the existing `mountOptions` field on PersistentVolume specs (which controls filesystem-level mount options applied by CSI drivers at the storage layer).
- Deprecating or replacing the existing `ReadOnly` boolean on `VolumeMount`. While `ro` is conceptually a bind mount option, unifying it into `bindMountOptions` is out of scope for this KEP.

## Proposal

Add a `bindMountOptions` field to the `VolumeMount` struct in the Kubernetes API. When set, the kubelet passes these options through the CRI `Mount` message to the container runtime, which includes them in the OCI runtime spec mount options. The low-level runtime (runc/crun) then applies the flags during the bind mount + remount inside the container, ensuring kernel-level enforcement.

### User Stories

#### Story 1

As a Kubernetes application developer, I want to run my applications with a read-only root filesystem. My application needs writable directories for scratch data (e.g. `/tmp`) and persistent storage, so I use emptyDir and PersistentVolume mounts. By default these volume mounts do not carry `noexec` or `nosuid` flags, so they can be abused to run malicious binaries or setuid programs. I want to set `bindMountOptions: [noexec, nosuid]` on the `volumeMount` so that writable mounts in my container cannot be used to execute code or escalate privileges.

```yaml
volumes:
  - name: tmp
    emptyDir: {}
containers:
  - name: app
    volumeMounts:
      - name: tmp
        mountPath: /tmp
        bindMountOptions: [noexec, nosuid]
```

#### Story 2

As a cluster administrator, I need workloads to meet CIS Kubernetes Benchmarks or NIST 800-190, which require writable partitions to be mounted with hardening options (e.g. `noexec`, `nosuid`, `nodev`). Today I have to rely on custom node setup or workarounds to get these flags on volumes. I want a native way to specify bind options in the pod spec so teams can pass security audits without node-level changes.

#### Story 3

As a platform engineer, I want to mount a shared PersistentVolume into two containers in the same pod: one container needs to write data but should not be able to execute anything from the volume, and the other needs to read and execute scripts from it. I want to set `bindMountOptions: [noexec]` on one container's mount while leaving the other unrestricted, giving each container the appropriate level of access.

```yaml
volumes:
  - name: shared
    persistentVolumeClaim:
      claimName: my-pvc
containers:
  - name: writer
    volumeMounts:
      - name: shared
        mountPath: /data
        bindMountOptions: [noexec]
  - name: runner
    volumeMounts:
      - name: shared
        mountPath: /data
```

### Notes/Constraints/Caveats (Optional)

- The allowed bind options (`noexec`, `nodev`, `nosuid`) correspond to VFS-level bind mount flags that the Linux kernel supports on `MS_BIND | MS_REMOUNT`. Filesystem-specific mount options (e.g. `uid=`, `gid=`, `data=ordered`) are not applicable to bind mounts and are not supported.
- The field is named `bindMountOptions` (rather than `mountOptions`) to avoid confusion with the existing `mountOptions` field on PersistentVolume specs, which controls filesystem-level options applied by CSI drivers. The name `bindMountOptions` reflects that these are bind mount flags applied by the container runtime, and is appropriate since the field is already scoped within `volumeMounts`.

### Risks and Mitigations

- **Runtime compatibility**: Container runtimes (CRI-O, containerd) must be updated to read the new `mount_options` CRI field and pass it to the OCI spec. To prevent silent degradation, the kubelet uses `runtimeFeatures` (advertised by the CRI runtime via the `Status` RPC) to check whether the runtime supports `mount_options`. If a pod uses `bindMountOptions` but the runtime does not advertise support, the kubelet rejects the pod. This is similar to how user namespace support is detected. Additionally: (1) the feature is behind a feature gate so users must explicitly opt in; (2) runtime support for `mount_options` is a beta graduation requirement; (3) cluster administrators can verify enforcement by checking `/proc/self/mountinfo` inside a test container.
- **Interaction with PV mountOptions**: PersistentVolumes already have a `mountOptions` field that controls filesystem-level mount options applied by the CSI driver. The new `volumeMount.bindMountOptions` field controls bind mount flags applied by the container runtime. These operate at different layers and do not conflict. The CSI driver's mount options affect the filesystem mount on the node, while `bindMountOptions` affects the bind mount into the container. In the case of VFS flags like `noexec`, `nosuid`, and `nodev`, the PV `mountOptions` field cannot reliably enforce them inside the container because OCI runtimes (runc, crun) remount bind mounts using only the flags from the OCI spec, stripping any inherited source mount flags. `bindMountOptions` is the reliable mechanism for these flags.

## Design Details

### API Changes

A new optional field `BindMountOptions` is added to the `VolumeMount` struct:

```go
type VolumeMount struct {
	Name      string          `json:"name" protobuf:"bytes,1,opt,name=name"`
	ReadOnly  bool            `json:"readOnly,omitempty" protobuf:"varint,2,opt,name=readOnly"`
	MountPath string          `json:"mountPath" protobuf:"bytes,3,opt,name=mountPath"`
	SubPath   string          `json:"subPath,omitempty" protobuf:"bytes,4,opt,name=subPath"`
	// ...existing fields...

	// bindMountOptions is the list of additional bind mount options to apply when
	// mounting this volume into the container. Allowed values are noexec,
	// nodev, and nosuid.
	// +featureGate=VolumeBindMountOptions
	// +optional
	// +listType=set
	BindMountOptions []string `json:"bindMountOptions,omitempty" protobuf:"bytes,8,rep,name=bindMountOptions"`
}
```

The feature is gated behind the `VolumeBindMountOptions` feature gate. When the gate is disabled:
- **kube-apiserver**: Strips `bindMountOptions` from new pod specs via field dropping (unless the field is already persisted on an existing pod).
- **Validation**: Rejects pods with `bindMountOptions` set to invalid values. Only `noexec`, `nodev`, and `nosuid` are allowed. Duplicates are rejected.

### Implementation mechanism

The implementation spans four components:

**1. Kubelet (`kubelet_pods.go`)**

In `makeMounts()`, the kubelet reads `mount.BindMountOptions` from the `VolumeMount` spec and passes it to the internal `kubecontainer.Mount` struct:

```go
mounts = append(mounts, kubecontainer.Mount{
	Name:          mount.Name,
	ContainerPath: containerPath,
	HostPath:      hostPath,
	// ...existing fields...
	BindMountOptions:   mount.BindMountOptions,
})
```

**2. Kubelet runtime feature detection**

Before processing `bindMountOptions`, the kubelet checks the CRI runtime's `Status` RPC for `runtimeFeatures` to determine whether the runtime advertises support for `mount_options`. If a pod uses `bindMountOptions` but the runtime does not advertise support, the kubelet rejects the pod. This is similar to how user namespace support is detected.

**3. Kubelet runtime (`kuberuntime_container.go`)**

In `makeMounts()`, the kubelet runtime converts `kubecontainer.Mount` to `runtimeapi.Mount` for the CRI call, passing through bind options:

```go
mount := &runtimeapi.Mount{
	HostPath:      v.HostPath,
	ContainerPath: v.ContainerPath,
	// ...existing fields...
	MountOptions:  v.BindMountOptions,
}
```

**4. CRI (`Mount` message)**

A new field `mount_options` (field 11) is added to the CRI `Mount` message:

```protobuf
message Mount {
    string container_path = 1;
    string host_path = 2;
    bool readonly = 3;
    bool selinux_relabel = 4;
    MountPropagation propagation = 5;
    // ...existing fields...

    // mount_options specifies additional bind mount options (e.g., noexec,
    // nodev, nosuid) that the runtime must apply when mounting this volume
    // into the container. These are passed as OCI mount options.
    repeated string mount_options = 11;
}
```

**5. Container runtimes (CRI-O, containerd)**

The runtime reads `mount_options` from the CRI `Mount` message and merges them into the OCI spec mount options, deduplicating and resolving any conflicts with existing defaults. For example, if the default options include `exec` but the CRI request specifies `noexec`, the runtime must remove `exec` before adding `noexec` to ensure a consistent option set:

```go
// CRI-O: server/container_create_linux.go
// Remove conflicting options (e.g., remove "exec" when "noexec" is requested)
options = resolveConflicts(options, m.GetMountOptions())
options = append(options, m.GetMountOptions()...)

ociMounts = append(ociMounts, rspec.Mount{
	Source:      src,
	Destination: dest,
	Options:     options,  // now includes noexec, nodev, nosuid
})
```

At the OCI level, runc/crun converts the options list into mount flags (e.g., `noexec` → `MS_NOEXEC`) and applies them during the bind remount.

### CRI changes

The CRI `Mount` message in `api.proto` is extended with a new `mount_options` field (field number 11). This is a backward-compatible addition - older runtimes that do not recognize field 11 will skip it (standard protobuf behavior for unknown fields).

### Container runtime changes (CRI-O, containerd)

Both CRI-O and containerd need to be updated to:
1. Read the `mount_options` field from the CRI `Mount` message.
2. Merge the options into the OCI runtime spec `Mount.Options` array, deduplicating and resolving conflicts with existing default options (e.g., removing `exec` when `noexec` is requested, removing `suid` when `nosuid` is requested, and removing `dev` when `nodev` is requested). The CRI-specified options take precedence over defaults.
3. Advertise `mount_options` support in `runtimeFeatures` (returned via the CRI `Status` RPC), so the kubelet can detect whether the runtime supports this feature.

No validation is needed in the runtime - the kubelet validates the allowed options before sending them through CRI.

### Cluster-wide Enforcement

Cluster administrators can enforce `bindMountOptions` (e.g. requiring `noexec` on all volume mounts) across a cluster or specific namespaces using existing policy tools:

**Validation (rejecting non-compliant pods):**
`ValidatingAdmissionPolicy` can be used to reject pods that do not include the required bind options. It uses CEL expressions and is a native Kubernetes resource. For example, a policy can require that every container mounting a given volume sets `bindMountOptions: [noexec]`, ensuring consistent enforcement across all containers in a pod. Third-party tools such as Gatekeeper (OPA) using `ConstraintTemplate` with Rego policies can also reject non-compliant pods.

**Mutation (auto-injecting bindMountOptions):**
`MutatingAdmissionWebhook` can auto-inject `bindMountOptions` using JSON patches. Gatekeeper supports mutation through its `Assign` and `ModifySet` CRDs.

Admission webhooks operate at the API level and can only enforce that the `bindMountOptions` field is present or valid in the pod spec. Actual enforcement of bind options at the OS/kernel level is done by the container runtime via the CRI, which is what this enhancement implements.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No prerequisite testing updates are required.

##### Unit tests

The following unit tests have been added:

- **`pkg/apis/core/validation`**
  - Validates that only `noexec`, `nodev`, and `nosuid` are accepted as bind options.
  - Validates that duplicates are rejected.
  - Validates that empty and nil `bindMountOptions` are accepted.

- **`pkg/api/pod`**
  - `TestDropVolumeBindMountOptions`: Verifies that `bindMountOptions` is stripped from pod specs when the feature gate is disabled, and preserved when enabled or when the field is already persisted on an existing pod.

- **`pkg/kubelet/kubelet_pods_test.go`**
  - Verifies that `bindMountOptions` from the `VolumeMount` spec is correctly passed through to the `kubecontainer.Mount` struct.

- **`pkg/kubelet/kuberuntime/kuberuntime_container_test.go`**
  - Verifies that `bindMountOptions` from `kubecontainer.Mount` is correctly passed to the `runtimeapi.Mount` struct for the CRI call.

- **`staging/src/k8s.io/api/core/v1`**
  - API roundtrip tests for serialization/deserialization of `bindMountOptions`.

Container runtime unit tests (verifying that `mount_options` from the CRI `Mount` message is merged into OCI spec mount options with correct conflict resolution and deduplication) will be added in the respective CRI-O and containerd repositories.

##### Integration tests

No integration tests are planned.

##### e2e tests

`e2e_node` tests for `volumeMount.bindMountOptions` are added, gated behind `framework.WithFeatureGate`(`features.VolumeBindMountOptions`) so they only run in CI jobs that enable alpha features. These are node-level tests because bind option enforcement depends on the container runtime applying mount flags at the kernel level:

- **bind options enforcement**: Creates a pod with a volume mounted with `bindMountOptions: [noexec, nosuid, nodev]`. Verifies that `/proc/self/mountinfo` shows the flags and that executing a script on the mount fails with "Permission denied". Tested for both disk-backed and tmpfs (`medium: Memory`) emptyDir, and for PersistentVolumes.

- **per-container granularity**: Creates a pod with two containers mounting the same volume - one with `bindMountOptions: [noexec]` and one without. Verifies that execution is blocked in the first container and allowed in the second.

- **control test**: Verifies that a volume mounted without `bindMountOptions` allows normal execution.

##### Manual validation

End-to-end validation was performed on an OpenShift GCP cluster (v1.35) with custom-built kubelet and CRI-O binaries. The full pipeline - kubelet to CRI protobuf (`mount_options`, field 11) to CRI-O to runc to kernel - was confirmed working:

- CRI-O logs showed `mount_options=[noexec]` received from kubelet and included in OCI spec options.
- `/proc/self/mountinfo` inside the container showed the `noexec` flag on the volume mount.
- Executing a script on the mount failed with "Permission denied"; a control mount (`/tmp`) allowed execution.
- Validated on both emptyDir and GCP PersistentDisk (CSI) volumes.

### Graduation Criteria

#### Alpha

- Feature implemented behind `VolumeBindMountOptions` feature gate (disabled by default)
- Kubelet passes `bindMountOptions` through CRI `Mount.mount_options` to the container runtime
- Unit tests and initial `e2e_node` tests completed

#### Beta

- Feature enabled by default
- CRI-O and containerd implement `mount_options` support and advertise via `runtimeFeatures`
- E2E testing for both containerd and CRI-O
- Downgrade and upgrade testing completed
- Address feedback and bugs reported during Alpha

#### GA

- At least 2 releases in Beta without major bugs
- Remove feature gate
- No negative user feedback from production usage
- CRI-O and containerd have shipped stable support for `mount_options`

### Upgrade / Downgrade Strategy

**Upgrade**: Enabling the `VolumeBindMountOptions` feature gate on kubelet and kube-apiserver allows new pods to use `bindMountOptions` on volume mounts. Existing running pods are unaffected. Bind options are only applied at container creation time.

**Downgrade**: Disabling the feature gate causes:
- **kube-apiserver**: Strips `bindMountOptions` from new pod specs via field dropping (unless already persisted with the field).
- **kubelet**: Rejects pods that have `bindMountOptions` set. In practice, if the API server gate is also disabled, the field is already stripped before reaching the kubelet.

Running pods are not affected by downgrade; their volumes are already mounted. Only newly created pods are affected.

### Version Skew Strategy

**New Apiserver, older kubelet:** The apiserver accepts pods with `bindMountOptions`. The older kubelet does not have the `VolumeBindMountOptions` code - it does not recognize the field. If such a pod lands on this node, the kubelet silently ignores `bindMountOptions` and mounts volumes with default options. The Node Declared Features framework prevents this: the scheduler will not place pods with `bindMountOptions` on nodes that do not declare `VolumeBindMountOptions`, so the pod will only be scheduled to nodes with the feature enabled and a compatible runtime. If the pod bypasses the scheduler (e.g., static pod, custom scheduler), the options are silently ignored.

**Old Apiserver, newer kubelet:** The apiserver does not recognize `bindMountOptions` and strips it as an unknown field. The kubelet never sees the field and behaves as before.

**Apiserver ON, kubelet OFF (gate disabled):** The apiserver accepts `bindMountOptions`. The kubelet does not declare `VolumeBindMountOptions` in `node.status.declaredFeatures`, so the scheduler avoids placing the pod on this node. If the pod is somehow scheduled, the kubelet rejects it with an error, preventing silent degradation.

**Apiserver OFF, kubelet ON:** The apiserver strips the field when the gate is disabled. The kubelet never sees it and behaves as before.

**Both ON, runtime does not support `mount_options`:** The kubelet detects via `runtimeFeatures` that the runtime does not advertise `mount_options` support. The kubelet does not declare `VolumeBindMountOptions` in `node.status.declaredFeatures`, so the scheduler avoids placing pods with `bindMountOptions` on this node. If the pod is somehow scheduled, the kubelet rejects it.

**Both ON, runtime supports `mount_options`:** Full enforcement. The kubelet declares `VolumeBindMountOptions` in `node.status.declaredFeatures`, the scheduler places pods on compatible nodes, and the kubelet passes `bindMountOptions` through CRI to the runtime.

**Both OFF:** Feature disabled, existing behavior.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `VolumeBindMountOptions`
  - Components depending on the feature gate: kubelet, kube-apiserver

###### Does enabling the feature change any default behavior?

No. Bind options (`noexec`, `nodev`, `nosuid`) are only applied when a user explicitly sets the `bindMountOptions` field on a `volumeMount`. If `bindMountOptions` is omitted, all volume mounts keep their default behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. The feature is opt-in per volumeMount via the `bindMountOptions` field. To stop using it for a pod, remove `bindMountOptions` from the volumeMount. To roll back at the cluster level, set the `VolumeBindMountOptions` feature gate to `false` on the kubelet and kube-apiserver - the API server will strip the field from new pods, and the kubelet will reject any pods that still have it set. Disable is supported.

###### What happens if we reenable the feature if it was previously rolled back?

After reenabling the feature gate, newly created pods can use `bindMountOptions` again. Existing pods keep their current behavior until they are deleted and recreated with the feature gate enabled.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests cover the feature gate: `TestDropVolumeBindMountOptions` in `pkg/api/pod` verifies that `bindMountOptions` is stripped from pod specs when the gate is disabled and preserved when enabled or when the field is already persisted.

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

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

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
- [ ] Other (treat as last resort)
  - Details:

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

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

- Container runtime with CRI `mount_options` support
  - Usage description: The container runtime (CRI-O or containerd) must support the `mount_options` field (field 11) in the CRI `Mount` message and advertise it via `runtimeFeatures` in the `Status` RPC. If the runtime does not advertise `mount_options` support, the kubelet rejects pods that use `bindMountOptions`, preventing silent degradation.
    - Impact of its outage on the feature: Pods with `bindMountOptions` are rejected by the kubelet, ensuring users get a clear error instead of silently missing security flags.
    - Impact of its degraded performance or high-error rates on the feature: No impact - bind options are a simple field addition with negligible overhead.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

No. No new API calls are introduced. The `bindMountOptions` field is read from the existing pod spec during container creation and passed through the existing CRI `CreateContainer` call.

###### Will enabling / using this feature result in introducing new API types?

No. A new field (`bindMountOptions`) is added to the existing `VolumeMount` type. No new API types are introduced.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No. Bind options are applied locally by the container runtime during bind mount processing. No cloud provider APIs are involved.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API type: Pod
- Estimated increase in size: Negligible — up to ~30 bytes per volumeMount when `bindMountOptions` is set (e.g. `["noexec", "nodev", "nosuid"]`).
- Estimated amount of new objects: None. No new objects are created.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No measurable impact. Bind options are an additional string array field passed through CRI and appended to the OCI mount options. The container runtime already performs a bind + remount for every mount; the only difference is that additional flags are included in the remount syscall.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The feature adds a few string values to the CRI message and OCI spec per mount. No additional syscalls, memory, disk, or network usage.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

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

- 2026-01-30: KEP created

## Drawbacks

- **Multi-project coordination**: This requires changes to the CRI proto, the kubelet, and container runtimes (CRI-O, containerd). However, the changes in each component are small and well-contained.

- **Older runtimes without `mount_options` support**: If the container runtime does not advertise `mount_options` support via `runtimeFeatures`, the kubelet rejects pods that use `bindMountOptions`. This prevents silent degradation but means the feature is unavailable until the runtime is updated.

- **Linux-only**: The `noexec`, `nodev`, and `nosuid` flags are Linux mount options. This feature has no effect on Windows nodes.

## Alternatives

#### Mount options on EmptyDirVolumeSource

An alternative API placement is adding mount options to `EmptyDirVolumeSource` rather than `VolumeMount`. This was not chosen because:

- **emptyDir-only**: Does not address the same security need for PersistentVolumes, CSI volumes, configMaps, secrets, etc.
- **No per-container granularity**: All containers sharing the volume get the same options. Placing options on `VolumeMount` allows different containers to mount the same volume with different flags.

#### ACL

Using ACLs would allow per-user or per-group control (e.g. allow/deny execute on specific files or directories). ACLs are applied at the file or directory level, not to the whole filesystem. We did not choose this because:

- Control is per file/directory and per user/group, not filesystem-wide.
- The root user can still execute any binary because ACLs do not restrict root in the same way a mount option does.

#### SELinux

Using SELinux could restrict execution based on context. We did not choose this because:

- Root can still execute binaries when SELinux policy allows it and we need execution disabled at the mount level for the whole volume.
- Behavior depends on the host's SELinux configuration and policy, which is not uniform across environments.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
