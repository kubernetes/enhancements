# Disable CAdvisor Json Metrics

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
    - [GA Graduation](#ga-graduation)
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
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP outlines the process to deprecate cAdvisor json metrics collected by Kubelet.

This is one step towards removing cAdvisor APIs from the kubelet, which has been a long-time goal of sig-node.  cAdvisor only supports linux, and only supports a small set of container runtimes.  For that reason, sig-node has historically wanted to allow vendors to replace cAdvisor without paying a double-collection performance penalty.  This hasn't been achieved yet, despite some incremental progress.  This KEP is another small step towards that goal.

Note that cAdvisor endpoints are not believed to be widely used.  They were entirely broken for multiple releases before someone reported it: [kubernetes/kubernetes#62544](https://github.com/kubernetes/kubernetes/pull/62544).

This was originally part of the ["metrics overhaul" KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/20181106-kubernetes-metrics-overhaul.md), but was [removed](https://github.com/kubernetes/enhancements/pull/1935) because it had not been completed. There were also [concerns raised](https://github.com/kubernetes/kubernetes/issues/68522#issuecomment-645454818) about removing these metrics.

cAdvisor json metrics were disabled by default starting in 1.18, and could be re-enabled by setting `--enable-cadvisor-json-endpoints` to true.  However, given the concerns about removing the endpoints, we want to re-visit the deprecation and removal of these metrics before removing them permanently.

## Motivation

### Goals

* Remove cAdvisor v1 ContainerInfo json metrics (`/stats/container`, `/stats/<podname>/<containername>`, `/stats/<namespace>/<podname>/<poduid>/<containername>`) from the kubelet.
* Remove cAdvisor v1 MachineInfo json metrics (/spec) from the kubelet.

### Non-Goals

* Remove or modify cadvisor prometheus metrics from the kubelet (/metrics/prometheus).
* Remove or modify the Summary API 
* Eliminate the kubelet's dependence on cAdvisor for metrics to supply the Summary API.

## Proposal

### Risks and Mitigations

The main risk that we face is breaking existing consumers of the cAdvisor json metrics.

There are a few migration paths for users:

For all metrics: Run cAdvisor as a daemonset.  See the [Instructions for running cAdvisor as a daemonset](https://github.com/google/cadvisor/tree/master/deploy/kubernetes#cadvisor-kubernetes-daemonset).
* Pros: Provides the exact same APIs.
* Cons: Can be expensive to run another instance of cAdvisor.

For container metrics: Use an alternative kubelet endpoint.  Container metrics are available in /metrics/resource, /metrics/cadvisor, /stats/summary.
* Pros: No additional cost
* Cons: Metrics are in a different format and may not contain all information available in the json endpoints.

For machine metrics: Use the prometheus node exporter.
* Pros: Community-supported and widely used machine-level monitoring tool.  Easy-to-use configuration to enable/disable metrics.
* Cons: Metrics are in a different format, and may not have the same set of information

## Design Details

Remove the `--enable-cadvisor-json-endpoints` flag and the kubelet stops serving on the paths listed in the Goals section.

### Test Plan

* This will not have any e2e testing.

### Graduation Criteria

#### GA Graduation

* The deprecated flag and relevant code have been removed.

### Upgrade / Downgrade Strategy

- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to keep previous behavior? N/A.
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to make use of the enhancement? N/A.

### Version Skew Strategy

- Does this enhancement involve coordinating behavior in the control plane and in the kubelet? How does an n-2 kubelet without this feature available behave when this feature is used? N/A.
- Will any other components on the node change? For example, changes to CSI, CRI or CNI may require updating that component before the kubelet. N/A.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**
  As of 1.18, these metrics can be re-enabled using `--enable-cadvisor-json-endpoints=true`.  After this KEP, it will not be possible to re-enable these metrics.

* **Does enabling the feature change any default behavior?** No.  These metrics are already disabled by default.
* **Can the feature be disabled once it has been enabled (i.e. can we rollback the enablement)?** The metrics can be enabled using the flag, but after this "feature", it will no longer be possible to do so.

* **What happens if we reenable the feature if it was previously rolled back?** Metrics are collected again.
* **Are there any tests for feature enablement/disablement?** No.

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
* **Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?** No.

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?** N/A.
* **What are other known failure modes?** N/A.
* **What steps should be taken if SLOs are not being met to determine the problem?** N/A

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2020-10-04: Initial version of the KEP

## Drawbacks

This feature is likely to break consumers of that metric.

## Alternatives

Keep the cAdvisor json endpoints.  Either plan to remove them at a later date, or plan to keep cAdvisor in the kubelet.
