# KEP-5785: Per-container ulimits configuration

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
Without native support, users are forced to resort to workarounds such as modifying host ulimits, setting ulimits through additional entrypoint scripts in privileged mode, or patching container runtime configurations—all of which lack portability, security, and scalability in multi-tenant clusters.

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

Risk: Excessive nofile configuration As discussed in moby/moby#45534, setting `RLIMIT_NOFILE` to effectively infinite can cause issues, leading to excessive memory consumption or crashes.

Mitigation: Kubelet validation will not arbitrarily cap values, but documentation should warn against using MaxInt64 for nofile unless necessary. The runtime implementation logic (detailed below) handles `-1` by mapping it to the host's maximum, avoiding "true infinity" issues where possible.

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

We will enforce a strict allowlist of ulimit names in the API validation layer to prevent passing arbitrary or unsafe strings to the kernel.

```
var ulimitNameMapping = sets.New(
  "as",
	"core",
	"cpu",
	"data",
	"fsize",
	"locks",
	"memlock",
	"msgqueue",
	"nice",
	"nofile",
	"nproc",
	"rss",
	"rtprio",
	"rttime",
	"sigpending",
	"stack",
)
```

Additional Rules:

- If the container is not privileged, setting a limit to -1 (Unlimited) is forbidden. Unprivileged processes cannot raise their hard limits, and allowing them to request "Unlimited" implies bypassing the system defaults set by the admin.

- Soft limit must be less than or equal to Hard limit.

- In most modern Linux distributions, the maximum allowed value for `RLIMIT_NOFILE` is 1048576. When attempting to set a limit higher than this global threshold, the setrlimit system call will return an `EPERM` (Operation not permitted) error, even for privileged processes. Therefore, we will also add validation to ensure that the value of nofile does not exceed 1048576.

**Kubelet and Runtime Implementation:**

The Kubelet will convert the API Ulimit objects into CRI Ulimit objects. The Container Runtime (e.g., `containerd/crio`) acts as follows:
1. The runtime calculates the OCI spec options. It handles the conversion of `-1` values and standardizes names (prepending `RLIMIT_` if missing).
2. If a user requests `-1`, the runtime resolves this to the current limit of the runtime process or the system max, ensuring valid kernel values are passed.
3. In the low-level runtime (e.g., `runc/crun`), we use prlimit to apply these limits to the container PID before user code execution but after the process is created.


**Node Declared Features Integration**

To ensure seamless integration with node declared features, the Kubelet must be able to discover if the underlying container runtime supports the `Ulimits` configuration. This prevents scheduling Pods with custom ulimits to nodes that cannot enforce them.

We define a specific runtime handler features: `ContainerUlimits`.

After integrating node declared features, it determines whether the container runtime supports the `ContainerUlimits` feature, enabling early-stage decision-making during scheduling to avoid scheduling Pods with custom ulimits to nodes that do not support this feature.

**Mapping Between API Field Names and System Limits**

The following table establishes the correspondence between the Pod API field names, the standard shell ulimit flags, and the underlying Linux kernel constants.
| Pod Spec Name | Kernel Constant | Ulimit Flag | Description |
| :--- | :--- | :---: | :--- |
| **`as`** (or `vmem`) | `RLIMIT_AS` | `-v` | Maximum amount of virtual memory available to the process. |
| **`core`** | `RLIMIT_CORE` | `-c` | Maximum size of core files created. |
| **`cpu`** | `RLIMIT_CPU` | `-t` | CPU time limit in seconds. |
| **`data`** | `RLIMIT_DATA` | `-d` | Maximum size of a process's data segment. |
| **`fsize`** | `RLIMIT_FSIZE` | `-f` | Maximum size of files created by the shell. |
| **`locks`** | `RLIMIT_LOCKS` | `-x` | Maximum number of file locks held. |
| **`memlock`** | `RLIMIT_MEMLOCK` | `-l` | Maximum size of memory that may be locked into RAM. |
| **`msgqueue`** | `RLIMIT_MSGQUEUE` | `-q` | Maximum number of bytes in POSIX message queues. |
| **`nice`** | `RLIMIT_NICE` | `-e` | Maximum scheduling priority (`nice` value). |
| **`nofile`** | `RLIMIT_NOFILE` | `-n` | Maximum number of open file descriptors. |
| **`nproc`** | `RLIMIT_NPROC` | `-u` | Maximum number of processes available to a single user. |
| **`rss`** | `RLIMIT_RSS` | `-m` | Maximum resident set size (standard Linux usually ignores this). |
| **`rtprio`** | `RLIMIT_RTPRIO` | `-r` | Maximum real-time scheduling priority. |
| **`rttime`** | `RLIMIT_RTTIME` | *N/A* | Timeout for real-time tasks (usually not exposed by standard bash ulimit). |
| **`sigpending`** | `RLIMIT_SIGPENDING` | `-i` | Maximum number of pending signals. |
| **`stack`** | `RLIMIT_STACK` | `-s` | Maximum stack size. |

