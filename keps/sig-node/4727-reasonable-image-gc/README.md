
# KEP-4727: reasonable --image-gc-high-threshold according to imagefs.available hard evict option

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Add an feature gate ImageGCBeforeStorageEviction, which denote image gc must occur before kubelet evict.
When ImageGCBeforeStorageEviction is false or is not configured, keep the current behavior.


## Motivation

The default value of `--image-gc-high-threshold` option is 85%, and the default value of `imagefs.available` option is
less than 15%, this will result in image garbage collection not taking effect until node gets disk pressure.

There is no standard to judge whether the value of `--image-gc-high-threshold` is reasonable in different scenarios

### Goals

Discuss reasonable values of `--image-gc-high-threshold` for different scenarios, and constrain them by some means.
Eventually protect users from inopportune configurations, and fix the defaults of `--image-gc-high-threshold` and `imagefs.available` to make they more sense.


### Non-Goals


## Proposal



### User Stories (Optional)


#### Story 1

In big data computing scenarios, user often run some computing tasks . 
When these tasks are completed, a large number of images are stored on the node. 
At this time, user want to perform image garbage collection before the node disk pressure occurs. 
There should be validation to protect that expectation

#### Story 2

Cluster administrator misconfigures `--image-gc-high-threshold` and `imagefs.available`.


### Notes/Constraints/Caveats (Optional)


### Risks and Mitigations


## Design Details

Add `ImageGCBeforeStorageEviction` feature gate to kubelet. The usage scenarios are as follows:

1. When the feature gate is turned on 
   - the value of `--image-gc-high-threshold` must be smaller than  value of `100 - imagefs.available`.
   - the default value of `--image-gc-high-threshold` is 80
   - the default value of `--image-gc-low-threshold` is 75
   - the default value of `imagefs.available` is not changed

2. When the feature gate is turned off
   - keep the previous usage
   - the default value of `--image-gc-high-threshold`、`--image-gc-low-threshold`、`imagefs.available` are not changed
   - if `--image-gc-high-threshold` must be greater than  value of `100 - imagefs.available`， will output warning level log

### Test Plan


[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates



##### Unit tests


- `pkg/kubelet/apis/config/validation`: `2024:08:08` - `97.1%`

##### Integration tests



- <test>: <link to test coverage>

##### e2e tests


- <test>: <link to test coverage>

### Graduation Criteria


#### Alpha
- Feature implemented behind feature gate.
- Unit tests passed as designed in [TestPlan](#test-plan).

#### Beta

- Gather feedback from developers.

#### GA
- No negative feedback.
- Update documents to reflect the changes.

### Upgrade / Downgrade Strategy

This option is purely contained within the Kubelet, so the only concern is the flag is added to the configuration of the newer
Kubelet and then downgraded.

### Version Skew Strategy


## Production Readiness Review Questionnaire


### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ImageGCBeforeStorageEviction
  - Components depending on the feature gate: kubelet


###### Does enabling the feature change any default behavior?

Yes

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, we can just disable the feature gate.

###### What happens if we reenable the feature if it was previously rolled back?

The constraints are respected again.

###### Are there any tests for feature enablement/disablement?

No

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

It's an opt-in feature for end-users and will maintain current behaviors if not set, so
it will not impact the running workloads.

###### What specific metrics should inform a rollback?

No

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

Monitor the metrics
- "kubelet_image_gc_before_storage_eviction" that contains `image-gc-threshold` and `imagefs-available` labels


###### How can an operator determine if the feature is in use by workloads?

- Verify the Kubelet Configuration with the Kubelet's configz endpoint
- Monitor the `kubelet_image_gc_before_storage_eviction`, denote whether `--image-gc-high-threshold` smaller than  `100 - imagefs.available`


###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - `kubelet_image_gc_before_storage_eviction` metric is 1 when `--image-gc-high-threshold` smaller than  `100 - imagefs.available`.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?



###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?


- [x] Metrics
  - Metric name: `kubelet_image_gc_before_storage_eviction`
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

Just Kubelet

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

- Potentially, depending on the value of `--image-gc-high-threshold` chosen, there could be more CPU used to do the image removal.
  - The frequency of the image removal will be a tradeoff for disk pressure of node

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

- It's intended to prevent node get disk pressure

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

- N/A

###### What are other known failure modes?

Node gets disk pressure

###### What steps should be taken if SLOs are not being met to determine the problem?

- N/A

## Implementation History

2024-06-26: KEP opened, targeted at Alpha

## Drawbacks

No

## Alternatives

- Add a distinguish unused image growth trends to dynamically adjust `--image-gc-high-threshold` plugin
    - Too complicated, probably won't needed, difficult to distinguish whether an unused image is quickly used or not

## Infrastructure Needed (Optional)

N/A
