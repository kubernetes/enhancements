# KEP-4265: add ProcMount option

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
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

For Linux containers, the Kubelet instructs container runtimes to mask and set as read-only certain paths in `/proc`.
This is to prevent data from being exposed into a container that should not be.
However, there are certain use-cases where it is necessary to turn this off.

This KEP proposes adding a field to the Pod security context to allow bypassing the usual restrictions.

In 1.12, this was introduced as the ProcMountType feature gate, and has it has languished in alpha ever since. This KEP is
a successor to (and heavily based on) https://github.com/kubernetes/community/pull/1934, updated for the modern era.

## Motivation

Some end users would like to run unprivileged containers _nested inside_ a Kubernetes container using user namespaces. The outer container is started by the CRI implementation.
Kubernetes defaults to masking the `/proc` mount of a container, setting some paths as read only. To run a nested container within an unprivileged Pod, a user would need a way to
override that default masking behavior.

Please see the following filed issues for more information:
- [opencontainers/runc#1658](https://github.com/opencontainers/runc/issues/1658#issuecomment-373122073)
- [moby/moby#36597](https://github.com/moby/moby/issues/36597)
- [moby/moby#36644](https://github.com/moby/moby/pull/36644)

### Goals

- Allow users to opt out of the CRI masking `/proc` for Linux containers.

### Non-Goals

## Proposal


Add a new `string` named `procMount` to the `securityContext` definition for choosing from a set of proc mount isolation mode options.

The default for `procMount` is `Default`, which instructs the container runtime to mask the aforementioned paths.

This will look like the following in the spec:

```go
type ProcMountType string

const (
    // DefaultProcMount uses the container runtime default ProcType.  Most 
    // container runtimes mask certain paths in /proc to avoid accidental security
    // exposure of special devices or information.
    DefaultProcMount ProcMountType = "Default"

    // UnmaskedProcMount bypasses the default masking behavior of the container
    // runtime and ensures the newly created /proc the container stays intact with
    // no modifications.  
    UnmaskedProcMount ProcMountType = "Unmasked"
)

procMount *ProcMountType
```

where nil is default, and is interpreted as "Default" ProcMountType.

When the kubelet is presented with a pod that has a ProcMountType as Unmasked, it will edit the default list of
masked paths it passes down to the CRI to be [empty](https://github.com/kubernetes/kubernetes/blob/964529b/pkg/securitycontext/util.go#L216) which it does
with the [CRI request](https://github.com/kubernetes/kubernetes/blob/964529b/staging/src/k8s.io/cri-api/pkg/apis/runtime/v1/api.proto#L889-L891).

This requires changes to the CRI runtime integrations so that kubelet will add the specific `unmasked` option.
This was done after alpha:
- CRI-O has support in v1.25.0 after https://github.com/cri-o/cri-o/pull/6025/commits/4102586132214263c5d0ae93ec257432653ab82b
- containerd has support in 1.6. See https://github.com/containerd/containerd/pull/5070/commits/07f1df4541d6a81c205d194f4f6ea3a6a95c3e29

The main use case for unmasking paths in `/proc` are for a user nesting unprivileged containers within a container. However, having an Unmasked ProcMountType
is a privileged operation, and thus is part of the [privileged](https://k8s.io/docs/concepts/security/pod-security-standards/#privileged) Pod Security Admission (PSA). Since a user must have be in
the privileged policy, they are also trusted to choose the correct user ID and run a workload that won't interfere with the host.

A container running as root user on the host and an unmasked `/proc` could be able to write to the host `/proc`, and thus this privileged designation is appropriate.

### User Stories (Optional)

#### Story 1

As a cluster admin, I would like a way to nest containers within containers. To do so, kernel the top level containers need an unmasked /proc.

#### Story 2

As a kubernetes user, I may want to build containers from within a kubernetes container. 
See [this article for more information](https://github.com/jessfraz/blog/blob/master/content/post/building-container-images-securely-on-kubernetes.md).

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

- A user turning this on without user namespaces enabled
    - Admission should deny a pod that tries to use `ProcMountType: Unmasked` with `HostUsers: true`
- More trust in user namespacing/the kernel instead of container runtime
    - This is probably the correct direction to head in.

## Design Details

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

- `pkg/securitycontext`: `10-05-2023` - `70.04`

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- N/A (Kubelet barely defines integration tests today, focusing on e2e_node tests instead)

##### e2e tests

- test/e2e_node

- additional tests should be added to e2e_node suite to test the adherence of the ProcMount field
  - Test default behavior actually masks /proc paths.
  - Test Unmasked behavior is not masking /proc paths.
  - Test PSA integration (if possible to test in e2e)
  - Test that Windows pod cannot be scehduled with the value of ProcMount specifies

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

- Feature implemented behind a feature flag
- Add e2e tests for the feature (must be done before beta)
    - Including ones for enabling/disabling the feature

#### Beta

- Explicitly require hostUsers option to be `false` if this option is enabled.
    - Otherwise, this option effectively becomes another "privileged" field

#### GA

- Allowing time for feedback


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
Turn off the feature gate to turn off the feature.

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

The feature gate is only processed by the API server--Kubelet has no awareness of it. API server will scrub the ProcMount field from the request
if it doesn't support the feature gate. Since all supported Kubelet versions support ProcMountType field, there's no version skew worry.
API server can have the feature gate toggled without worrying about doing the same for Kubelets.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ProcMountType
  - Components depending on the feature gate: kube-apiserver (kube-apiserver filters `procMount` field if it's not enabled).

###### Does enabling the feature change any default behavior?

No, only gives a user access to the Unmasked ProcMountType

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. This can be done by removing the feature gate from all kube-apiservers. To fully roll back, the nodes will need to be drained or rebooted,
as the Kubelet will not remove the `procMount` of an already running container.

###### What happens if we reenable the feature if it was previously rolled back?

Nothing special. The pod's `procMount` field depends on where in the enablement process the kube-apiserver was when it was created.
The container has to be restarted to be up to date with the kube-apiserver.

###### Are there any tests for feature enablement/disablement?

Yes. I have manually tested feature enablement and disablement on kube-apiserver, and verified that pods are not recreated without
a drain. There will be an e2e test to verify this as well.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

It cannot. Either the kube-apiserver has the feature gate on or not. If it has it on, then workloads with the feature enabled will get an Unmasked
ProcMountType if they request it. If it's off, then the kube-apiserver will force it to default, and the container's creation will move forward
without an Unmasked ProcMountType.

Already running workloads aren't stopped and restarted on a feature revert, so an admin would need to reboot or drain to impact running workloads.

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

The behavior of this feature has been consistent for more than 10 minor releases, so these tests are less relevant now.
Put differently: there is no upgrade->downgrade->upgrade path between supported versions of kubernetes that support this feature.

Manual testing has been done between versions that do support it, toggling the feature on and off. In these cases, the feature works as described.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

`kubectl get pods --all-namespaces -o jsonpath="{range .items[*]}{.metadata.name}{' '}{.spec.containers[*].securityContext.procMount}{'\n'}{end}"  | grep -i unmasked`
Will print all pods that has an Unmasked ProcMountType, along with the pod name.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [x] Other (treat as last resort)
  - Details: Container created with the Unmasked ProcMountType have paths [here](https://github.com/kubernetes/kubernetes/blob/964529b/pkg/securitycontext/util.go#L193) as writable, not read only.
    - "/proc/asound",
	- "/proc/acpi",
	- "/proc/kcore",
	- "/proc/keys",
	- "/proc/latency_stats",
	- "/proc/timer_list",
	- "/proc/timer_stats",
	- "/proc/sched_debug",
	- "/proc/scsi",
	- "/sys/firmware",
  - Another option is to run `kubectl exec $podname -- mount | grep /proc`.
    - If there's just one mount, and it looks like `proc on /proc type proc (rw,nosuid,nodev,noexec,relatime)` this is an unmasked `/proc`

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No noticeable change in pod start times when this feature is enabled.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `kubelet_pod_start_sli_duration_seconds`
  - [Optional] Aggregation method:
  - Components exposing the metric: kubelet


I don't think any would be useful.
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

I don't think any would be useful.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- A CRI implementation that supports this feature
    - All supported versions currently do.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

ProcMountType in the pod spec

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

There is one additional field in the pod API: `procMount`. It has an enum value of two values: `Default` and `Unmasked`.
The Kubelet is also passing the MaskedPaths to the CRI, which involves a single slice of strings.
When the value `Default` is chosen, the slice is defined [here](https://github.com/kubernetes/kubernetes/blob/964529b/pkg/securitycontext/util.go#L193).
If `Unmasked`, the slice is empty.
Both of these are size changes on the order of bytes and can be considered negligible.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

Potentially a malicious user given access and running with a root container in the host context could mess with the host processes.
PSA has already been configured to mitigate this by required a user be in a privileged namespace to get access to the field.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No effect

###### What are other known failure modes?

Malicious user gaining access to the host `/proc` with a rootful container
- admission should be updated to deny unmasked ProcMountType without user namespaces (hostUsers: true)

###### What steps should be taken if SLOs are not being met to determine the problem?

The field can be unset in a pod spec (or feature gate turned off) to see if SLOs met after the feature is disabled for pods.

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

2018-05-07: k/community update opened
2018-05-27: k/kubernetes PR merged with support.
2023-10-02: KEP opened and retargeted at Alpha
2024-02-26: [Update](https://github.com/kubernetes/kubernetes/pull/123520) Unmasked ProcMountType to fail validation without a pod level user namespace.
2024-05-31: Added e2e [tests](https://github.com/kubernetes/kubernetes/pull/123303)
2024-05-31: KEP updated to Beta
2025-01-31: KEP updated to on by default Beta
2026-01-29: KEP updated to GA

## Alternatives
- `--oci-worker-no-process-sandbox` like in [BuildKit](https://github.com/moby/buildkit/blob/v0.12.2/examples/kubernetes/job.rootless.yaml#L31)
    - Not broadly supported with other container runtimes/builders. 
- Update the kernel to allow mounting a new procfs with masks.
    - Proposed, but [denied](https://patchwork.kernel.org/project/linux-fsdevel/patch/20180404115311.725-1-alban@kinvolk.io/) in the kernel
- Adopt a similar approach to LXD where `/proc` and `/sys` are mounted to different locations within the container, instead of masked.
- Give all pods with `hostUsers: false` (pod level user namespace) access to these mounts by default
    - Even though it potentially is safe, it opens an argument that user namespaced pods are less secure than non user namespaced pods. The weakining of these boundries should be opt-in.
- Ditch this option
    - Most use cases don't really need this. However, if a pod wants to be able to, for instance, set its own sysctls, it would need this option.

## Infrastructure Needed (Optional)
