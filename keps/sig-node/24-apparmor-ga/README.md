# AppArmor graduation to General Availability (GA)

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
      - [Localhost Profile](#localhost-profile)
      - [RuntimeDefault Profile](#runtimedefault-profile)
- [Design Details](#design-details)
  - [Failure and Fallback Strategy](#failure-and-fallback-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
    - [Pod Creation](#pod-creation)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required _prior to targeting to a milestone / release_.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in
      [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and
      SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for
      publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to
      mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once
it is marked `implementable` should be approved by each of the KEP approvers. If
any of those approvers is no longer appropriate than changes to that list should
be approved by the remaining approvers and/or the owning SIG (or SIG-arch for
cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every
time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

This is a proposal to upgrade the AppArmor annotation on pods to a dedicated
field, and graduate the feature to GA. This proposal aims to do the _bare
minimum_ to clean up the feature, without blocking future enhancements.

## Motivation

AppArmor support has been added with Kubernetes v1.4 and is already in beta.
Profiles have to be available on each node whereas the container runtime ensures
that the profile is loaded when specified at pod or PSP level. Profiles can be
specified per-container via the pod’s metadata annotation:

```
container.apparmor.security.beta.kubernetes.io/<container_name>: {unconfined,runtime/default,localhost/<profile>}
```

The feature has been more or less unchanged ever since. The main motivation
behind this KEP is to promote the AppArmor feature to GA.

_NOTE: Seccomp was in a very similar state, but with some subtle differences.
Promoting Seccomp to GA will be covered by a [separate
KEP](/keps/sig-node/20190717-seccomp-ga.md)._

### Goals

- Declare AppArmor as GA
- Fully document and formally spec the feature support
- Add equivalent API fields to replace AppArmor annotations and provide a pod
  level field, which applies to all containers.
- Deprecate the AppArmor annotations

### Non-Goals

This KEP proposes the absolute minimum to get AppArmor to GA, therefore all
functional enhancements are out of scope, including:

- Defining any standard "Kubernetes branded" AppArmor profiles
- Formally specifying the AppArmor profile format in Kubernetes
- Providing mechanisms for loading profiles from outside of the node
- Changing the semantics around AppArmor support
- Windows support

## Proposal

AppArmor is not available on every Linux distribution. Beside this, container
runtimes have AppArmor as compile-time feature which may be disabled as well.
With the GA API we do not change the error handling and behave exactly the same
as the current error propagation paths.

### API

The AppArmor API will be functionally equivalent to the current beta API, with
the enhancement of adding pod level profiles to match the behavior with seccomp.
This includes the Pod API, which specifies what profile the containers run with.

#### Pod API

The Pod AppArmor API is generally immutable, except in `PodTemplates`.

```go
type PodSecurityContext struct {
    ...
    // The AppArmor options to use by the containers in this pod.
    // +optional
    AppArmor  *AppArmorProfile
    ...
}

type SecurityContext struct {
    ...
    // The AppArmor options to use by this container. If AppArmor options are
    // provided at both the pod & container level, the container options
    // override the pod options.
    // +optional
    AppArmor  *AppArmorProfile
    ...
}

// Only one profile source may be set.
// +union
type AppArmorProfile struct {
    // +unionDescriminator
    Type AppArmorProfileType

    // LocalhostProfile indicates a loaded profile on the node that should be used.
    // The profile must be preconfigured on the node to work.
    // Must match the loaded name of the profile.
    // Must only be set if type is "Localhost".
    // +optional
    LocalhostProfile *string
}

type AppArmorProfileType string

const (
    AppArmorProfileTypeUnconfined     AppArmorProfileType = "Unconfined"
    AppArmorProfileTypeRuntimeDefault AppArmorProfileType = "RuntimeDefault"
    AppArmorProfileTypeLocalhost      AppArmorProfileType = "Localhost"
)
```

This API makes the options more explicit and leaves room for new profile sources
to be added in the future (e.g. Kubernetes predefined profiles or ConfigMap
profiles) and for future extensions, such as defining the behavior when a
profile cannot be set.

##### Localhost Profile

This KEP proposes we GA LocalhostProfile as the only source of user-defined
profiles at this point. User-defined profiles are essential for users to realize
the full benefits out of AppArmor, allowing them to decrease their attack
surface based on their own workloads.

###### Updating AppArmor profiles

AppArmor profiles are applied at container creation time. The underlying
container runtime only references already loaded profiles by its name.
Therefore, updating the profiles content requires a manual reload via
`apparmor_parser`.

Note that changing profiles is not recommended and may cause containers to fail
on next restart, in the case of the new profile being more restrictive, invalid
or the file no longer available on the host.

Currently, users have no way to tell whether their physical profiles have been
deleted or modified. This KEP proposes no changes to the existing functionality.

The recommended approach for rolling out changes to AppArmor profiles is to
always create _new profiles_ instead of updating existing ones. Create and
deploy a new version of the existing Pod Template, changing the profile name to
the newly created profile. Redeploy, once working delete the former Pod
Template. This will avoid disruption on in-flight workloads.

The current behavior lacks features to facilitate the maintenance of AppArmor
profiles across the cluster. Two examples being: 1) the lack of profile
synchronization across nodes and 2) how difficult it can be to identify that
profiles have been changed on disk/memory, after pods started using it. However,
given its current "pseudo-GA" state, we don't want to change it with this KEP.
Out of tree enhancements like the
[security-profiles-operator](https://github.com/kubernetes-sigs/security-profiles-operator) can
provide such enhanced functionality on top.

###### Profiles managed by the cluster admins

The current support relies on profiles being loaded on all cluster nodes
where the pods using them may be scheduled. It is also the cluster admin's
responsibility to ensure the profiles are correctly saved and synchronised
across the all nodes. Existing mechanisms like node `labels` and `nodeSelectors`
can be used to ensure that pods are scheduled on nodes supporting their desired
profiles.

##### RuntimeDefault Profile

We propose maintaining the support to a single runtime profile, which will be
defined by using the `AppArmorProfileTypeRuntimeDefault`. The reasons being:

- No changes to the current behavior. Users are currently not allowed to specify
  other runtime profiles. The existing API server rejects runtime profile names
  that are different than `runtime/default`.
- Most runtimes only support the default profile, although the CRI is flexible
  enough to allow the kubelet to send other profile names.
- Dockershim does not currently provide a way to pass other runtime profile
  names.
- Multiple runtime profiles has never been requested as a feature.

If built-in support for multiple runtime profiles is needed in the future, a new
KEP will be created to cover its details.

## Design Details

### Failure and Fallback Strategy

There are different scenarios in which applying an AppArmor profile may fail,
below are the ones we mapped and their outcome once this KEP is implemented:

| Scenario                                                                                           | API Server Result    | Kubelet Result                                                                                                                                        |
| -------------------------------------------------------------------------------------------------- | -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1) Using custom or `runtime/default` profile when container runtime does not support AppArmor.     | Pod created          | The outcome is container runtime dependent. In this scenario containers may 1) fail to start or 2) run normally without having its policies enforced. |
| 2) Using custom or `runtime/default` profile that restricts actions a container is trying to make. | Pod created          | The outcome is workload and AppArmor dependent. In this scenario containers may 1) fail to start, 2) misbehave or 3) log violations.                  |
| 3) Using custom profile that does not exist on the node.                                           | Pod created          | Containers fail to start. Retry respecting RestartPolicy and back-off delay.                                                                          |
| 4) Using an unsupported runtime profile (i.e. `runtime/default-audit`).                            | Pod **not** created. | N/A                                                                                                                                                   |
| 5) Using localhost profile with invalid name                                                       | Pod **not** created. | N/A                                                                                                                                                   |
| 6) AppArmor is disabled by the host or the build                                                   | Pod **not** created. | Kubelet puts Pod in blocked state.                                                                                                                    |

