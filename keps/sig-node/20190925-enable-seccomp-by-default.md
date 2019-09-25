---
title: enable-seccomp-by-default
authors:
  - "@pjbgf"
owning-sig: sig-node
participating-sigs:
  - sig-auth
  - sig-apimachinery
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-09-25
last-updated: yyyy-mm-dd
status: provisional
see-also:
  - "https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/seccomp.md"
  - "20190717-seccomp-ga.md"
replaces:
  - "https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/seccomp.md"
superseded-by: {}
---

# enable-seccomp-by-default

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [1. Audit mode support](#1-audit-mode-support)
  - [2. Built-in profiles](#2-built-in-profiles)
  - [3. New Kind for custom profiles](#3-new-kind-for-custom-profiles)
  - [4. Set <code>runtime/default-audit</code> as the default profile](#4-set--as-the-default-profile)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
    - [Story 4](#story-4)
  - [Implementation Details](#implementation-details)
  - [1. Audit mode support (Details)](#1-audit-mode-support-details)
  - [2. Built-in profiles (Details)](#2-built-in-profiles-details)
  - [3. New Kind for custom profiles (Details)](#3-new-kind-for-custom-profiles-details)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
- [References](#references)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

A proposal to enhance the current seccomp support, enabling it by default in new deployments by leveraging a new audit-only feature.

## Motivation

Kubernetes supports Seccomp in some capacity since v1.3. But it is [disabled by default](https://github.com/kubernetes/kubernetes/issues/81115), as highlighted by the recent [security audit](https://github.com/kubernetes/kubernetes/issues/81146). 
Enabling it poses a few challenges to users due to the absence of 1) auditing capabilities, 2) user-friendly ways to create and maintain profiles, and 3) mechanisms to synchronise custom profiles across nodes. 

Removing such barriers of entry for seccomp usage in Kubernetes clusters, will encourage adoption and provide a safer baseline across the ecosystem.


### Goals

- Add audit-mode support for logging violations instead of blocking them.
- Set clusters to use the new audit profile (i.e. `runtime/default-audit`) by default.  
- Define built-in seccomp profiles that are secure and meaningful.
- Create a new Kind to represent seccomp profiles.
- Provide a new mechanism for profiles to be sent to CRI without the use of files in the node's filesystem.
- Avoid breaking changes for Kubernetes api and user workloads.

### Non-Goals

- Changes to make Seccomp GA. This is being covered by another [KEP](20190717-seccomp-ga.md).
- Changes to `PodSecurityPolicy`.
- Windows Support.

## Proposal

The proposed change aims to make the seccomp support in Kubernetes more user-friendly, by 1) supporting audit mode, 2) creating new built-in profiles, 3) creating a new Kind for seccomp profiles and 4) enabling seccomp in audit-mode by default. 


### 1. Audit mode support 
In Linux kernel 4.14 support for a new seccomp return action named `SECCOMP_RET_LOG` was added. This return action permits that systems calls are logged and then executed. Before such feature, seccomp profiles had only the ability to block, allow and trace system calls, which could lead to disruptive profiles that would evolve through trial and error. 

Audit mode empowers users to monitor the impact of new profiles for extensive periods of time before switching into more restrictive modes. An example of it would be a profile with safe system calls whitelisted using `SCMP_ACT_ALLOW` and a default action of `SCMP_ACT_LOG`, instead of the current `SCMP_ACT_ERRNO`. The result would be that all system calls would be executed, however, only the ones outside the whitelist would be logged into the system logs.

To arrive in Kubernetes, this functionality needs to first make it to libseccomp and OCI/runc. It was recently added to Libseccomp-golang [v0.9.1](https://github.com/seccomp/libseccomp-golang/releases/tag/v0.9.1) as `SCMP_ACT_LOG`. And it has currently a PR to make it into [runC](https://github.com/opencontainers/runc/pull/1951) and the [OCI](https://github.com/opencontainers/runtime-spec/pull/1019).

The support is based on the downstream dependencies, therefore Kubernetes changes are not _required_. However, some changes may be improve its usability as pointed out on the [Implementation Details](#implementation-details) section.


### 2. Built-in profiles
The table below shows what built-in profiles and the two supported ways to create user-defined profiles.

| Profile Name 	| Description 	| Status 	| Requires Audit Support 	| Fallback profile 	|
|-------------------------	|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------	|-------------------------------------------	|-----------------------------------------	|------------------	|
| `runtime/default` 	| The default container runtime. Syscalls outside allowed list are blocked. 	| Unchanged 	| No 	| N/A 	|
| `runtime/default-audit` 	| Allows the same syscalls as `runtime/default`, but logs all violations. 	| New 	| Yes 	| `unconfined` 	|
| `runtime/audit-verbose` 	| Remove all whitelisted calls, logging every time any system calls are used. Useful for creating new profiles based on the execution of a container. 	| New 	| Yes 	| `unconfined` 	|
| `custom/<profile-name>` 	| User defined profiles based off the new `SeccompProfile` Kind. 	| New 	| Only when `SCPM_ACT_LOG` is being used 	| `unconfined` 	|
| `localhost/<path>` 	| User defined profile as a file on the node located at <seccomp_root>/<path>, where <seccomp_root> is defined via the  --seccomp-profile-root flag on the Kubelet. _Note that the user is responsible for physically synchronising the profile files across all nodes._ 	| Unchanged 	| Only when ` SCPM_ACT_LOG` is being used 	| `unconfined` 	|
| `docker/default` 	| The Docker default seccomp profile is used. Deprecated as of Kubernetes 1.11. Use  `runtime/default` instead. 	| Unchanged, Deprecated 	| No 	| N/A 	|
| `unconfined` 	| Seccomp is not applied to the container processes (the current default in Kubernetes), if no alternative is provided. Usage should be disencouraged. 	| Unchanged 	| No 	| N/A 	|


### 3. New Kind for custom profiles

Promote seccomp profiles to first-class citizen as a new *cluster-level* resource, represented by a new `SeccompProfile` API resource. 
The new API resource would remove the existing overhead of creating and synchronising files across all new and existing nodes. Allowing users to maintain custom profiles by interacting with the API server.

More details on types needed and usage on the [Implementation Details](#implementation-details) section.

 
### 4. Set `runtime/default-audit` as the default profile

By default clusters will have `runtime/default-audit` applied to all containers (users can opt-out). This profile is similar to the existing `runtime/default`, with the difference being that syscalls used that are outside the allowed list will be logged instead of having their execution blocked. This improves the default auditing capabilities of high risk syscalls whilst not breaking existing workloads.

An extra benefit is to provide an easier route for users to move to `runtime/default`. They can determine their workload readiness by going through violations on syslogs, instead of having to profile each application individually. 

To support audit mode, some validation will be required ensuring the downstream dependencies (Kernel, Libseccomp and runc) also supports it. If audit mode is not supported, the default seccomp profile will be set back to `unconfined`, for backwards compatibility. 

### User Stories

#### Story 1
As a user and administrator, I want to be able to assess what pods within my cluster would be affected by applying more restrictive seccomp profiles.

#### Story 2
As a user, I want to be able to assess whether my pods will be affected by a more restrictive seccomp profiles.

#### Story 3
As a user and administrator, I want to be able to create and maintain custom seccomp profiles without having to physically copy files across all my existing (and new) nodes.

#### Story 4
As a user and administrator, I want to be able Kubernetes to audit all potentially dangerous system calls from my containers by default.


### Implementation Details

### 1. Audit mode support (Details)

Kubernetes will continue to be unaware of downstream support. If a user tries to use unsupported actions (i.e. `SCMP_ACT_LOG`) today, the lower level dependencies will return an error. As a result, the container won't be able to start. We are proposing no changes to this behaviour.


### 2. Built-in profiles (Details)

The internal built-in profiles will be implemented in golang, not allowing users to amend them. Similar to the implementation in the [containerd](https://github.com/containerd/containerd/blob/master/contrib/seccomp/seccomp_default.go) project.


### 3. New Kind for custom profiles (Details)

To represent the seccomp profiles a new `SeccompProfile` resource type will be created: 
```
// SeccompProfile defines custom seccomp profiles to be enforced at runtime.
type SeccompProfile struct {
	metav1.TypeMeta
	// +optional
	metav1.ObjectMeta

	// +optional
	Spec LinuxSeccomp
}
```

Referencing the types defined by the [OCI Spec](https://github.com/opencontainers/runtime-spec/blob/master/specs-go/config.go#L556):

```
// LinuxSeccomp represents syscall restrictions
type LinuxSeccomp struct {
	DefaultAction LinuxSeccompAction `json:"defaultAction"`
	Architectures []Arch             `json:"architectures,omitempty"`
	Flags         []LinuxSeccompFlag `json:"flags,omitempty"`
	Syscalls      []LinuxSyscall     `json:"syscalls,omitempty"`
}

// Arch used for additional architectures
type Arch string

// LinuxSeccompFlag is a flag to pass to seccomp(2).
type LinuxSeccompFlag string

// Additional architectures permitted to be used for system calls
// By default only the native architecture of the kernel is permitted
const (
	ArchX86         Arch = "SCMP_ARCH_X86"
	ArchX86_64      Arch = "SCMP_ARCH_X86_64"
	ArchX32         Arch = "SCMP_ARCH_X32"
	ArchARM         Arch = "SCMP_ARCH_ARM"
	ArchAARCH64     Arch = "SCMP_ARCH_AARCH64"
	ArchMIPS        Arch = "SCMP_ARCH_MIPS"
	ArchMIPS64      Arch = "SCMP_ARCH_MIPS64"
	ArchMIPS64N32   Arch = "SCMP_ARCH_MIPS64N32"
	ArchMIPSEL      Arch = "SCMP_ARCH_MIPSEL"
	ArchMIPSEL64    Arch = "SCMP_ARCH_MIPSEL64"
	ArchMIPSEL64N32 Arch = "SCMP_ARCH_MIPSEL64N32"
	ArchPPC         Arch = "SCMP_ARCH_PPC"
	ArchPPC64       Arch = "SCMP_ARCH_PPC64"
	ArchPPC64LE     Arch = "SCMP_ARCH_PPC64LE"
	ArchS390        Arch = "SCMP_ARCH_S390"
	ArchS390X       Arch = "SCMP_ARCH_S390X"
	ArchPARISC      Arch = "SCMP_ARCH_PARISC"
	ArchPARISC64    Arch = "SCMP_ARCH_PARISC64"
)

// LinuxSeccompAction taken upon Seccomp rule match
type LinuxSeccompAction string

// Define actions for Seccomp rules
const (
	ActKill  LinuxSeccompAction = "SCMP_ACT_KILL"
	ActTrap  LinuxSeccompAction = "SCMP_ACT_TRAP"
	ActErrno LinuxSeccompAction = "SCMP_ACT_ERRNO"
	ActTrace LinuxSeccompAction = "SCMP_ACT_TRACE"
	ActAllow LinuxSeccompAction = "SCMP_ACT_ALLOW"
	ActLog   LinuxSeccompAction = "SCMP_ACT_LOG"    ## Pending pull request
)

// LinuxSeccompOperator used to match syscall arguments in Seccomp
type LinuxSeccompOperator string

// Define operators for syscall arguments in Seccomp
const (
	OpNotEqual     LinuxSeccompOperator = "SCMP_CMP_NE"
	OpLessThan     LinuxSeccompOperator = "SCMP_CMP_LT"
	OpLessEqual    LinuxSeccompOperator = "SCMP_CMP_LE"
	OpEqualTo      LinuxSeccompOperator = "SCMP_CMP_EQ"
	OpGreaterEqual LinuxSeccompOperator = "SCMP_CMP_GE"
	OpGreaterThan  LinuxSeccompOperator = "SCMP_CMP_GT"
	OpMaskedEqual  LinuxSeccompOperator = "SCMP_CMP_MASKED_EQ"
)

// LinuxSeccompArg used for matching specific syscall arguments in Seccomp
type LinuxSeccompArg struct {
	Index    uint                 `json:"index"`
	Value    uint64               `json:"value"`
	ValueTwo uint64               `json:"valueTwo,omitempty"`
	Op       LinuxSeccompOperator `json:"op"`
}

// LinuxSyscall is used to match a syscall in Seccomp
type LinuxSyscall struct {
	Names  []string           `json:"names"`
	Action LinuxSeccompAction `json:"action"`
	Args   []LinuxSeccompArg  `json:"args,omitempty"`
}
```


The profile definition would be passed as a serialised json object inside an dockerOpt object, in the same way that it is done currently for file based profiles:
```
jsonSeccomp, _ := json.Marshal(profile.Spec)
return []dockerOpt{{"seccomp", string(jsonSeccomp), seccompProfileName}}, nil
```
_Needs confirmation as to whether this would also work for non-docker implementations._


User-defined Seccomp profiles would be created this way:

```
apiVersion: v1
kind: SeccompProfile
metadata:
  name: my-custom-profile
spec:
  defaultAction: SCMP_ACT_ERRNO
  architectures:
  - SCMP_ARCH_X86_64
  syscalls:
  - action: SCMP_ACT_ALLOW
    names: 
    - accept
    - accept4
    - access
    - alarm
    args: {}
``` 

And referenced by `custom/my-custom-profile`.


## Implementation History
- 2019-09-25: Initial KEP


## Alternatives

**Start deprecation process for `localhost/<path>`.** The new `SeccompProfile` will better support custom profiles. Starting the deprecation process would signal to users what the end goal is. However, this can be started at a later stage once users have already started using the new approach.


**Implement `SeccompProfile` as a namespaced resource.** However, the points considered for not doing it were: 

1. The current concept of profiles _is_ a cluster-level resource, although not yet materialised as its own resource type. 
1. Profiles are closely aligned with `PodSecurityPolicy` objects. Making the new `SeccompProfile` Kind as an namespaced object would break the existing user expectation of how both concepts interact. 
1. Avoid the proliferation of several similar custom policies across the cluster. 


**Downstream seccomp support awareness.** Validation could be added to assert whether the Seccomp Profile could be applied by the downstream dependencies on a _per- node_ basis, and lead to a list of available profiles for each node. This would benefit clusters with heterogeneus nodes. It would also benefit the usage of the current `localhost/<path>` profile, which an administrator has no way to tell which nodes have them and which ones don't. 

This can be treated as an incremental enhancement in the future, based on the community feedback.

**Use an unstructured data type instead of SeccompSpec.** The main motivation here would be to remove the tight coupling with OCI and downstream components. However, by using unstructured data type may bring its own challenges - especially around migration paths in case the specification changes.

## References
- [Original Seccomp Proposal](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/seccomp.md)
- [Seccomp GA KEP](20190717-seccomp-ga.md)