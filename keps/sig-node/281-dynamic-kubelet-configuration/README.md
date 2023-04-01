# KEP-281: Dynamic Kubelet Configuration

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Deprecation](#deprecation)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Deprecation Readiness Review Questionnaire](#deprecation-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade, and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [X] Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Dynamic Kubelet Configuration allows a new Kubelet configurations to be rolled
out in a live cluster.

The feature predates the KEP process as it is defined today. Please find
motivation, goals, and design section in [community repository](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/dynamic-kubelet-configuration.md).

Dynamic Kubelet Configuration feature is deprecated at the version v1.22 due to
lack of interest in promoting it to stable. See [Deprecation](#deprecation).

## Deprecation

Dynamic Kubelet Configuration feature was promoted to beta in [v1.11, 3 July 2018](https://kubernetes.io/blog/2018/06/27/kubernetes-1.11-release-announcement).
As per the [Avoiding Permanent Beta](https://kubernetes.io/blog/2020/08/21/moving-forward-from-beta/#avoiding-permanent-beta)
policy this feature is marked for deprecation as there were no interest expressed to
promote the feature to stable. Removal of Dynamic Kubelet Configuration logic will
simplify code and improve code reliability.

The functionality was removed from kubelet in 1.24 and will be removed from API server in 1.26.

### Risks and Mitigations

The deprecation announcement may uncover wide feature usage.
So far only one complain [was expressed](https://github.com/kubernetes/enhancements/issues/281#issuecomment-695751859).

Mitigation is to suggest adoption of alternative solutions for
configuration management. Since the feature existed for many releases,
deprecation period may be prolonged comparing to the obligated one release
as documented in [Rule #5b](https://kubernetes.io/docs/reference/using-api/deprecation-policy/#:~:text=rule%20%235b%3A)
of the deprecation policy.

### Test Plan

All tests will be running while the feature is marked as deprecated.

### Graduation Criteria

Graduation criteria for Dynamic Kubelet Config was to receive feedback on
feature usage and shortcomings and address those before promoting it to stable.
So far the big feedback is that the feature needs to be expanded to provide
better reliability promises and allow to restrict the set of configuration
options available for change. Some Kubernetes vendors already using alternative
solutions for Kubelet Configuration distribution and update.

Since graduation criteria wasn't met for many releases, the feature is marked for
deprecation.

Note, there will be a separate update of this doc when feature will be marked
for removal.

### Upgrade / Downgrade Strategy

Not applicable for deprecation.

### Version Skew Strategy

N/A

TODO: Section is to be updated once feature is scheduled for removal.

## Deprecation Readiness Review Questionnaire

Note, this is **deprecation** readiness questionnaire.

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**

Feature will be marked as deprecated and deprecated warnings will be displayed.

* **Does enabling the feature change any default behavior?**

Deprecated warnings will be shown when the feature is being used.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?**

There is a slight chance the feature will be un-deprecated if it will
find an owner committed to it.

* **What happens if we reenable the feature if it was previously rolled back?**

N/A, feature is removed from kubelet.

* **Are there any tests for feature enablement/disablement?**

There will be no tests for the deprecation warnings.

### Rollout, Upgrade, and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**

No, feature deprecation will not impact any workload. On removal of the
feature, new kubelet versions will not apply any dynamic configuration.

* **What specific metrics should inform a rollback?**

N/A, feature is removed

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

N/A, feature is removed

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**

Deprecated warnings will be displayed for the operator to learn about deprecation.

Operator can also track metrics that report the assigned (node_config_assigned),
last-known-good (node_config_last_known_good), and active (node_config_active)
config sources. Metrics will indicate that the feature is in use and the migration
effort needs to be scheduled.

* **What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?**

N/A

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

N/A

* **Are there any missing metrics that would be useful to have to improve observability of this feature?**

N/A

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**

No

### Scalability

* **Will enabling / using this feature result in any new API calls?**

No

* **Will enabling / using this feature result in introducing new API types?**

No

* **Will enabling / using this feature result in any new calls to the cloud provider?**

No

* **Will enabling / using this feature result in increasing size or count of the existing API objects?**

No

* **Will enabling / using this feature result in increasing time taken by any operations covered by [existing SLIs/SLOs]?**

No

* **Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?**

No

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**

N/A

* **What are other known failure modes?**

None

* **What steps should be taken if SLOs are not being met to determine the problem?**

N/A

## Implementation History

- 1.8 Alpha release
- 1.9 Incremental improvements working towards 1.10 goals
- 1.11 Beta release
- 1.21 Feature is marked for deprecation
- 1.24 Feature is removed from kubelet. It will be removed from control plane in 1.26 to support version skew.
- 1.26 Fully removed from kubernetes

## Drawbacks

The biggest drawback of feature deprecation is potential use of the feature.

## Alternatives

Alternatives for Dynamic Kubelet Configuration feature is a solution that updates
kubelet configuration file and restarts kubelet which is preferred solution as it
increases kubelet reliability and simplifies it's code. While allowing extra features
for kubelet configuring be implemented faster and specific to the kubernetes environment.

## Infrastructure Needed (Optional)

None
