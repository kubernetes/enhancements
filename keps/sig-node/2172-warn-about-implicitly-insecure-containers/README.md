# KEP-2172: Warn about implicitly insecure running containers

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Report running user and group IDs](#report-running-user-and-group-ids)
  - [Condition when running implicitly-insecure](#condition-when-running-implicitly-insecure)
  - [Events when running implicitly-insecure](#events-when-running-implicitly-insecure)
  - [CLI warnings when running implicitly-insecure](#cli-warnings-when-running-implicitly-insecure)
    - [Color](#color)
  - [Metrics](#metrics)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; GA Graduation](#alpha---ga-graduation)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubernetes does not make it hard enough to do the wrong thing.  Specifically,
there are things that users do ALL THE TIME that they really should not, such
as running containers as root.  This KEP aims to make it more obvious that they
are doing that, and to encourage them to declaring their intent better.

## Motivation

Recent CVEs around containerd come down to containers running as root when they
shouldn't.  After discussion, I realized many users simply don't know they are
doing anything wrong.  It's just not obvious.

Many container images do not opt-in to a non-root UID.  Users run these
containers without specifying a non-root `runAsUser`.  Kubernetes happily runs
the container as root, whether it needs root or not.

For clarity, we define the following terms:

implicitly-insecure: containers which run as UID or GID 0 and do not set
`runAsUser` or `runAsGroup` to 0.

explicitly-insecure: containers which run as UID or GID 0 but set
`runAsUser` or `runAsGroup` to 0.

### Goals

1) To make it more obvious when a pod has implicitly-insecure
   containers.
2) To subtly suggest to users that running as root is an error.

### Non-Goals

1) To make it impossible or opt-in to run as root.
2) To actively impact users who run as root.
3) To make noise about explicitly-insecure containers.

## Proposal

This proposal includes several parts.

### Report running user and group IDs

This KEP proposes to add 2 fields - `userID` and `groupID` to
`Pod.status.containerStatuses` to report the UID and GID of the root process
for each container (as found in `runtimeSpec.process.user`).

### Condition when running implicitly-insecure

Whenever kubelet sees a container running as user or group 0, it will add a
condition - "InsecureUserID" or "InsecureGroupID" respectively - to the pod.
Users who see these can look at the `Pod.status.containerStatuses[]` for more
details.

These conditions will be bypased if the user explicitly sets `runAsUser` or
`runAsGroup` in their pod.

### Events when running implicitly-insecure

Whenever kubelet sees a container running as user or group 0, it will creat a
kubernetes Event object warning the user.  These events will be logged no more
often than once per hour per pod.

These events will be bypased if the user explicitly sets `runAsUser` or
`runAsGroup` in their pod.

### CLI warnings when running implicitly-insecure

Kubectl will add fields in `describe pod` to report UID/GID and will flag when
those are 0, without respective `runAs...` fields.

#### Color

If possible, kubectl will detect whether it is printing to a console or not,
and if so it will color pods that are running as root in red.  For example,
`kubectl get pods` would highlight problematic pods.

### Metrics

Kubelet will add metrics indicating the number of pods with
implicitly-insecure and explicitly-insecure containers.

### User Stories (Optional)

#### Story 1

Catie the cluster admin can run `kubectl get pods --all-namespaces` and quickly
see any pods that are implicitly-insecure.

#### Story 2

Pete the platform admin can track the new metrics and set alerts when they
become non-zero.  He can investigate and ask users to set a specific
`runAs...` or to set it 0.  Pete can also install admission controllers to only
allow approved users to set the `runAs...` fields to 0.

#### Story 3

Usain the user will constantly see the red lines and ominous-sounding conditions
when they examine their workloads, and will eventually choose to make them go
away by running as non-root.

### Risks and Mitigations

Realistically, many users will simply set `runAs...` to 0, but I still think of
that as a win because a) they thought about it and b) they are being explicit
about it.

There's a risk that all these Events could overwhelm the system.  If this seems
like a real problem, we can reduce the frequency further.

## Design Details

Described above.

### Test Plan

* Requisit unit tests.
* Add e2e tests that run insecurely and verify the conditions and events are
  created.
* Add e2e tests that run as root with the `runAs...` fields set and verify the
  conditions and events are NOT created.

### Graduation Criteria

#### Alpha -> GA Graduation

- Gather feedback from users and providers
- Tests are in Testgrid and linked in KEP
- SIG-Scalability confirms the Events are a non-issue

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness/README.md.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.

-->

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Also set `disable-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?).

* **What happens if we reenable the feature if it was previously rolled back?**

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling or disabling feature
  gates. However, unit tests in each component dealing with managing data, created
  with and without the feature, are necessary. At the very least, think about
  conversion tests if API types are being modified.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
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

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
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

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

* 2020-12-01: First draft

## Drawbacks

* This will annoy some users.
* This does not solve the root problems.

## Alternatives

We considered changing defaults and making users opt-in to running as root.
This is a breaking change and was discarded.

We considered injecting artificial slowdowns on insecure container startup, but
this is user-hostile and was discarded.