Scenario 2 is the expected behavior of using AppArmor and it is included here
for completeness.

Scenario 5 represents the case of failing the existing validation, which is
defined at [Pod API](#pod-api).

### Version Skew Strategy

All API skew is resolved in the API server. New Kubelets will only use the
AppArmor values specified in the fields, and ignore the annotations.

#### Pod Creation

If no AppArmor annotations or fields are specified, no action is necessary.

If the `AppArmor` feature is disabled per feature gate, then the annotations and
fields are cleared (current behavior).

If _only_ AppArmor fields are specified, add the corresponding annotations. This
ensures that the fields are enforced even if the node version trails the API
version (see [Version Skew Strategy](##version-skew-strategy)).

If _only_ AppArmor annotations are specified, copy the values into the
corresponding fields. This ensures that existing applications continue to
enforce AppArmor, and prevents the kubelet from needing to resolve annotations &
fields. If the annotation is empty, then the `runtime/default` profile will be
used by the CRI container runtime. If a localhost profile is specified, then
container runtimes will strip the `localhost/` prefix, too. This will be covered
by e2e tests during the GA promotion.

If both AppArmor annotations _and_ fields are specified, the values MUST match.
This will be enforced in API validation.

If a Pod with a container specifies an AppArmor profile by field/annotation,
then the container will only apply the Pod level field/annotation if no none
are set on the container level.

To raise awareness of annotation usage (in case of old automation), a warning
mechanism will be used to highlight that support will be dropped in v1.24.
The mechanisms being considered are audit annotations, annotations on the
object, events, or a warning as described in [KEP
#1693](/keps/sig-api-machinery/1693-warnings).

#### Pod Update

The AppArmor fields on a pod are immutable, which also applies to the
[annotation](https://github.com/kubernetes/kubernetes/blob/b46612a74224b0871a97dae819f5fb3a1763d0b9/pkg/apis/core/validation/validation.go#L177-L182).

When an [Ephemeral Container](20190212-ephemeral-containers.md) is added, it
will follow the same rules for using or overriding the pod's AppArmor profile.
Ephemeral container's will never sync with an AppArmor annotation.

#### PodTemplates

PodTemplates (e.g. ReplaceSets, Deployments, StatefulSets, etc.) will be
ignored. The field/annotation resolution will happen on template instantiation.

To raise awareness of existing controllers using the AppArmor annotations that
need to be migrated, a warning mechanism will be used to highlight that support
will be dropped in v1.24.

The mechanisms being considered are audit annotations, annotations on the
object, events, or a warning as described in [KEP
#1693](/keps/sig-api-machinery/1693-warnings).

#### Runtime Profiles

The API Server will continue to reject annotations with runtime profiles
different than `runtime/default`, to maintain the existing behavior.

Violations would lead to the error message:

```
Invalid value: "runtime/profile-name": must be a valid AppArmor profile
```

#### Kubelet Backwards compatibility

The changes brought to the Kubelet by this KEP will ensure backwards
compatibility in a similar way the changes above define it at API Server level.
Therefore, the AppArmor profiles will be applied following the priority order:

1. Container-specific field.
2. Container-specific annotation.
3. Pod-wide field.

In case annotations and fields at either container or pod level exist, the
kubelet will ignore the annotations and will only apply the profile defined on
the relevant field.

#### Upgrade / Downgrade

Nodes do not currently support in-place upgrades, so pods will be recreated on
node upgrade and downgrade. No special handling or consideration is needed to
support this.

On the API server side, we've already taken version skew in HA clusters into
account. The same precautions make upgrade & downgrade handling a non-issue.

Since
[we support](https://kubernetes.io/docs/setup/release/version-skew-policy/) up
to 2 minor releases of version skew between the master and node, annotations
must continue to be supported and backfilled for at least 2 versions passed the
initial implementation. However, we can decide to extend support farther to
reduce breakage. If this feature is implemented in v1.20, I propose v1.24 as a
target for removal of the old behavior.

### Test Plan

AppArmor already has [e2e tests][https://github.com/kubernetes/kubernetes/blob/6596a14/test/e2e_node/apparmor_test.go],
but the tests are guarded by the `[Feature:AppArmor]` tag and not run in the
standard test suites.

Tests will be tagged as `[Feature:AppArmor]` like it is implemented right now,
but they will be migrated to use the new fields API.

New tests will be added covering the annotation/field conflict cases described
under [Version Skew Strategy](#version-skew-strategy).

Test coverage for localhost profiles will be added, unless we decide to [keep
localhost support in alpha](#alternatives).

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

- **How can this feature be enabled / disabled in a live cluster?**

  The feature can be enabled/disabled by the `AppArmor` feature gate. This
  feature gate can be used to disable the feature until it reaches GA.

- **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**

  Yes, it works in the same way as before moving the feature to GA. However, the
  GA related changes are backwards compatible, and the API supports rollback of
  the Kubernetes API Server as described in the [Version Skew
  Strategy](#version-skew-strategy).

- **What happens if we reenable the feature if it was previously rolled back?**

  N/A - the feature is already enabled by default since Kubernetes v1.4.

- **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling and disabling feature
  gates. However, unit tests in each component dealing with managing data created
  with and without the feature are necessary. At the very least, think about
  conversion tests if API types are being modified.

  N/A - the feature is already enabled by default since Kubernetes v1.4.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

- **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g. what if some components will restart
  in the middle of rollout?

  The [Version Skew Strategy](#version-skew-strategy) section covers this point.
  Running workloads should have no impact as the Kubelet will support either the
  existing annotations or the new fields introduced by this KEP.

- **What specific metrics should inform a rollback?**

  N/A - the feature is already enabled by default since Kubernetes v1.4.

- **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and do that now.

  Automated tests will cover the scenarios with and without the changes proposed
  on this KEP. As defined under [Version Skew Strategy](#version-skew-strategy),
  we are assuming the cluster may have kubelets with older versions (without
  this KEP' changes), therefore this will be covered as part of the new tests.

### Monitoring requirements

_This section must be completed when targeting beta graduation to a release._

- **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metrics. Operations against Kubernetes API (e.g.
  checking if there are objects with field X set) may be last resort. Avoid
  logs or events for this purpose.

  The feature is built into the kubelet and api server components. No metric is
  planned at this moment. The way to determine usage is by checking whether the
  pods/containers have a AppArmorProfile set.

- **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**

  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

  N/A

- **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At the high-level this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide a comprehensive guidance, but at the
  very high level (they needs more precise definitions) those may be things
  like:

  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

  N/A

- **Are there any missing metrics that would be useful to have to improve
  observability in this feature?**
  Describe the metrics themselves and the reason they weren't added (e.g. cost,
  implementation difficulties, etc.).

  N/A

### Dependencies

_This section must be completed when targeting beta graduation to a release._

- **Does this feature depend on any specific services running in the cluster?**
  Think about both cluster-level services (e.g. metrics-server) as well
  as node-level agents (e.g. specific version of CRI). Focus on external or
  optional services that are needed. For example, if this feature depends on
  a cloud provider API, or upon an external software-defined storage or network
  control plane.

  For each of the fill in the following, thinking both about running user
  workloads and creating new ones, as well as about cluster-level services (e.g.
  DNS):

  This KEP adds no new dependencies.

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirms the
previous answers based on experience in the field._

- **Will enabling / using this feature result in any new API calls?**

  NO

- **Will enabling / using this feature result in introducing new API types?**

  NO

- **Will enabling / using this feature result in any new calls to cloud
  provider?**

  NO

- **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**

  NO

- **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**

  NO

- **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data send and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits][].

  NO

### Troubleshooting

Troubleshooting section serves the `Playbook` role as of now. We may consider
splitting it into a dedicated `Playbook` document (potentially with some
monitoring details). For now we leave it here though.

_This section must be completed when targeting beta graduation to a release._

- **How does this feature react if the API server and/or etcd is unavailable?**

  This is integral part of both API server and the Kubelet. All their
  dependencies will impact

- **What are other known failure modes?**

  No impact is being foreseen to running workloads based on the nature of
  changes brought by this KEP.

  Although some general errors and failures can be seen on [Failure and Fallback
  Strategy](#failure-and-fallback-strategy).

* **What steps should be taken if SLOs are not being met to determine the problem?**

  N/A

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing slis/slos]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2020-01-10: Initial KEP
- 2020-08-24: Major rework and sync with seccomp
- 2021-04-25: PSP mentions

## Drawbacks

Promoting AppArmor as-is to GA may be seen as "blessing" the current
functionality, and make it harder to make some of the enhancements listed under
[Non-Goals](#non-goals). Since the current behavior is unguarded, I think we
already need to treat the behavior as GA.