**Ulimit Representation in Shell Environments**

When configuring Ulimit, administrators may notice discrepancies between the configured values and the output of standard shell tools inside the container (e.g., ulimit -a). This discrepancy is a display issue of the shell and is not caused by truncation of values by the container runtime or the kernel. The container runtime and the underlying prlimit system call transmit and store the exact integer values provided in the Pod specification. The kernel maintains these limits in raw units (typically bytes or counts). Standard POSIX-compliant shells (such as bash or sh) apply specific scaling factors when querying and displaying these limits to improve readability.

- Memory resources are converted from bytes to kilobytes (divisor: 1024).

- File/block resources are converted from bytes to 512-byte blocks (divisor: 512).

- Count/time resources generally maintain a 1:1 ratio.

**Value Mapping Table**

The following table demonstrates the mapping between the configured value and the observed output in a standard shell environment:

| Resource Category | Ulimit Flag | Kernel Unit | Shell Display Unit | Scaling Factor |
| :--- | :---: | :---: | :---: | :---: |
| **virtual memory** | `-v` | Bytes | Kbytes | $\div 1024$ |
| **core file size** | `-c` | Bytes | 1024-byte | $\div 1024$ |
| **cpu time** | `-t` | Seconds | Seconds | 1 |
| **data seg size** | `-d` | Bytes | Kbytes | $\div 1024$ |
| **file size** | `-f` | Bytes | 1024-byte | $\div 1024$ |
| **file locks** | `-x` | Count | Count | 1 |
| **locked memory** | `-l` | Bytes | Kbytes | $\div 1024$ |
| **msg queue** | `-q` | Bytes | Bytes | 1 |
| **nice priority** | `-e` | Priority | Priority | 1 |
| **open files** | `-n` | Count | Count | 1 |
| **processes** | `-u` | Count | Count | 1 |
| **resident set size** | `-m` | Bytes | Kbytes | $\div 1024$ |
| **real-time prio** | `-r` | Priority | Priority | 1 |
| **real-time timeout**| *N/A* | Microsecs | Microsecs | 1 |
| **sigpending** | `-i` | Count | Count | 1 |
| **stack size** | `-s` | Bytes | Kbytes | $\div 1024$ |

For more detailed information, refer to the definition of the [ulimit table](https://github.com/bminor/bash/blob/637f5c8696a6adc9b4519f1cd74aa78492266b7f/builtins/ulimit.def#L239-L300) in the Bash source code, which contains hardcoded scaling factors.

### Test Plan

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests


- `k8s.io/kubernetes/pkg/apis/core/validation`:`2025-12-30` - `85.3%`
- `k8s.io/kubernetes/pkg/apis/core/validation`:`2025-12-30` - `85.3%`

##### Integration tests

##### e2e tests

- Create Pods with ulimits and verify runtime application via exec checks.
- Test privileged vs. unprivileged modes.

### Graduation Criteria

#### Alpha

- Feature implemented behind `ContainerUlimits` feature gate.
- CRI changes merged.
- Integration with Node Declared Features complete.


#### Beta
- Add e2e tests to ensure feature availability.
- Mainstream container runtimes (such as containerd/CRI-O) have implemented the changes in CRI.

#### GA

- The feature has been stable in Beta for at least 2 Kubernetes releases.
- Multiple major container runtimes support the feature.

### Upgrade / Downgrade Strategy

Upgrade: After upgrading to a version that supports this KEP, the `ContainerUlimits` feature gate can be enabled at any time.

Downgrade: If downgraded to a version that does not support this KEP, kube-apiserver will revert to strict validation.
Pods that have already applied ulimit settings will continue running with their current configuration.
To disable this feature, all Pods using this configuration must be manually cleared.

### Version Skew Strategy

The new version of kube-apiserver with this feature enabled will accept such Pods.

Due to the integration with the node declaration feature, if the container runtime does not support `Ulimit` restrictions, the Pod will not be scheduled onto nodes that support `Ulimit`.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?


- [ ] Feature gate (also fill in values in `kep.yaml`)
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

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No impact to the running workloads

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

* 2025-12-29: Initial proposal

## Drawbacks

- Setting ulimits too high (e.g., RLIMIT_NOFILE to max) can lead to node-wide file descriptor exhaustion
- PID-level limits do not scale to multi-process workloads like cgroups.

## Alternatives

Configure default ulimits directly in the container runtime
  - This applies a global default to all containers on a node, making it impossible to support mixed workloads with conflicting requirements (e.g., high nofile vs. restricted nproc).

Modifying Ulimits by adding an extra entrypoint script in privileged mode:
  - This approach is not secure, lacks reproducibility, and is very inconvenient to use.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
