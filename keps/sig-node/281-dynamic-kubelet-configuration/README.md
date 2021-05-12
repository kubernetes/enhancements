# KEP-281: Dynamic Kubelet Configuration

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Dynamic Kubelet Configuration allows a new Kubelet configurations to be rolled out in a live cluster.

The feature predates the KEP process as it is defined today. Please find
motivation, goals, and design section in [community repository](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/dynamic-kubelet-configuration.md).

### Risks and Mitigations

TODO for the feature deprecation effort

### Test Plan

TODO for the feature deprecation effort

### Graduation Criteria

TODO for the feature deprecation effort

### Upgrade / Downgrade Strategy

TODO for the feature deprecation effort

### Version Skew Strategy

TODO for the feature deprecation effort

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**

TODO for the feature deprecation effort

* **Does enabling the feature change any default behavior?**

TODO for the feature deprecation effort

* **Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?**

TODO for the feature deprecation effort

* **What happens if we reenable the feature if it was previously rolled back?**

TODO for the feature deprecation effort

* **Are there any tests for feature enablement/disablement?**

TODO for the feature deprecation effort

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**

TODO for the feature deprecation effort

* **What specific metrics should inform a rollback?**

TODO for the feature deprecation effort

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

TODO for the feature deprecation effort

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**

TODO for the feature deprecation effort

* **What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?**

TODO for the feature deprecation effort

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

TODO for the feature deprecation effort

* **Are there any missing metrics that would be useful to have to improve observability of this feature?**

TODO for the feature deprecation effort

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**

TODO for the feature deprecation effort

### Scalability

* **Will enabling / using this feature result in any new API calls?**

TODO for the feature deprecation effort

* **Will enabling / using this feature result in introducing new API types?**

TODO for the feature deprecation effort

* **Will enabling / using this feature result in any new calls to the cloud provider?**

TODO for the feature deprecation effort

* **Will enabling / using this feature result in increasing size or count of the existing API objects?**

TODO for the feature deprecation effort

* **Will enabling / using this feature result in increasing time taken by any operations covered by [existing SLIs/SLOs]?**

TODO for the feature deprecation effort

* **Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?**

TODO for the feature deprecation effort

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**

TODO for the feature deprecation effort

* **What are other known failure modes?**

TODO for the feature deprecation effort

* **What steps should be taken if SLOs are not being met to determine the problem?**

TODO for the feature deprecation effort

## Implementation History

- 1.8 Alpha release
- 1.9 Incremental improvements working towards 1.10 goals
- 1.11 Beta release

## Drawbacks

TODO for the feature deprecation effort

## Alternatives

TODO for the feature deprecation effort

## Infrastructure Needed (Optional)

TODO for the feature deprecation effort
