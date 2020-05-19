# KEP-785: Scheduler Component Config API

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The kube-scheduler configuration API `kubescheduler.config.k8s.io` is currently
in version `v1alpha2`. We propose its graduation to `v1beta1` in order to
promote its wider use.

## Motivation

The `kubescheduler.config.k8s.io` API has been in alpha stage for several
releases. In release 1.18, we introduced `v1alpha2`, including important
changes such as:

- The removal of the old Policy API in favor of plugin configurations, that
  align with the new scheduler framework.
- The introduction of scheduling profiles, that allow a scheduler to appear
  as multiple schedulers under different configurations.
  
A configuration API allows cluster administrators to build, validate and
version their configurations in a more robust way than using command line flags.

Graduating this API to Beta is a sign of its maturity that would encourage wider
usage.

### Goals

- Introduce `kubescheduler.config.k8s.io/v1beta1` as a copy of
`kubescheduler.config.k8s.io/v1alpha2` with minimal cleanup changes.
- Use the newly created API objects to build the default configuration for kube-scheduler.
- Remove support for `kubescheduler.config.k8s.io/v1alpha2`

### Non-Goals

- Update configuration scripts in /cluster to use API.

## Proposal

For the most part, `kubescheduler.config.k8s.io/v1beta1` will be a copy of
`kubescheduler.config.k8s.io/v1alpha2`, with the following differences:

- [ ] `.bindTimeoutSeconds` will be an argument for `VolumeBinding` plugin.
- [ ] `.profiles[*].plugins.unreserve` will be removed.
- [ ] Embedded types of `RequestedToCapacityRatio` will include missing json tags
  and will be decoded with a case-sensitive decoder.

### Risks and Mitigations

The major risk is around the removal of the `unreserve` extension point.
However, this is mitigated for the following reasons:

- The function from `Unreserve` interface will be merged into `Reserve`,
  effectively requiring plugins to implement both functions.
- There are no in-tree Reserve or Unreserve plugins prior to 1.19.
  The `VolumeBinding` plugin is now implementing both interfaces.
  
The caveat is that out-of-tree plugins that want to work 1.19 need to
updated to comply with the modified `Reserve` interface, otherwise scheduler
startup will fail. Plugins can choose to provide empty implementations.
This will be documented in https://kubernetes.io/docs/reference/scheduling/profiles/

### Test Plan

- [ ] Compatibility tests for defaults and overrides of `.bindTimeoutSeconds`
  in `VolumeBindingArgs` type.
- [ ] Tests for `RequestedToCapacityRatioArgs` that: (1) fail to pass with
  bad casing and (2) get encoded with lower case.

### Graduation Criteria

#### Alpha -> Beta Graduation

- Complete features listed in [proposal][#proposal].
- Tests in [test plan](#test-plan)

## Production Readiness Review Questionnaire

### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**

  Operators can use the config API via `--config` command line flag for
  `kube-scheduler`. To disable, operators can remove `--config` flag and use
  other command line flags to configure the scheduler.

* **Does enabling the feature change any default behavior?**

  No

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**

  By removing `--config` command line flag for `kube-scheduler`.

* **What happens if we reenable the feature if it was previously rolled back?**

  N/A.

* **Are there any tests for feature enablement/disablement?**

  The e2e framework does not currently support changing configuration files.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**

  A malformed configuration will cause the scheduler to fail to start.
  Running workloads are not affected.

* **What specific metrics should inform a rollback?**

  Metric "schedule_attempts_total" remaining at zero when new pods are added.

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**

  N/A.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  
  Configuration API `kubescheduler.config.k8s.io/v1alpha2` is removed.

### Monitoring requirements

* **How can an operator determine if the feature is in use by workloads?**

  N/A.

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  
  N/A.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

  N/A.

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**

  N/A.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**

  No.


### Scalability

* **Will enabling / using this feature result in any new API calls?**

  No

* **Will enabling / using this feature result in introducing new API types?**

  No.

* **Will enabling / using this feature result in any new calls to cloud
  provider?**
  
  No.

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**
  
  No.

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  
  No.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  
  No.

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**

  N/A.

* **What are other known failure modes?**

  Configuration errors are logged to stderr.

* **What steps should be taken if SLOs are not being met to determine the problem?**

  N/A.

## Implementation History

- 2020-05-08: KEP for beta graduation sent for review, including motivation,
  proposal, risks, test plan and graduation criteria.
- 2020-05-13: KEP updated to remove v1alpha2 support.
