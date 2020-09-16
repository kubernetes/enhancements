# Make Cadvisor MetricsSet Configurable

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha Graduation](#alpha-graduation)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
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
<!-- /toc -->

## Release Signoff Checklist

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements](https://github.com/kubernetes/enhancements/issues/1920)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This KEP outlines the process to configure Cadvisor Metrics collected by Kubelet.

Now Cadviosr metrics collected by kubelet are hard coding. It's not kind for all nodes which has more than 250 pods, secrets, configmaps using syscall to scan file.
Kubelet will hang if all metrics collected default without configurable metrics set.Or we can configure more metrics collected by cadvisor like advanced tcp stats or perf metrics.
It will be free to do more if CustomCadvisorMetrics feature gate enabled.

## Motivation

### Goals

Make Cadvisor MetricsSet configurable in kubelet. Also can configure kubelet stats prometheus json API by edit metrics set.

### Non-Goals

Removing feature gate like `DisableAcceleratorUsageMetrics`, it is overlapped.

## Proposal

### Risks and Mitigations
Users may add more metrics collected by cadvisor interface worked background which leads to kubelet hang. But feature gate `CustomCadvisorMetrics` is default disabled. Users can ignore
changes after this PR if do not care about what metrics collected before. Not all metrics like cpu, mem can be configured, because these metrics required by kubelet itself, we should add
a whitelist to handle these metrics.


## Design Details

Add a feature flag and pass the enable option and custom metrics set supported to kubelet.

### Test Plan

* E2E test that checks when the feature flag is enabled and passes custom metrics sets if these metrics are present or not.

### Graduation Criteria

#### Alpha Graduation

* Feature Flag is present.
* E2E tests are implemented.
* Documentation has been published on how to transition to the new metrics.
* Release Notes have been created and promote immediate usage of that flag.
  Our recommendation should be to enable this flag at alpha.

#### Alpha -> Beta Graduation

* One release has been waited to allow for feedback from users.
* A Blog post has been written and published on the Kubernetes blog.

#### Beta -> GA Graduation

* At least one year has been waited to allow for feedback from users. Most users won't notice this until it is enabled by default, and thus if we want to give users time to adapt and migrate to the daemonset, it would be between Beta and GA
* Address feedback on usage/changed behavior, provided on GitHub issues.

### Upgrade / Downgrade Strategy

- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to keep previous behavior? N/A.
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to make use of the enhancement? N/A.

### Version Skew Strategy

- Does this enhancement involve coordinating behavior in the control plane and in the kubelet? How does an n-2 kubelet without this feature available behave when this feature is used? N/A.
- Will any other components on the node change? For example, changes to CSI, CRI or CNI may require updating that component before the kubelet. N/A.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate
    - Feature gate name: CustomCadvisorMetrics.
    - Components depending on the feature gate: Kubelet.

* **Does enabling the feature change any default behavior?** Yes, it will change metrics set for cadvisor.
* **Can the feature be disabled once it has been enabled (i.e. can we rollback the enablement)?** Yes it's a feature gate.

* **What happens if we reenable the feature if it was previously rolled back?** Custom metrics provided are collected again.
* **Are there any tests for feature enablement/disablement?** Planned for Alpha.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?** N/A.
* **What specific metrics should inform a rollback?** N/A.
* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?** Not yet, probably N/A.
* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?** Yes.

### Monitoring requirements

* **How can an operator determine if the feature is in use by workloads?** N/A.
* **What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?** N/A.
* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?** N/A.
* **Are there any missing metrics that would be useful to have to improve observability if this feature?** N/A.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?** N/A.

### Scalability

* **Will enabling / using this feature result in any new API calls?** No.
* **Will enabling / using this feature result in introducing new API types?** No.
* **Will enabling / using this feature result in any new calls to cloud provider?** No.
* **Will enabling / using this feature result in increasing size or count of the existing API objects?** No.
* **Will enabling / using this feature result in increasing time taken by any operations covered by [existing SLIs/SLOs][]?** No.
* **Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?** No, actually results in decreased resource usage.

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?** N/A.
* **What are other known failure modes?** N/A.
* **What steps should be taken if SLOs are not being met to determine the problem?** N/A

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History


## Drawbacks

This feature may collected fussy metrics which leads to kubelet hang.

## Alternatives

Remove cadvisor metrics from kubelet and run cadvisor as daemonset in kubernetes. It will export a port or unix socket domain to handle connection between cadvisor and kubelet.

