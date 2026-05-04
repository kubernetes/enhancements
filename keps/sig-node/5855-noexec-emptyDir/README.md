# KEP-5855: Allow mountOptions (noexec, nodev, nosuid) to be specified on emptyDir volumes

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
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Implementation mechanism](#implementation-mechanism)
  - [Cluster-wide Enforcement](#cluster-wide-enforcement)
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
    - [ACL](#acl)
    - [SELinux](#selinux)
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

This KEP introduces a `mountOptions` field on `emptyDir` volumes so users can apply security-related mount flags such as `noexec`, `nodev`, and `nosuid`. Today, emptyDir volumes use default mount behavior: they are writable and allow execution, setuid/setgid, and device access, which can undermine security for containers that rely on read-only root filesystems or need to meet hardening standards. By adding configurable (opt-in) `mountOptions` to the pod spec, users can restrict emptyDir to be writable for data only (e.g. `noexec`), block setuid/setgid binaries (`nosuid`), and prevent use of device files (`nodev`), in any combination, in a native and declarative way.

## Motivation

The primary goal of this KEP is to increase the security of Kubernetes workloads by allowing security-related mount options on `emptyDir` volumes. Currently, `emptyDir` volumes are mounted with default permissions that allow execution, setuid/setgid, and device access. While many applications require writable scratch space for logs, buffers, or temporary files, these defaults can undermine security-for example, with `noexec` missing, an attacker can use the writable `emptyDir` to download, `chmod +x`, and execute arbitrary binaries even when the container has a read-only root filesystem (`readOnlyRootFilesystem: true`). Supporting `noexec`, `nodev`, and `nosuid` gives users a native way to harden emptyDir mounts to match security benchmarks and policy.

The following security issues and audits have identified this as a critical gap in the Kubernetes volume subsystem:

-   [Issue #48912](https://github.com/kubernetes/kubernetes/issues/48912):
    Architectural Gap for `emptyDir`. Essentially, this issue confirms that the lack of `mountOptions` for `emptyDir` is a recognized security gap, but despite being flagged in an audit, it remains unresolved as of early 2026.

-   [Issue #119627](https://github.com/kubernetes/kubernetes/issues/119627):
    Kubernetes 1.24 Security Audit (Finding NCC-E003660-7HM). This issue tracks a formal vulnerability identified during an external audit. The auditors specifically noted that the inability to mount `emptyDir` with `noexec` represents a failure in security.

Implementing configurable mount options (`noexec`, `nodev`, `nosuid`) directly within the `emptyDir` volume source provides a flexible security enhancement: users can lock down temporary scratch space (e.g. no execution, no setuid, no devices) as needed.

### Goals

Add a `mountOptions` field to the `emptyDir` volume type in the pod `spec`, supporting the `noexec`, `nodev`, and `nosuid` flags.

### Non-Goals

This KEP does not aim to support the `stickyBit` on `emptyDir` volumes in this implementation.

[Add `stickyBit` support for `emptydir` kubernetes #130277](https://github.com/kubernetes/kubernetes/pull/130277)

Extending the `noexec` mount option to volume types beyond `emptyDir` is beyond the scope of this KEP.
Supporting this feature on platforms that don't support Unix-style mount flags for execution control is out of scope.

## Proposal

A mechanism within the pod specification to apply mount options (e.g. `noexec`, `nodev`, `nosuid`) to `emptyDir` volumes. Users can declare in the pod spec that a temporary volume should not allow execution of binaries (`noexec`), should ignore setuid/setgid bits (`nosuid`), or should not allow device files (`nodev`), in any combination, so that emptyDir remains suitable for scratch or temp data while meeting security requirements.

### User Stories

#### Story 1

As a Kubernetes application developer, I want to run my applications with a read-only root filesystem. My application needs a writable directory for scratch data (e.g. `/tmp`), so I use an `emptyDir`. By default that volume is executable and allows setuid and device access, so it can be abused to run malicious binaries or setuid programs. I want to set `mountOptions` such as `noexec`, `nosuid`, and `nodev` on this `emptyDir` so the only writable area in my container cannot be used to execute code or escalate privileges.

#### Story 2

As a cluster administrator, I need workloads to meet CIS Kubernetes Benchmarks or NIST 800-190, which require writable partitions to be mounted with hardening options (e.g. `noexec`, `nosuid`, `nodev`). Today I have to rely on custom node setup or workarounds to get these flags on `emptyDir`. I want a native way to specify `mountOptions` on `emptyDir` in the pod spec so teams can pass security audits without node-level changes.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

No significant risks have been identified. Mount options are opt-in and use existing kernel mount APIs.

## Design Details

Currently `emptyDir` supports 2 options `medium` and `sizeLimit`:

```yaml
volumes:
  - name: shared-data
    emptyDir:
      medium: Memory # Optional: use RAM for high-speed access
      sizeLimit: 100Mi # Optional: limit the size to 100 MiB
```
This design adds a `mountOptions` array (e.g. `noexec`, `nodev`, `nosuid`):

```yaml
    emptyDir:
      medium: Memory # Optional: use RAM for high-speed access
      sizeLimit: 100Mi # Optional: limit the size to 100 MiB
      mountOptions:
        - noexec
        - nodev
        - nosuid
```

This design is in sync with the `stickyBit` work done here: 
[Add `stickyBit` support for `emptydir` kubernetes #130277](https://github.com/kubernetes/kubernetes/pull/130277).

`mountOptions` are strictly validated to contain only explicitly allowed values (in this KEP: `noexec`, `nodev`, `nosuid`). Adding a new permitted value is an API and validation change, therefore each new mount option will require a KEP.

### API Changes

A new optional field `MountOptions` is added to the `EmptyDirVolumeSource` struct:

```go
type EmptyDirVolumeSource struct {
	// medium represents what type of storage medium should back this directory.
	// The default is "" which means to use the node's default medium.
	// Must be an empty string (default) or Memory.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir
	// +optional
	Medium StorageMedium `json:"medium,omitempty" protobuf:"bytes,1,opt,name=medium,casttype=StorageMedium"`
	// sizeLimit is the total amount of local storage required for this EmptyDir volume.
	// The size limit is also applicable for memory medium.
	// The maximum usage on memory medium EmptyDir would be the minimum value between
	// the SizeLimit specified here and the sum of memory limits of all containers in a pod.
	// The default is nil which means that the limit is undefined.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir
	// +optional
	SizeLimit *resource.Quantity `json:"sizeLimit,omitempty" protobuf:"bytes,2,opt,name=sizeLimit"`
	// mountOptions is the list of mount options to apply for this EmptyDir volume. The options noexec, nodev, and nosuid are allowed.
	// +optional
	// +listType=atomic
	MountOptions []string `json:"mountOptions,omitempty" protobuf:"bytes,3,rep,name=mountOptions"`
}
```

The feature is gated behind the `EmptyDirMountOptions` feature gate, following standard patterns for feature-gated API fields (validation gating and field dropping when the gate is disabled).

### Implementation mechanism

The mechanism to apply mount options (e.g. `noexec`, `nodev`, `nosuid`) differs by medium: mount options for memory-backed emptyDir, and a self-bind mount with the requested options for default (disk-backed) emptyDir.

**tmpfs (medium: Memory)**

In the emptyDir volume plugin, the tmpfs mount options are built in `generateTmpfsMountOptions`. The user-specified mount options are appended in this function before mounting.

```go
func (ed *emptyDir) generateTmpfsMountOptions(noswapSupported bool) (options []string) {
    if ed.sizeLimit != nil && ed.sizeLimit.Value() > 0 {
        options = append(options, fmt.Sprintf("size=%d", ed.sizeLimit.Value()))
    }

    if noswapSupported {
        options = append(options, swap.TmpfsNoswapOption)
    }

    options = append(options, ed.mountOptions...)
    return options
}
```

**Default (medium: {})**

For default emptyDir there is no separate filesystem mount, only a directory on the node's existing filesystem is present, so mount options cannot be applied the same way as for tmpfs. To apply mount options, the directory is turned into a mount point by doing a self-bind mount with the requested options (bind the directory to itself, then remount with the options). The same kubelet mounter API used for other volume types is used here.

In the emptyDir plugin's SetUpAt, for disk-backed emptyDir (`StorageMediumDefault`):

```go
case ed.medium == v1.StorageMediumDefault:
    if err = ed.setupDir(dir); err != nil {
        break
    }
    if ed.mounter != nil && len(ed.mountOptions) > 0 {
			notMnt, mountErr := ed.mounter.IsLikelyNotMountPoint(dir)
			if mountErr != nil {
				err = mountErr
				break
			}
			if notMnt {
				opts := append([]string{"bind"}, ed.mountOptions...)
				err = ed.mounter.MountSensitiveWithoutSystemd(dir, dir, "", opts, nil)
				if err != nil {
					klog.ErrorS(err, "emptyDir: bind mount with options failed", "pod", ed.pod.UID, "dir", dir, "options", opts)
				}
			}
		}
```

### Cluster-wide Enforcement

Cluster administrators can enforce `mountOptions` (e.g. requiring `noexec` on all emptyDir volumes) across a cluster or specific namespaces using existing policy tools:

**Validation (rejecting non-compliant pods):**  
`ValidatingAdmissionPolicy` can be used to reject pods that do not include the required mount options. It uses CEL expressions and is a native Kubernetes resource. Third-party tools such as Gatekeeper (OPA) using `ConstraintTemplate` with Rego policies can also reject non-compliant pods.

**Mutation (auto-injecting mountOptions):**  
`MutatingAdmissionWebhook` can auto-inject `mountOptions` using JSON patches. Gatekeeper supports mutation through its `Assign` and `ModifySet` CRDs.

Admission webhooks operate at the API level and can only enforce that the `mountOptions` field is present or valid in the pod spec. Without the kubelet implementation in this KEP, that field would be ignored. Actual enforcement of mount options at the OS/kernel level is done by the kubelet's emptyDir volume plugin, which is what this enhancement implements.

### Test Plan

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

The `emptyDir` mountOptions tests were conducted using both a `local-up-cluster` and KIND (Kubernetes in Docker). Testing on `local-up-cluster` confirmed that mount options are correctly propagated into containers and enforced at the kernel level for both disk-backed and memory-backed emptyDir volumes.

##### Prerequisite testing updates

No prerequisite testing updates are required.

##### Unit tests

The following unit tests have been added:

- **`pkg/volume/emptydir`**
  - `TestPluginDefaultEmptyDirWithMountOptions`: Verifies that mount options are correctly applied as a bind-mount for disk-backed emptyDir volumes.
  - `TestTmpfsMountOptions`: Verifies that mount options are appended to tmpfs mount flags for memory-backed emptyDir volumes.
  - `TestMountOptionsFeatureGate`: Verifies that the emptyDir plugin rejects pods with `mountOptions` when the `EmptyDirMountOptions` feature gate is disabled, and processes them when enabled.
  - `TestTeardownDuringBindMount`: Verifies that teardown correctly handles cleanup when a bind-mount is active (unmounts then removes) and when no bind-mount exists (partial setup, just removes).

- **`pkg/api/pod`**
  - `TestDropEmptyDirMountOptions`: Verifies that `mountOptions` is stripped from pod specs when the feature gate is disabled, and preserved when enabled or when the field is already persisted on an existing pod.

##### Integration tests

Integration tests for emptyDir `mountOptions` will be added.

##### e2e tests

e2e tests for emptyDir `mountOptions` are added in `test/e2e/common/storage/empty_dir.go`, gated behind `framework.WithFeatureGate`(`features.EmptyDirMountOptions`) so they only run in CI jobs that enable alpha features:

- should apply mount options on default medium: Creates a pod with a disk-backed emptyDir volume with `mountOptions: [noexec, nodev, nosuid]`. Verifies the pod is accepted and runs, the `mountOptions` field is preserved in the pod spec after API round-trip, and the volume is mounted and writable.

- should apply mount options on tmpfs medium: Creates a pod with a memory-backed (tmpfs) emptyDir volume with `mountOptions: [noexec, nodev, nosuid]`. Verifies the same conditions as above.

### Graduation Criteria

#### Alpha

- API field implemented and functional
- Unit tests passing
- Documentation available

#### Beta

- No major bugs reported during alpha
- Gather feedback from users

#### GA

- Stable for at least two releases
- No major issues reported

### Upgrade / Downgrade Strategy

Upgrade: Enabling the `EmptyDirMountOptions` feature gate on kubelet and kube-apiserver allows new pods to use `mountOptions`. Existing running pods are unaffected. Mount options are only applied at pod creation time (during volume setup).

Downgrade: Disabling the feature gate causes:
- kube-apiserver: Rejects `mountOptions` on new pods via validation. Strips `mountOptions` from existing pod specs via field dropping (unless already persisted with the field).
- kubelet: Rejects pods with `mountOptions` set if the feature gate is disabled.

Running pods are not affected by downgrade; their volumes are already mounted. Only newly created pods are affected.

### Version Skew Strategy

The `mountOptions` field is optional and additive. In a version-skewed cluster:

- API server newer than kubelet (or feature gate disabled on kubelet): If a pod with `mountOptions` is scheduled to a node where the `EmptyDirMountOptions` feature gate is not enabled, the kubelet will reject the pod with an error. This prevents a silent security gap where a policy engine verifies that `mountOptions` is set, but the kubelet silently ignores it.

- Kubelet newer than API server (or feature gate disabled on API server): The API server does not know about `mountOptions` or strips the field when the gate is disabled. The kubelet never sees the field and behaves as before.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `EmptyDirMountOptions`
  - Components depending on the feature gate: kubelet, kube-apiserver
- [ ] Other
  - Will enabling / disabling the feature require downtime of the control plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning of a node? No

###### Does enabling the feature change any default behavior?

No. Mount options (`noexec`, `nodev`, `nosuid`) are only applied when a user explicitly sets the `mountOptions` field on an `emptyDir` volume. If `mountOptions` is omitted, existing and new emptyDir volumes keep the default behavior (executable, setuid, and device access allowed).

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. The feature is opt-in per pod via the `mountOptions` field. To stop using it for a pod, remove `mountOptions` from the pod spec. To roll back at the cluster level, set the `EmptyDirMountOptions` feature gate to `false` on the kubelet and restart it, and no other changes are required. Disable is supported.

###### What happens if we reenable the feature if it was previously rolled back?

After reenabling the feature gate, newly created pods can use `mountOptions` again. Existing pods keep their current behavior until they are deleted and recreated with the feature gate enabled.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests cover the feature gate: `TestMountOptionsFeatureGate` in `pkg/volume/emptydir` verifies that the emptyDir plugin rejects pods with `mountOptions` when the gate is disabled and applies them when enabled. `TestDropEmptyDirMountOptions` in `pkg/api/pod` verifies that `mountOptions` is stripped from pod specs when the gate is disabled and preserved when enabled or when the field is already persisted.

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

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

No. No new API calls are introduced. The `mountOptions` field is read from the existing pod spec during volume setup. 

###### Will enabling / using this feature result in introducing new API types?

No. A new field (`mountOptions`) is added to the existing `EmptyDirVolumeSource` type. No new API types are introduced.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No. Mount options are applied locally on the node by the kubelet. No cloud provider APIs are involved.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API type: Pod
- Estimated increase in size: Negligible — up to ~30 bytes per emptyDir volume when `mountOptions` is set (e.g. ["noexec", "nodev", "nosuid"]).
- Estimated amount of new objects: None. No new objects are created.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

For disk-backed emptyDir, one additional bind remount syscall is performed during volume setup. For tmpfs, the options are appended to the existing mount command with no extra syscall. Both are expected to be sub-millisecond operations.

Scalability test plan (WIP): To validate the performance impact, we plan to measure pod startup latency with and without `mountOptions` at scale. The approach:
1. Add timing instrumentation (log statements) before and after the volume setup in the emptyDir plugin's `SetUpAt` function to measure the overhead of the additional bind remount.

2. Create multiple pods with emptyDir volumes (with and without mountOptions) and compare pod startup times.

3. Verify that the additional mount operation does not measurably affect pod startup latency at scale.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The feature adds one mount syscall (for disk-backed) or a few extra string options (for tmpfs) per emptyDir volume that uses `mountOptions`. No additional memory, disk, or network usage.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

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

- Adds complexity to the emptyDir plugin: The emptyDir volume plugin gains additional mount logic (bind remount for disk-backed). This increases the surface area for potential mount-related failures during pod startup, though the risk is low since the same mount APIs used by other volume types are reused.

- Linux-only: The noexec, nodev, and nosuid flags are Linux mount options. This feature has no effect on Windows nodes.

## Alternatives

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
