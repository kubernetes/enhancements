# KEP-5758: Per-container ulimits configuration

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1 (Optional)](#story-1-optional)
    - [Story 2 (Optional)](#story-2-optional)
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

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes adding a `ulimits` field to the Container `SecurityContext` in the Pod API. 
This allows users to configure POSIX resource limits (rlimits) such as `nofile` (open files), 
`nproc` (process count), and `memlock` on a per-container basis.

## Motivation

Currently, Users cannot configure ulimits for individual containers. 
Ulimits are typically inherited from the container runtime (e.g., crio, containerd), 
which in turn inherit them from the host operating system or systemd configuration.

When running workloads with specific resource requirements, fine-grained control over ulimits is often necessary to optimize performance, ensure stability, or meet application needs.
For instance, databases such as Elasticsearch or high-concurrency servers may require higher limits for open files or processes than the runtime defaults.
Without native support, users are forced to resort to workarounds such as modifying host ulimits, setting ulimits through additional entrypoint scripts, or patching container runtime configurations—all of which lack portability, security, and scalability in multi-tenant clusters.

### Goals

* Introduce a new ulimits field in `Container.SecurityContext` for specifying per-container ulimits.
* Add corresponding fields in the CRI API to propagate ulimits to container runtimes.
* Add a runtime handler to detect container runtime support for ulimits, enabling integration with node declared features.

### Non-Goals

* Support for Windows containers (ulimits are Linux-specific).
* cgroup-level enforcement of ulimits; this remains PID-level only.

## Proposal

We propose adding a `Ulimits` list to the `SecurityContext`. The Kubelet will validate these requests and pass them to the runtime via the CRI `LinuxContainerSecurityContext`.
The runtime is responsible for applying these limits to the container process.

### User Stories (Optional)

#### Story 1 (Optional)

As a database operator running Elasticsearch in Kubernetes, 
I need to set a high nofile ulimit (e.g., soft: 65535, hard: 65535) 
per container to handle thousands of concurrent file operations without hitting "too many open files" errors, ensuring cluster stability.

#### Story 2 (Optional)

As a developer running compute-intensive workloads,
I want to configure nproc ulimits (e.g., soft: 1024, hard: 2048) to limit the number of processes a container can spawn,
preventing fork bombs or resource exhaustion in shared clusters.

### Notes/Constraints/Caveats (Optional)

* Ulimits are applied per-PID and do not integrate with cgroups, so they may not prevent all forms of resource abuse.
* Integration with node-declared features allows nodes to advertise ulimit support dynamically.

### Risks and Mitigations

**Risks:**

1. File Descriptor Exhaustion (nofile)
   - Risk: Each container can request up to 1,048,576 file descriptors (FDs). Many containers with high nofile values can exhaust the kernel FD table.
   - Symptom: Node-wide "too many open files" errors, potentially affecting system services.
   - Reference: As discussed in moby/moby#45534, setting RLIMIT_NOFILE to effectively infinite can lead to excessive memory consumption or crashes.

2. Process Table Exhaustion (nproc)
   - Risk: The nproc limit is applied per-UID on the host, not per-container. User namespaces do not affect this behavior, as the nproc limit is enforced based on the host UID. For example, 10 containers running as UID 1000 with nproc=1000 share a single pool of 1000 processes.
   - Symptom: Fork failures may occur even if individual containers are under their specified limits.
   - Reference: For more details, see [LWN article on user namespaces and resource limits](https://lwn.net/Articles/842842/).

3. Memory Exhaustion (memlock)
   - Risk: High memlock values can exhaust physical RAM.
   - Symptom: Out-of-memory (OOM) kills and node instability.

**Mitigations:**

1. Validation and Limits:
   - Validation caps nofile at 1,048,576 to prevent excessive file descriptor usage in the Pod API validation layer.

2. Documentation and Warnings:
   - Documentation includes warnings against setting unnecessarily high values for ulimits.


## Design Details

**API Changes**

We introduce the `Ulimit` struct in core v1 and add it to `SecurityContext`:
```
type SecurityContext struct {
  ......
// The ulimits to be applied to the container.
// Each element in this list maps to a system ulimit setting.
// If unspecified, the container runtime default ulimits will be used.
// Note that this field cannot be set when spec.os.name is windows.
// +optional
// +listType=map
// +listMapKey=name
Ulimits []Ulimit `json:"ulimits,omitempty" protobuf:"bytes,13,rep,name=ulimits"`
}

// Ulimit corresponds to a ulimit setting on a Linux system.
type Ulimit struct {
	// Name of the ulimit to be set.
	// Must be one of the supported ulimit names (e.g., "nofile", "nproc", "core").
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Hard is the hard limit for the ulimit type.
	// The hard limit acts as a ceiling for the soft limit.
	// An unprivileged process may only set its soft limit to a value between 0 and the hard limit
	// and (irreversibly) lower its hard limit.
	Hard int64 `json:"hard" protobuf:"varint,2,opt,name=hard"`

	// Soft is the soft limit for the ulimit type.
	// The soft limit is the value that the kernel enforces for the corresponding resource.
	// The soft limit can be increased in the process up to the hard limit value.
	Soft int64 `json:"soft" protobuf:"varint,3,opt,name=soft"`
}

```

**CRI Changes**

We update the `LinuxContainerSecurityContext` in the CRI API:
```
type LinuxContainerSecurityContext struct {
  .......
// Ulimits specifies the ulimits for the container.
// Each element maps to a system ulimit setting.
Ulimits       []*Ulimit `protobuf:"bytes,18,rep,name=ulimits,proto3" json:"ulimits,omitempty"`
}

// Ulimit corresponds to a ulimit setting on a Linux system.
message Ulimit {
    // Name of the ulimit to be set.
    // e.g. "nofile", "nproc", "core".
    string name = 1;

    // Hard is the hard limit for the ulimit type.
    int64 hard = 2;

    // Soft is the soft limit for the ulimit type.
    int64 soft = 3;
}
```

**Validation Logic:**

We will enforce a strict allowlist of ulimit names in the API validation layer to prevent passing arbitrary or unsafe strings to the kernel. Only the following ulimit options will be supported:

```
var ulimitNameMapping = sets.New(
  "nofile",
  "memlock",
  "core",
  "nice",
  "rtprio",
  "stack",
)
```

Any other ulimit options will result in a Pod creation rejection.

**Future Considerations:**

We may consider relaxing these restrictions in the beta or GA stages of this feature, based on user feedback and demand. This will allow us to evaluate additional use cases and ensure the feature meets the needs of a broader range of workloads.

Additional Rules:

- Soft limit must be less than or equal to Hard limit.

- In most modern Linux distributions, the maximum allowed value for `RLIMIT_NOFILE` is 1048576. When attempting to set a limit higher than this global threshold, the setrlimit system call will return an `EPERM` (Operation not permitted) error, even for privileged processes. Therefore, we will also add validation to ensure that the value of nofile does not exceed 1048576.

- If the `PodSpec.OS.Name` field of a Pod is set to "windows", we will reject the creation of that pod.

**Kubelet and Runtime Implementation:**

The following describes the behavior of the system when a user specifies -1 for ulimit values in the Pod specification:

1. User specifies: {name: "nofile", soft: -1, hard: -1} in the Pod spec.

2. Kubelet to CRI:
   - The Kubelet passes the -1 value unchanged in `LinuxContainerSecurityContext.ulimits[].soft/hard` to the CRI.

3. Container Runtime (CRI-O/containerd):
   - The container runtime receives the -1 value from the Kubelet.
   - The runtime converts the -1 value to the maximum allowable value for the specific resource by mapping it to the Linux kernel's `RLIM_INFINITY` constant. This is achieved by converting the signed integer -1 to an unsigned 64-bit integer, which corresponds to the maximum value (`^uint64(0)`).
   - For most ulimit types, setting the value to `RLIM_INFINITY` results in the resource being displayed as `unlimited` when queried inside the container (e.g., using `ulimit -a`).
   - For specific ulimit types like `nofile`, the kernel enforces a maximum value (e.g., 1048576 for `nofile`), which will be reflected in the container.
   - For any other negative values (other than -1), the runtime caps them to 0 to prevent invalid configurations.
   - The converted value is then passed to the OCI runtime spec.

4. OCI Runtime (runc/crun):
   - The OCI runtime receives the converted value from the container runtime.
   - The OCI runtime calls the setrlimit(2) syscall with the converted value.

5. Linux Kernel:
   - For nofile: It receives the concrete value passed by the runtime. The kernel does not automatically map RLIM_INFINITY down to nr_open; instead, it relies on the runtime to have clamped the value beforehand to avoid an EPERM error.
   - For nproc: The kernel compares the value against `/proc/sys/kernel/pid_max` during fork/exec, but technically accepts RLIM_INFINITY as a setting.
   - For other resources: The kernel accepts RLIM_INFINITY and treats the resource as effectively unlimited (up to ULONG_MAX).

 When queried inside the container (e.g., using `ulimit -a`), most resources set to RLIM_INFINITY will be displayed as `unlimited`, except for those with kernel-enforced maximum values like `nofile`.


**Node Declared Features Integration**

To ensure seamless integration with node declared features, the Kubelet must be able to discover if the underlying container runtime supports the `Ulimits` configuration. This prevents scheduling Pods with custom ulimits to nodes that cannot enforce them.

We need to add a new field called `ContainerUlimits` to the `RuntimeFeatures` in CRI.

After integrating node declared features, it determines whether the container runtime supports the `ContainerUlimits` feature, enabling early-stage decision-making during scheduling to avoid scheduling Pods with custom ulimits to nodes that do not support this feature.


**Pod Security Admission Integration**

To ensure compliance with Pod Security Standards (PSS), the following integration will be implemented:

- Only the `Privileged` PSS level may set the `ulimits` field in a Pod specification. Pods created under the `Baseline` or `Restricted` PSS levels must not include the `ulimits` field.
- Any attempt to set `ulimits` when the PSS level is `Baseline` or `Restricted` will result in a Pod creation rejection.

This integration will be enforced in the Pod Security Admission (PSA) controller. The controller will validate the presence and values of the `ulimits` field during the admission process and reject any Pod that violates this rule.

**Mapping Between API Field Names and System Limits**

The following table establishes the correspondence between the Pod API field names, the standard shell ulimit flags, and the underlying Linux kernel constants. A new column has been added to indicate whether the ulimit is supported.

| Pod Spec Name | Kernel Constant | Ulimit Flag | Description | Supported |
| :--- | :--- | :---: | :--- | :---: |
| **`as`** (or `vmem`) | `RLIMIT_AS` | `-v` | Maximum amount of virtual memory available to the process. | No |
| **`core`** | `RLIMIT_CORE` | `-c` | Maximum size of core files created. | Yes |
| **`cpu`** | `RLIMIT_CPU` | `-t` | CPU time limit in seconds. | No |
| **`data`** | `RLIMIT_DATA` | `-d` | Maximum size of a process's data segment. | No |
| **`fsize`** | `RLIMIT_FSIZE` | `-f` | Maximum size of files created by the shell. | No |
| **`locks`** | `RLIMIT_LOCKS` | `-x` | Maximum number of file locks held. | No |
| **`memlock`** | `RLIMIT_MEMLOCK` | `-l` | Maximum size of memory that may be locked into RAM. | Yes |
| **`msgqueue`** | `RLIMIT_MSGQUEUE` | `-q` | Maximum number of bytes in POSIX message queues. | No |
| **`nice`** | `RLIMIT_NICE` | `-e` | Maximum scheduling priority (`nice` value). | Yes |
| **`nofile`** | `RLIMIT_NOFILE` | `-n` | Maximum number of open file descriptors. | Yes |
| **`nproc`** | `RLIMIT_NPROC` | `-u` | Maximum number of processes available to a single user. | No |
| **`rss`** | `RLIMIT_RSS` | `-m` | Maximum resident set size (standard Linux usually ignores this). | No |
| **`rtprio`** | `RLIMIT_RTPRIO` | `-r` | Maximum real-time scheduling priority. | Yes |
| **`rttime`** | `RLIMIT_RTTIME` | *N/A* | Timeout for real-time tasks (usually not exposed by standard bash ulimit). | No |
| **`sigpending`** | `RLIMIT_SIGPENDING` | `-i` | Maximum number of pending signals. | No |
| **`stack`** | `RLIMIT_STACK` | `-s` | Maximum stack size. | Yes |

**Ulimit Representation in Shell Environments**

When configuring Ulimit, administrators may notice discrepancies between the configured values and the output of standard shell tools inside the container (e.g., ulimit -a). This discrepancy is a display issue of the shell and is not caused by truncation of values by the container runtime or the kernel. The container runtime and the underlying prlimit system call transmit and store the exact integer values provided in the Pod specification. The kernel maintains these limits in raw units (typically bytes or counts). Standard POSIX-compliant shells (such as bash or sh) apply specific scaling factors when querying and displaying these limits to improve readability.

- Memory resources are converted from bytes to kilobytes (divisor: 1024).

- File/block resources are converted from bytes to 512-byte blocks (divisor: 512).

- Count/time resources generally maintain a 1:1 ratio.

**Value Mapping Table**

The following table demonstrates the mapping between the configured value and the observed output in a standard shell environment. A new column has been added to indicate whether the ulimit is supported.

| Resource Category | Ulimit Flag | Kernel Unit | Shell Display Unit | Scaling Factor | Supported |
| :--- | :---: | :---: | :---: | :---: | :---: |
| **virtual memory** | `-v` | Bytes | Kbytes | $\div 1024$ | No |
| **core file size** | `-c` | Bytes | 1024-byte | $\div 1024$ | Yes |
| **cpu time** | `-t` | Seconds | Seconds | 1 | No |
| **data seg size** | `-d` | Bytes | Kbytes | $\div 1024$ | No |
| **file size** | `-f` | Bytes | 1024-byte | $\div 1024$ | No |
| **file locks** | `-x` | Count | Count | 1 | No |
| **locked memory** | `-l` | Bytes | Kbytes | $\div 1024$ | Yes |
| **msg queue** | `-q` | Bytes | Bytes | 1 | No |
| **nice priority** | `-e` | Priority | Priority | 1 | Yes |
| **open files** | `-n` | Count | Count | 1 | Yes |
| **processes** | `-u` | Count | Count | 1 | No |
| **resident set size** | `-m` | Bytes | Kbytes | $\div 1024$ | No |
| **real-time prio** | `-r` | Priority | Priority | 1 | Yes |
| **real-time timeout**| *N/A* | Microsecs | Microsecs | 1 | No |
| **sigpending** | `-i` | Count | Count | 1 | No |
| **stack size** | `-s` | Bytes | Kbytes | $\div 1024$ | Yes |

### Test Plan

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests


- `k8s.io/kubernetes/pkg/apis/core/validation`:`2025-12-30` - `85.3%`
- `k8s.io/kubernetes/pkg/kubelet/kuberuntime`: `2026-01-27` - `69.2%`

##### Integration tests

unit tests and e2e tests are sufficient to validate this feature. We don’t need additional integration tests.

##### e2e tests

- Create Pods with ulimits and verify runtime application via exec checks.
- Test privileged vs. unprivileged modes.

### Graduation Criteria

#### Alpha

- ##### Alpha - 1
  - CRI changes merged
- ##### Alpha - 2
  - Integration with Node Declared Features complete
  - Feature implemented with container runtime implementation with new CRI API.

#### Beta
- Add e2e tests to ensure feature availability.
- Mainstream container runtimes (such as containerd/CRI-O) have implemented the changes in CRI.
- Evaluate user feedback to determine if additional ulimit options should be supported. If deemed necessary, update the validation logic and documentation accordingly.

#### GA

- The feature has been stable in Beta for at least 2 Kubernetes releases.
- Multiple major container runtimes support the feature.
- Finalize the list of supported ulimit options based on user feedback and operational experience.

### Upgrade / Downgrade Strategy

Upgrade: After upgrading to a version that supports this KEP, the `ContainerUlimits` feature gate can be enabled at any time.

Downgrade: If downgraded to a version that does not support this KEP, kube-apiserver will revert to strict validation.
Pods that have already applied ulimit settings will continue running with their current configuration.
To disable this feature, all Pods using this configuration must be manually cleared.

### Version Skew Strategy

The new version of kube-apiserver with this feature enabled will accept such Pods.

Due to the integration with the node declaration feature, Pods specifying ulimits will only be scheduled to nodes where the container runtime advertises `Ulimit` support via `RuntimeFeatures`.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?


- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ContainerUlimits`
  - Components depending on the feature gate: `kubelet`, `kube-apiserver`
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes; disable gate and restart components. Existing pods retain applied ulimits until restarted.

###### What happens if we reenable the feature if it was previously rolled back?

Pods can again specify ulimits; no state loss.

###### Are there any tests for feature enablement/disablement?

During the alpha stage, unit tests for enabling and disabling the toggle functionality will be added to the validation code. Manual testing will also be conducted during the beta stage, and the testing process will be documented here.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The [Version Skew Strategy](#version-skew-strategy) section covers this point.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be validated via manual testing. 

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

Operators can determine if this feature is in use by inspecting the Pod specifications using kubectl. Specifically, they can check for the presence of the securityContext field in the Pod spec and look for the ulimits configuration. For example:
`kubectl get pod <pod-name> -o yaml | grep -A 5 ulimits`


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
- [x] Other (treat as last resort)
  - Details: Users can exec into the container and run `cat /proc/1/limits` or `ulimit -a` to verify the configured limits are applied.  

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

No.

### Scalability


###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.


###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes; Pod objects increase by ~100B per ulimit entry (minimal).

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

Yes, it can.

File descriptor exhaustion (nofile):
- Each container can request up to 1,048,576 file descriptors (FDs).
- Risk: Many containers with high nofile values can exhaust the kernel FD table.
- Symptom: Node-wide "too many open files" errors, potentially affecting system services.


Process table exhaustion (nproc):
- The nproc limit is applied per-UID on the host, not per-container.
- User namespaces do not affect this behavior, as the nproc limit is enforced based on the host UID.
- For example, 10 containers running as UID 1000 with nproc=1000 share a single pool of 1000 processes.
- Symptom: Fork failures may occur even if individual containers are under their specified limits.
- Reference: For more details, see [Resource limits in user namespaces](https://lwn.net/Articles/842842/).

Memory exhaustion (memlock):
- High memlock values can exhaust physical RAM.
- Symptom: Out-of-memory (OOM) kills and node instability.


Mitigations:
- Validation caps nofile at 1,048,576.
- Documentation includes warnings against setting unnecessarily high values.
- Existing node-level monitoring tracks file descriptor and process usage.
- Validation allows -1 values for all containers. This maps to the maximum value permitted by the kernel, ensuring they cannot exceed system-imposed hard limits while still allowing flexible configuration.

**Operator Recommendations:**

- Using third-party Prometheus exporters to monitor node-level resources, such as the number of file descriptors and processes, for example, [node_exporter](https://github.com/prometheus/node_exporter).
- Use admission webhooks or `ValidatingAdmissionPolicy` to restrict high ulimit values.
- Test configurations in a staging environment before deploying to production.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No impact to the running workloads

###### What are other known failure modes?

- File Descriptor Exhaustion (nofile)
  - Description: Many containers requesting high `nofile` values can exhaust node file-descriptor capacity.
  - Detection: Node-wide "too many open files" errors, FD metrics spikes, or EMFILE errors in application logs.
  - Mitigations: Cap `nofile` via validation/admission, monitor FD usage, and limit/scale down offenders.

- Process Table Exhaustion (nproc)
  - Description: The `nproc` limit is applied per-UID on the host, not per-container; containers sharing a host UID can exhaust the host process quota.
  - Detection: Fork failures in application logs, high PID counts from monitoring, or kernel messages about process limits.
  - Mitigations: Restrict `nproc` via admission, prefer distinct UIDs, and monitor node PID usage.

- Memory Exhaustion (memlock)
  - Description: Large `memlock` values can consume physical RAM and cause OOMs on the node.
  - Detection: Memory pressure alerts, OOM events, or sudden node memory usage increases.
  - Mitigations: Validate/limit `memlock`, restrict to trusted workloads, and monitor node memory.


###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

* 2025-12-29: Initial proposal

## Drawbacks

- Setting ulimits too high (e.g., RLIMIT_NOFILE to max) can lead to node-wide file descriptor exhaustion
- PID-level limits do not scale to multi-process workloads like cgroups.
- Adds further complexity to the pod/container spec.
- Introduces platform and node inconsistencies per each uniquely defined pod spec.

## Alternatives

Configure default ulimits directly in the container runtime
  - This applies a global default to all containers on a node, making it impossible to support mixed workloads with conflicting requirements (e.g., high nofile vs. restricted nproc).

Modifying Ulimits by adding an extra entrypoint script:
  - This approach is not secure, lacks reproducibility, and is very inconvenient to use.

Using a ulimit-adjuster NRI plugin:
  - [The ulimit-adjuster NRI plugin](https://github.com/containerd/nri/tree/main/plugins/ulimit-adjuster) provides a more flexible and secure alternative.
  - It does not require privilege escalation, adjustments to host limits, or modifications to the container runtime itself.
  - NRI (Node Resource Interface) is supported by both CRI-O and containerd, making it a viable option for clusters using these runtimes.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
