# KEP-2413: Enable seccomp by default

<!-- toc -->

- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha to Beta Graduation](#alpha-to-beta-graduation)
    - [Beta to GA Graduation](#beta-to-ga-graduation)
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
  - [Alternative 1: Define a new <code>KubernetesDefault</code> profile](#alternative-1-define-a-new--profile)
  - [Alternative 2: Allow admins to pick one of <code>KubernetesDefault</code>, <code>RuntimeDefault</code> or a custom profile](#alternative-2-allow-admins-to-pick-one-of---or-a-custom-profile)
  <!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required _prior to targeting to a milestone / release_.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
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

Enable seccomp by default for all workloads running on Kubernetes to improve the
default security of the overall system.

## Motivation

Kubernetes provides a native way to specify seccomp profiles for workloads,
which is disabled by default today. Seccomp adds a layer of security that could
help prevent CVEs or 0-days if enabled by default. If we enable seccomp by
default, we make implicitly Kubernetes more secure.

### Goals

Provide a way to enable seccomp by default for Kubernetes.

### Non-Goals

Everything else related to the feature.

## Proposal

We introduce a feature gate that enables a seccomp for all workloads by default.
There are a few options for what should be the default seccomp profile.

The most preferred solution is to promote the `RuntimeDefault` profile
(previously the `runtime/default` annotation) to the new default one.

Container runtimes already have their own defined default profiles, which get
referenced via the `RuntimeDefault` one. This means we now promote this profile
to the new default. Every workload created will then get the `RuntimeDefault`
(`SeccompProfileTypeRuntimeDefault`) as `SeccompProfile.type` value for the
`PodSecurityContext` as well as the `SecurityContext` for every container.

The advantages of using the `RuntimeDefault` profiles are that there is no need
for shipping an additional seccomp profile. The overall version skew handling in
conjunction with runtime versions and the kubelet is easier, because container
runtimes already support `RuntimeDefault` from the introduction of the seccomp
feature.

For alternative proposals please refer to the [Alternatives
section](#alternatives).

### User Stories

As a Kubernetes admin, I want to ensure that my cluster is secure by default
without relying on workloads opting to use seccomp.

### Risks and Mitigations

Some workloads that may be running without seccomp may break with seccomp
enabled by default. All workloads would have to be tested in a staging/test
environment to ensure there are no breakages. Seccomp could either be turned off
or custom profiles could be created for such workloads.

The configuration possibilities of container runtimes differ in conjunction with
seccomp. For example:

- **containerd**
  - can only use the internal default profile for `RuntimeDefault`
  - can use a different profile for empty (unconfined) workloads via the
    `unset_seccomp_profile` option
- **CRI-O**
  - can specify a different `RuntimeDefault` profile via the `seccomp_profile`
    option
  - can use `RuntimeDefault` for empty (unconfined) workloads via
    `seccomp_use_default_when_empty`

This can result in a behavioral change when doing cluster upgrades while runtime
administrates may have to take action if they enable the feature.

## Design Details

The feature gate `SeccompDefault` will ensure that the API graduation can be
done in the standard Kubernetes way. The implementation will be mainly a switch
from `Unconfined` to `RuntimeDefault`.

Documentation around the feature will be added to the [k/website seccomp
section](https://kubernetes.io/docs/tutorials/clusters/seccomp).

### Test Plan

There will be unit tests for the feature, whereas the existing seccomp tests can
be extended to cover the new behavior if enabled.

### Graduation Criteria

#### Alpha

- [ ] Implement the new feature gate
- [ ] Ensure proper tests are in place
- [ ] Update documentation to make the feature visible

#### Alpha to Beta Graduation

- [ ] Enable the feature per default
- [ ] No major bugs reported in the previous cycle

#### Beta to GA Graduation

- [ ] Allowing time for feedback (3 releases)
- [ ] Risks have been addressed by every common container runtime

### Upgrade / Downgrade Strategy

It's recommended to test the existing workloads with the `RuntimeDefault`
profile before turning the feature on. Beside that, the feature can be enabled
on a per node basis to reduce the risk of failing production workloads.

### Version Skew Strategy

There is no explicit version skew strategy required because the feature acts as
a toggle switch.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

- **How can this feature be enabled / disabled in a live cluster?**

  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `SeccompDefault`
    - Components depending on the feature gate: `kubelet`

- **Does enabling the feature change any default behavior?**

  Yes, it will change the `Unconfined` seccomp profile to `RuntimeDefault` if
  no profile is specified.

- **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**

  Yes, the feature can be disabled but workloads have to be restarted to apply
  the previous behavior.

- **What happens if we reenable the feature if it was previously rolled back?**

  It will enable the feature again but only apply the new profile to new/restarted
  workloads.

- **Are there any tests for feature enablement/disablement?**

  Yes, the behavior can be tested via unit tests.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

- **How can a rollout fail? Can it impact already running workloads?**

  Workloads on a node may starting to fail when (re)scheduled on the node where
  the feature is enabled. Required specific syscalls may be blocked by the
  default seccomp profile, which will cause the application to get terminated.

- **What specific metrics should inform a rollback?**

  If a workload is starting to fail because of blocked syscalls (audit logs),
  then a temporarily rollback would be appropriate in production.

- **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

  If we assume that enabling the feature will cause workloads to fail, then
  there are three possible mitigations available:

  1. Disable the feature on the node (downgrade):
     permanent mitigation
  2. Run the workload as `Unconfined` (the previous default):
     re-enabling possible
  3. Create a custom seccomp profile for the application (recommended):
     re-enabling possible

- **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
  fields of API types, flags, etc.?**

  No

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

- **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

- **What are the SLIs (Service Level Indicators) an operator can use to determine
  the health of the service?**

  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

- **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:

  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

- **Are there any missing metrics that would be useful to have to improve observability
  of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

- **Does this feature depend on any specific services running in the cluster?**
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

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

- **Will enabling / using this feature result in any new API calls?**

  No

- **Will enabling / using this feature result in introducing new API types?**

  No

- **Will enabling / using this feature result in any new calls to the cloud
  provider?**

  No

- **Will enabling / using this feature result in increasing size or count of
  the existing API objects?**

  No

- **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs]?**

  No

- **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**

  Enabling a seccomp profile for a workload will take more time compared to not
  applying a profile at all. There is also a very low overhead for checking the
  syscalls within the Linux Kernel.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

- **How does this feature react if the API server and/or etcd is unavailable?**

- **What are other known failure modes?**
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

- **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing slis/slos]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2021-05-05: KEP promoted to implementable

## Alternatives

There are multiple alternatives to the proposed approach.

### Alternative 1: Define a new `KubernetesDefault` profile

Kubernetes ships a default seccomp profile `KubernetesDefault`
(`SeccompProfileTypeKubernetesDefault`), which is the new default
`SeccompProfile.type` value for the `PodSecurityContext` as well as the
`SecurityContext` for every container.

On startup of the kubelet, it will place the default seccomp profile JSON in a
pre-defined path on the host machine. The container runtime has to verify the
existence of this profile and apply it to the container.

We could pass the information where the default profile resides on disk via the
CRI. This way we can change the path from a kubelet perspective. If the field
is empty, then we assume that the kubelet does not support or has disabled the
feature at all. This means we fallback to the currently implemented "unconfined
when not set" behavior.

A possible starting point to defining this profile is to look at docker,
containerd and cri-o default profiles.

The advantages of defining a `KubernetesDefault` profile are:

- The Kubernetes community / SIG Node owns the profile and is able to
  improve/change it without depending on multiple container runtimes.
- Increased transparency and a more uniform documentation around the feature.
- Users still can use the `SeccompProfileTypeRuntimeDefault` and will not
  encounter any changes to their workloads, even if they turn on the feature.

### Alternative 2: Allow admins to pick one of `KubernetesDefault`, `RuntimeDefault` or a custom profile

This is a combination of alternatives 1 and 2, which allows the highest amount
of flexibility. Users have the chance to either use the `KubernetesDefault`,
`RuntimeDefault` or configure a custom seccomp profile path directly at the
kubelet level. This also implies that the kubelet has to additionally pre-check
if the profile exists and is valid during its startup.
