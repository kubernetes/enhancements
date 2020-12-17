# Seccomp to GA

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API](#api)
    - [Pod API](#pod-api)
      - [LocalhostProfile](#localhostprofile)
        - [Updating seccomp profiles](#updating-seccomp-profiles)
        - [Profile files managed by the cluster admins](#profile-files-managed-by-the-cluster-admins)
      - [RuntimeProfile](#runtimeprofile)
- [Design Details](#design-details)
  - [Failure and Fallback Strategy](#failure-and-fallback-strategy)
  - [Seccomp root path configuration](#seccomp-root-path-configuration)
  - [Version Skew Strategy](#version-skew-strategy)
    - [Pod Creation](#pod-creation)
    - [PodSecurityPolicy Enforcement](#podsecuritypolicy-enforcement)
    - [Pod Update](#pod-update)
    - [PodTemplates](#podtemplates)
    - [Runtime Profiles](#runtime-profiles)
    - [Kubelet Backwards compatibility](#kubelet-backwards-compatibility)
    - [Upgrade / Downgrade](#upgrade--downgrade)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Localhost profiles](#localhost-profiles)
  - [Updating PodSecurityPolicy API](#updating-podsecuritypolicy-api)
    - [PodSecurityPolicy API](#podsecuritypolicy-api)
    - [PodSecurityPolicy Creation](#podsecuritypolicy-creation)
    - [PodSecurityPolicy Update](#podsecuritypolicy-update)
- [References](#references)
<!-- /toc -->

## Release Signoff Checklist
<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

This is a proposal to upgrade the seccomp annotation on pods to a field, and
mark the feature as GA. This proposal aims to do the _bare minimum_ to clean up the feature, without
blocking future enhancements.

## Motivation

Docker started enforcing a default seccomp profile in v1.10. At the time, Kubernetes (in v1.2)
didn't have a way to control the seccomp profile, so the profile was disabled (set to `unconfined`)
to prevent a regression (see https://github.com/kubernetes/kubernetes/pull/21790). In Kubernetes
v1.3, annotations were added to give users some control over the profile:

```
seccomp.security.alpha.kubernetes.io/pod: {unconfined,docker/default,localhost/<path>}
container.seccomp.security.alpha.kubernetes.io/<container_name>: ...
```

The feature has been more or less unchanged ever since. Also note that the addition predates feature
gates or our modern concept of feature lifecycle. So, even though the annotations include `alpha` in
the key, this is entirely useable on any production GA cluster.

There have been multiple attempts to [change the default
profile](https://github.com/kubernetes/kubernetes/issues/39845) or [formally spec the Kubernetes
seccomp profile](https://github.com/kubernetes/kubernetes/issues/39128), but both efforts were
abandoned due to friction and lack of investment.

Despite the `alpha` label, I think this feature needs to be treated as GA, and we're doing our users
a disservice by leaving it in this weird limbo state. As much as I would like to see seccomp support
fully fleshed out, if we block GA on those enhancements we will remain stuck in the current state
indefinitely. Therefore, I'm proposing we do the absolute minimum to clean up the current
implementation all accurately declare the feature "GA". Future enhancements can follow the standard
alpha -> beta -> GA feature process.

_NOTE: AppArmor is in a very similar state, but with some subtle differences. Promoting AppArmor to
GA will be covered by a separate KEP._

### Goals

- Declare seccomp GA
- Fully document and formally spec the feature support
- Add equivalent API fields to replace seccomp annotations
- Deprecate the seccomp annotations

### Non-Goals

This KEP proposes the absolute minimum to get seccomp to GA, therefore all functional enhancements
are out of scope, including:

- Changing the default seccomp profile from `unconfined`
- Defining any standard "Kubernetes branded" seccomp profiles
- Formally speccing the seccomp profile format in Kubernetes
- Providing mechanisms for loading profiles from outside the static seccomp node directory
- Changing the semantics around seccomp support
- Windows support (seccomp is very linux-specific)

## Proposal

### API

The seccomp API will be functionally equivalent to the current alpha API. This includes the Pod API,
which specifies what profile the pod & containers run with.

#### Pod API

The Pod Seccomp API is immutable, except in [`PodTemplates`](#podtemplates).

```go
type PodSecurityContext struct {
    ...
    // The seccomp options to use by the containers in this pod.
    // +optional
    Seccomp  *SeccompProfile
    ...
}

type SecurityContext struct {
    ...
    // The seccomp options to use by this container. If seccomp options are
    // provided at both the pod & container level, the container options
    // override the pod options.
    // +optional
    SeccompProfile  *SeccompProfile
    ...
}

// Only one profile source may be set.
// +union
type SeccompProfile struct {
    // +unionDescriminator
    Type SeccompProfileType
    // Load a profile defined in static file on the node.
    // The profile must be preconfigured on the node to work.
    // LocalhostProfile cannot be an absolute nor a descending path.
    // +optional
    LocalhostProfile *string
}

type SeccompProfileType string

const (
    SeccompProfileUnconfined      SeccompProfileType = "Unconfined"
    SeccompProfileRuntimeDefault  SeccompProfileType = "RuntimeDefault"
    SeccompProfileLocalhost       SeccompProfileType = "Localhost"
)
```

This API makes the options more explicit than the stringly-typed annotation values, and leaves room
for new profile sources to be added in the future (e.g. Kubernetes predefined profiles or ConfigMap
profiles). The seccomp options struct leaves room for future extensions, such as defining the
behavior when a profile cannot be set.

##### LocalhostProfile

This KEP proposes we GA LocalhostProfile as the only source of user-defined profiles at this point. 
User-defined profiles are essential for users to realize the full benefits out of seccomp, allowing 
them to decrease their attack surface based on their own workloads. 

###### Updating seccomp profiles
Seccomp profiles are applied at container creation time. Therefore, updating the file contents of a 
profile on disk after that point will not cause any changes to the running containers that are using 
it, but will apply the updated profile on container restart. Note that amending such files is not 
recommended and may cause containers to fail on next restart, in the case of the new profile being more 
restrictive, invalid or the file no longer being present on disk.

Currently, users have no way to tell whether their physical profiles have been deleted or modified.
This KEP proposes no changes to the existing functionality.

The recommended approach for rolling out changes to seccomp profiles is to always create _new profiles_ 
instead of updating existing ones. Create and deploy a new version of the existing Pod Template, changing the
profile name to the newly created profile. Redeploy, once working delete the former Pod Template. This 
will avoid disruption on in-flight workloads.

The current behavior lacks features to facilitate the maintenance of seccomp profiles across the cluster. 
Two examples being: 1) the lack of profile synchronization across nodes and 2) how difficult it can be to
identify that profiles have been changed on disk, after pods started using it.
However, given its current "pseudo-GA" state, we don't want to change it with this KEP. We will explore 
improvements to the behavior through the seccomp-operator and/or a new feature-gated improvement.

###### Profile files managed by the cluster admins 
The current support relies on profiles being saved as files in all cluster nodes where the pods using 
them may be scheduled. It is also the cluster admin's responsibility to ensure the files are correctly 
saved and synchronised across the all nodes. 

Cluster admins can build their own solutions to keep profiles in sync across nodes (e.g. 
[using daemonsets](https://gardener.cloud/050-tutorials/content/howto/secure-seccomp/)). Or use 
community driven projects (e.g. [seccomp config](https://github.com/UKHomeOffice/seccomp-config), 
[openshift's machine config operator](https://github.com/openshift/machine-config-operator), 
[seccomp operator](https://github.com/saschagrunert/seccomp-operator)) that focuses on solving similar 
problems.


##### RuntimeProfile

We propose maintaining the support to a single runtime profile, which will be defined by using the 
SeccompProfileRuntimeDefault SeccompProfileType. The reasons being:

- No changes to the current behavior. Users are currently not allowed to specify other runtime profiles.
The existing API server rejects runtime profile names that are different than `runtime/default`.  
The only exception being `docker/default` - for backwards compatibility. 
- Most runtimes only support the default profile, although the CRI is flexible enough to allow the kubelet 
to send other profile names. 
- Dockershim does not currently provide a way to pass other runtime profile names.
- Multiple runtime profiles has never been requested as a feature.

If built-in support for multiple runtime profiles is needed in the future, a new KEP will be created to 
cover its details. The implementation could be backwards compatible by creating a new profile type 
(i.e. `SeccompProfileRuntime`). 


## Design Details


### Failure and Fallback Strategy

There are different scenarios in which applying seccomp may fail, below are the ones we mapped 
and their outcome once this KEP is implemented:

| Scenario | API Server Result | Kubelet Result |
|--------------------------------------------------------------|------------------------------|----------------------------------------------------------------------------------------------|
| 1) Using custom or `runtime/default` profile when container runtime does not support seccomp. | Pod created | The outcome is container runtime dependent. In this scenario containers may 1) fail to start or 2) run normally without having its policies enforced. |
| 2) Using custom or `runtime/default` profile that restricts syscalls a container is trying to make. | Pod created | The outcome is workload and seccomp dependent. In this scenario containers may 1) fail to start, 2) misbehave or 3) log violations. |
| 3) Using custom profile that relies on unsupported or invalid seccomp actions (i.e. `SCPM_ACT_LOG` in versions earlier than 4.14 Linux kernel). | Pod created | Containers fail to start. Retry respecting RestartPolicy and back-off delay. |
| 4) Using custom profile that does not exist on the node's disk. | Pod created | Containers fail to start. Retry respecting RestartPolicy and back-off delay. |
| 5) Using a non supported runtime profile (i.e. `runtime/default-audit`). | Pod **not** created. | N/A |
| 6) Using localhost profile with invalid name (i.e. "/etc/profilename") | Pod **not** created. | N/A |

Scenario 2 is the expected behavior of using seccomp and it is included here for completeness.

Scenario 6 represents the case of failing the existing validation, which is defined at [Pod API](#pod-api).

### Seccomp root path configuration

The existing kubelet (alpha) flag `--seccomp-profile-root` allows for seccomp root path configuration.
This flag will be deprecated as of v1.19, and will be removed on v1.23.
The seccomp root path will then be derived from the kubelet root path, which is defined by `--root-dir`.
The current default value is `<root-dir>/seccomp`. This KEP will make the default behavior the only behavior.

### Version Skew Strategy

Because the API is currently represented as (mutable) annotations, care must be taken for migrating
to the API fields. The cases to consider are: pod create, pod update, PSP create, PSP update.

All API skew is resolved in the API server. New Kubelets will only use the seccomp values specified
in the fields, and ignore the annotations.

#### Pod Creation

If no seccomp annotations or fields are specified, no action is necessary.

If _only_ seccomp fields are specified, add the corresponding annotations. This ensures that the
fields are enforced even if the node version trails the API version (see [Upgrade /
Downgrade](#upgrade--downgrade))

If _only_ seccomp annotations are specified, copy the values into the corresponding fields. This
ensures that existing applications continue to enforce seccomp, and prevents the kubelet from
needing to resolve annotations & fields.

If both seccomp annotations _and_ fields are specified, the values MUST match. This will be enforced
in API validation.

To raise awareness of annotation usage (in case of old automation), a warning mechanism will be used
to highlight that support will be dropped in v1.23.
The mechanisms being considerated are audit annotations, annotations on the object, events, or a 
warning as described in [KEP #1693](/keps/sig-api-machinery/1693-warnings).

#### PodSecurityPolicy Enforcement

The PodSecurityPolicy admission controller must continue to check the PSP object for annotations, as
well as for fields.

When setting default profiles, PSP only needs to set the field. The API machinery will handle
setting the annotation as necessary.

When enforcing allowed profiles, the PSP should check BOTH the annotations & fields. In most cases,
they should be consistent. On pod update, the seccomp annotations may differ from the fields. In
that case, the PSP enforcement should check both values as the effective value depends on the node
version running the pod.

#### Pod Update

The seccomp fields on a pod are immutable.

The behavior on annotation update is currently ill-defined: the annotation update is allowed, but
the new value will not be used until the container is restarted. There is no way to tell (from the
API) what value a container is using.

Therefore, seccomp annotation updates will be ignored. This maintains backwards API compatibility
(no tightening validation), and makes a small stabilizing change to behavior (new Kubelets will
ignore the update).

When an [Ephemeral Container](277-ephemeral-containers) is added, it will follow the same
rules for using or overriding the pod's seccomp profile. Ephemeral container's will never sync with
a seccomp annotation.

#### PodTemplates

PodTemplates (e.g. ReplaceSets, Deployments, StatefulSets, etc.) will be ignored. The
field/annotation resolution will happen on template instantiation.

To raise awareness of existing controllers using the seccomp annotations that need to be migrated, 
a warning mechanism will be used to highlight that support will be dropped in v1.23.

The mechanisms being considerated are audit annotations, annotations on the object, events, or a 
warning as described in [KEP #1693](/keps/sig-api-machinery/1693-warnings).

#### Runtime Profiles 

The API Server will continue to reject annotations with runtime profiles different than `runtime/default`, 
to maintain the existing behavior. 

Violations would lead to the error message:
```
Invalid value: "runtime/profile-name": must be a valid seccomp profile
```

#### Kubelet Backwards compatibility

The changes brought to the Kubelet by this KEP will ensure backwards compatibility in a similar 
way the changes above define it at API Server level. Therefore, the seccomp profiles will be applied 
following the priority order:

1. Container-specific field.
2. Container-specific annotation.
3. Pod-wide field.
4. Pod-wide annotation.

In case annotations and fields at either container or pod level exist, the kubelet will ignore the
annotations and will only apply the profile defined on the relevant field.

#### Upgrade / Downgrade

Nodes do not currently support in-place upgrades, so pods will be recreated on node upgrade and
downgrade. No special handling or consideration is needed to support this.

On the API server side, we've already taken version skew in HA clusters into account. The same
precautions make upgrade & downgrade handling a non-issue.

Since [we support](https://kubernetes.io/docs/setup/release/version-skew-policy/) up to 2 minor
releases of version skew between the master and node, annotations must continue to be supported and
backfilled for at least 2 versions passed the initial implementation. However, we can decide to
extend support farther to reduce breakage. If this feature is implemented in v1.19, I propose v1.23
as a target for removal of the old behavior.

### Test Plan

Seccomp already has [E2E tests][], but the tests are guarded by the `[Feature:Seccomp]` tag and not
run in the standard test suites.

Prior to being marked GA, the feature tag will be removed from the seccomp tests, and the tests will
be migrated to the new fields API. Tests will be tagged as `[LinuxOnly]`.

New tests will be added covering the annotation/field conflict cases described under
[Version Skew Strategy](#version-skew-strategy).

Test coverage for localhost profiles will be added, unless we decide to [keep localhost support in
alpha](#alternatives).

[E2E tests]: https://github.com/kubernetes/kubernetes/blob/5db091dde4d7de74283ca94870958acf63010c0a/test/e2e/node/security_context.go#L147

### Graduation Criteria

_This section is excluded, as it is the subject of the entire proposal._

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable, can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/20190731-production-readiness-review-process.md

Production readiness review questionnaire must be completed for features in
v1.19 or later, but is non-blocking at this time. That is, approval is not
required in order to be in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.

-->

### Feature enablement and rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  
  This feature was implemented prior to the concept of feature gates. Meaning, it is already enabled 
  in live clusters and cannot be disabled. This KEP is simply "cleaning up" the API to make it GA.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**

  This feature (seccomp) cannot be disabled, as it is currently in a "pseudo-GA" state. 
  However, the changes it brings are backwards compatible, and the API supports rollback 
  of the kubernetes apiserver as described in the [Version Skew Strategy](#version-skew-strategy).

* **What happens if we reenable the feature if it was previously rolled back?**
  
  N/A - the feature is already enabled by default since Kubernetes 1.3.

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling and disabling feature
  gates. However, unit tests in each component dealing with managing data created
  with and without the feature are necessary. At the very least, think about
  conversion tests if API types are being modified.

  N/A - the feature is already enabled by default since Kubernetes 1.3.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g. what if some components will restart
  in the middle of rollout?

  The [Version Skew Strategy](#version-skew-strategy) section covers this point.
  Running workloads should have no impact as the Kubelet will support either the 
  existing annotations or the new fields introduced by this KEP. 

* **What specific metrics should inform a rollback?**

  N/A - the feature is already enabled by default since Kubernetes 1.3.

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and do that now.

  Automated tests will cover the scenarios with and without the changes proposed 
  on this KEP. As defined under [Version Skew Strategy](#version-skew-strategy),
  we are assuming the cluster may have kubelets with older versions (without 
  this KEP' changes), therefore this will be covered as part of the new tests.

### Monitoring requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metrics. Operations against Kubernetes API (e.g.
  checking if there are objects with field X set) may be last resort. Avoid
  logs or events for this purpose.

  The feature is built into the kubelet and api server components. No metric is
  planned at this moment. The way to determine usage is by checking whether the 
  pods/containers have a SeccompProfile set.

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

  N/A

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At the high-level this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide a comprehensive guidance, but at the very
  high level (they needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

  N/A

* **Are there any missing metrics that would be useful to have to improve
  observability in this feature?**
  Describe the metrics themselves and the reason they weren't added (e.g. cost,
  implementation difficulties, etc.).

  N/A

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  Think about both cluster-level services (e.g. metrics-server) as well
  as node-level agents (e.g. specific version of CRI). Focus on external or
  optional services that are needed. For example, if this feature depends on
  a cloud provider API, or upon an external software-defined storage or network
  control plane.

  For each of the fill in the following, thinking both about running user workloads
  and creating new ones, as well as about cluster-level services (e.g. DNS):
  
  This KEP adds no new dependencies.


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirms the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**

  NO

* **Will enabling / using this feature result in introducing new API types?**

  NO

* **Will enabling / using this feature result in any new calls to cloud
  provider?**

  NO

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**

  NO

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**

  NO

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data send and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits][].

  NO

### Troubleshooting

Troubleshooting section serves the `Playbook` role as of now. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now we leave it here though.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

  This is integral part of both API server and the Kubelet. All their dependencies
  will impact 

* **What are other known failure modes?**

  No impact is being foreseen to running workloads based on the nature of 
  changes brought by this KEP.
  
  Although some general errors and failures can be seen on [Failure and Fallback Strategy](#failure-and-fallback-strategy). 


* **What steps should be taken if SLOs are not being met to determine the problem?**

  N/A

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2019-07-17: Initial KEP
- 2020-05-07: 
  - Removed field RuntimeProfile from SeccompProfile. 
  - Removed field RuntimeProfiles from SeccompProfileSet.
  - Add validation details for LocalhostProfile.
  - Validation for runtime profiles being migrated.
  - Add Failure and Fallback Strategy.
- 2020-05-12: Add PRR Questionare

## Drawbacks

Promoting seccomp as-is to GA may be seen as "blessing" the current functionality, and make it
harder to make some of the enhancements listed under [Non-Goals](#non-goals). Since the current
behavior is unguarded, I think we already need to treat the behavior as GA (which is why it's been
so hard to change the default profile), so I do not think these changes will actually increase the
friction.

## Alternatives

### Localhost profiles
The localhost feature currently depends on an alpha Kubelet flag. We could therefore label the
localhostProfile source as an alpha field, and keep it's functionality in an alpha state.


### Updating PodSecurityPolicy API

#### PodSecurityPolicy API

```go
type PodSecurityPolicySpec struct {
    ...
    // seccomp is the strategy that will dictate allowable and default seccomp
    // profiles for the pod.
    // +optional
    Seccomp *SeccompStrategyOptions
    ...
}

type SeccompStrategyOptions struct {
    // The default profile to set on the pod, if none is specified.
    // The default MUST be allowed by the allowedProfiles.
    // +optional
    DefaultProfile *v1.SeccompProfile

    // The set of profiles that may be set on the pod or containers.
    // If unspecified, seccomp profiles are unrestricted by this policy.
    // +optional
    AllowedProfiles *SeccompProfileSet
}

// A set of seccomp profiles. This struct should be a plural of v1.SeccompProfile.
// All values are optional, and an unspecified field excludes all profiles of
// that type from the set.
type SeccompProfileSet struct {
    // The allowed seccomp profile types.
    // +optional
    Types []SeccompProfileType
    // The allowed localhostProfiles. Values may end in '*' to include all
    // localhostProfiles with a prefix.
    // +optional
    LocalhostProfiles []string
}
```

#### PodSecurityPolicy Creation

Unlike with pods, PodSecurityPolicy seccomp annotations and fields are _not_ synced.

If only seccomp annotations or fields are specified, no action is necessary. The set value is used
when applying the PodSecurityPolicy.

If both seccomp annotations _and_ fields are specified, the values MUST match. This will be enforced
in API validation.

#### PodSecurityPolicy Update

PodSecurityPolicy seccomp fields are mutable. On an update, the same rules are applied as for
creation, ignoring the old values.

If only seccomp annotations or fields are specified in the updated PSP, no action is necessary, and
the specified values are used.

If both seccomp annotations _and_ fields are specified in the updated PSP, the values MUST match.

## References

- [Original seccomp proposal](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/seccomp.md)
