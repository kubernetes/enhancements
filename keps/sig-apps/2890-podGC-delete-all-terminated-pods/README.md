# KEP-2879: Allow podGC to delete all terminated pods

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
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This feature allows users to configure whether they want to delete all terminated pods or not.

## Motivation

Currently podGC controller starts cleaning up terminated Pods (with a phase of Succeeded or Failed), when the number of pods exceeds the configured threshold (determined by terminated-pod-gc-threshold in the kube-controller-manager). when user sets terminated-pod-gc-threshold <= 0, the terminated pod garbage collector is disabled. but in some production environment, users do not want to leave any terminated pods. Because of this, it should be opt-in for users to determinate whether they want to delete all terminated pods or not.

### Goals

- a flag that allows users to determinate whether they want to delete all terminated pods or not.

### Non-Goals

- N/A

## Proposal

Add a flag delete-all-terminated-pods in kube-controller-manager to be opt-in for users whether they want to delete all terminated pods or not.

### Risks and Mitigations

When the feature JobTrackingWithFinalizers is disabled, Job controller relies on completed Pods to not be removed in order to track the Job completion status, if user set delete-all-terminated-pods to true, podGC will delete all terminated pods, it will break jobs, 
the job will never be completed. users should take appropriate care using this feature, when user set delete-all-terminated-pods to true, they should enable the feature JobTrackingWithFinalizers.

## Design Details

This will be launched as an alpha feature first, with feature gate `PodGCDeleteAllTerminatedPods`.
we also add a flag delete-all-terminated-pods in kube-controller-manager to determinate whether delete all terminated pods or not. the default value of delete-all-terminated-pods is false. 
when the feature PodGCDeleteAllTerminatedPods is disabled, The value of delete-all-terminated-pods configured by user will not work. we will use the default value(false).
when the feature PodGCDeleteAllTerminatedPods is enabled, If users does not configure delete-all-terminated-pods, we will use the default value(false).
when the feature PodGCDeleteAllTerminatedPods is enabled, If users set delete-all-terminated-pods to false, terminated-pod-gc-threshold will be taken effect, podGC controller will delete terminated pods according to terminated-pod-gc-threshold.
when the feature PodGCDeleteAllTerminatedPods is enabled, If users set delete-all-terminated-pods to true, terminated-pod-gc-threshold will be ignored，podGC controller will delete all terminated pods.

### Test Plan

Unit tests will be added to `pkg/controller/podgc/gc_controller_test.go`. 

### Graduation Criteria

#### Alpha

- Feature gate disabled by default.
- Unit tests passing.

#### Beta

- Gather feedback from end users.
- Feature gate enabled by default.

#### GA

- Every bug report is fixed.
- The podGC controller ignores the feature gate.

#### Deprecation

N/A

### Upgrade / Downgrade Strategy

This features adds a flag delete-all-terminated-pods in kube-controller-manager. The default value for the new flag maintains the existing behavior of podGC controller.
On upgrade, podGC controller will start taking into account flag delete-all-terminated-pods when delete terminated pods.
On downgrade, controller-manager will stop taking into account flag delete-all-terminated-pods when delete terminated pods, and so reverting to old behavior.

### Version Skew Strategy

There are only kube-controller-manager changes involved. Node components
are not involved so there is no version skew between nodes and the control plane.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: PodGCDeleteAllTerminatedPods
  - Components depending on the feature gate:
    - kube-controller-manager
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

No, the default behavior remains the same.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes this feature can be disabled. Once disabled, terminated-pod-gc-threshold will be taken effect, podGC controller will delete terminated pods according to terminated-pod-gc-threshold.

###### What happens if we reenable the feature if it was previously rolled back?

It should work as expected.

###### Are there any tests for feature enablement/disablement?

Yes, unit tests for feature enabled, disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

N/A

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

kubernetes cluster does not exist terminated pods.
It isn't a workloads feature, but a control plane one.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

podGC controller can not delete terminated pods successfully.

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

2021-08-25: Proposed KEP starting in beta status

## Drawbacks

N/A

## Alternatives

N/A
