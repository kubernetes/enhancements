# Disable AcceleratorUsage Metrics

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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements](https://github.com/kubernetes/enhancements/issues/1867)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP outlines the process to deprecate the Accelerator Metrics collected by Kubelet.

Accelerator metrics are no longer expected to collected by Kubelet, but by external monitoring agents using the PodResources API.
The purpose of creating that API was to provide an out of tree mechanism for all vendors to provide device specific metrics.
This allows them to provide these metrics without requiring them to make changes to Kubernetes.

Now that this API is beta and soon to be G.A, this KEP outlines the process to deprecate metrics that were added before sig-node conveged on the PodResources API.

## Motivation

### Goals

Deprecate and remove the AcceleratorUsage metrics that kubelet currently advertises.

### Non-Goals

Deprecation and removal of the summary API is certainly a goal of sig-node, and the step of removing the AcceleratorUsage metric is a step in that direction (agreed upon goal with incremental steps towards it) but not the goal of this KEP.

## Proposal

### Risks and Mitigations

The main risk that we face is breaking existing consumers of the AcceleratorUsage metrics.
The way this risk is mitigated is through the use of Feature Flags, which allows users to re-enable this metric.
We will also be pointing in documentation users towards the newer and richer method of collecting metrics.

Note that we don't know who are the consumers of that metric but we suspect that this will impact a small subset of users as these metrics on NVIDIA GPU utilization are often unreliable and very coarse.

## Design Details

Add a feature flag and pass the disable option to cadvisor.

### Test Plan

* E2E test that checks when the feature flag is enabled if the metrics are present or not.

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
    - Feature gate name: DisableAcceleratorUsageMetrics.
    - Components depending on the feature gate: Kubelet.

* **Does enabling the feature change any default behavior?** Yes, removes GPU Accelerator Metrics.
* **Can the feature be disabled once it has been enabled (i.e. can we rollback the enablement)?** Yes it's a feature gate.
  Grafana dashboards and other applications relying on this feature will likely show up as blank after an update.
  Note that this would be three metrics: `MemoryTotal`, `MemoryUsed`, `DutyCycle` (GPU Utilization).

* **What happens if we reenable the feature if it was previously rolled back?** Metrics are collected again.
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

- 2020-06-18: Initial version of the KEP

## Drawbacks

This feature is likely to break consumers of that metric.

## Alternatives

Add a config to Kubelet. However config defaults can not be changed. Which prevents us from having a deprecation period of time.
