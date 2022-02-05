# KEP-2903: Export relistThreshold as a field of KubeletConfiguration named PLEGRelistThreshold.  

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
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Scalability](#scalability)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

export relistThreshold as a field of KubeletConfiguration named PLEGRelistThreshold.

The relistthreshold needs to be greater than the relisting period + the relisting time, which can vary significantly. Set a conservative threshold to avoid flipping between healthy and unhealthy.

default value: 3 * time.Minute

## Motivation

The relistthreshold needs to be greater than the relisting period + the relisting time, which can vary significantly. Set a conservative threshold to avoid flipping between healthy and unhealthy.

default value: 3 * time.Minute

I think users should be allowed to configure relistthreshold. Different nodes have different performances, and the pod on the node also changes. Users should be allowed to determine the relistthreshold value.
To maintain logical compatibility, the default value is set to 3 minutes, providing more flexible configuration.

### Goals
 *  export relistThreshold as a field of KubeletConfiguration named PLEGRelistThreshold to make it configurable

### Non-Goals
None

## Proposal
Add kubelet cmd arg:  --pleg-relist-threshold to make  relistthreshold configurable

### Risks and Mitigations

| Risk                                             | Impact | Mitigation |
| -------------------------------------------------| -------| ---------- |
| Too low value | High   | A value that is too small will cause the node to be unhealthy  |

## Design Details

1. export relistThreshold as a field of KubeletConfiguration named PLEGRelistThreshold

### Test Plan

We will extend both the unit test suite and the E2E test suite.

### Graduation Criteria

#### Alpha

- [X] Implement the kubelet cmd arg.
- [X] Ensure proper unit tests are in place.
- [X] Ensure proper e2e node tests are in place.

#### Beta

- [X] Gather feedback from consumers of the new kubelet cmd arg.
- [X] Verify no major bugs reported in the previous cycle.

#### GA

- [X] Allow time for feedback (1 year).
- [X] Make sure all risks have been addressed.

### Upgrade / Downgrade Strategy

We expect no impact. The new policy option is opt-in and orthogonal to the existing ones.

### Version Skew Strategy

No changes needed

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [] Feature gate (also fill in values in `kep.yaml`)
- [X] Change the kubelet configuration to unset  set PLEGRelistThreshold in KubeletConfiguration
  - Will enabling / disabling the feature require downtime of the control
    plane? YES 
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
	Yes -- a kubelet restart is required.

###### Does enabling the feature change any default behavior?

No. default: 3m0s same as before

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

No changes.

###### Are there any tests for feature enablement/disablement?

- unittest 
- e2e test

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Inspect the kubelet configuration of a node.

###### How can someone using this feature know that it is working for their instance?

Inspect the kubelet configuration of a node.


###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

There are no specific SLOs for this feature.
Parallel workloads will benefit from this feature in application specific ways.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

None

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

None

###### Does this feature depend on any specific services running in the cluster?

None

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

## Implementation History

Todo
