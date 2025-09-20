# KEP-4818: Allow zero value for Sleep Action of PreStop Hook

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
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
    - [Upgrade](#upgrade)
    - [Downgrade](#downgrade)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
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

The sleep action for the PreStop container lifecycle hook was introduced in KEP 3960. It however doesn’t accept zero as a valid value for the sleep duration seconds. This KEP aims to add support for setting a value of zero with the sleep action of the PreStop hook.

## Motivation

Currently, trying to create a container with a PreStop lifecycle hook with sleep of 0 seconds will throw a validation error like so:

```
Invalid value: 0: must be greater than 0 and less than terminationGracePeriodSeconds (30)
```

The Sleep action is implemented with the time package from Go’s standard library. The `time.After()` which is used to implement the sleep permits a zero sleep duration. A negative or a zero sleep duration will cause the function to return immediately and function like a no-op.

The implementation in KEP 3960 supports only non-zero values for the sleep duration. It is semantically correct to support a zero value for this field since time.After() also supports zero and negative durations. Negative values as well as zero have the same effect with time.After(), they both return immediately. We don’t need to support negative values since they have the same effect as setting the duration to zero.

A potential use case for this behaviour is when you need a PreStop hook to be defined for the validation of your resource, but don't really need to sleep as part of the PreStop hook. An example of this is described by a user [here](https://github.com/kubernetes/enhancements/issues/3960#issuecomment-2208556397) in the parent KEP. They add a PreStop sleep hook in via an admission webhoook by default if the PreStop is hook is not specified by the user. In order to opt-out from this, a no-op PreStop hook with a duration of zero seconds can be used.

### Goals

- Update the validation for the Sleep action to allow zero as a valid sleep duration.
- Allow users to set a zero value for the sleep action in PreStop hooks to do a no-op.

### Non-Goals

- This KEP does not support adding negative values for the sleep duration.
- This KEP does not aim to provide a way to pause or delay pod termination indefinitely.

## Proposal

Introduce a `PodLifecycleSleepActionAllowZero` feature gate which is disabled by default. When the feature gate is enabled, the `validateSleepAction` method would allow values greater than or equal to zero as a valid sleep duration.

Since this update to the validation allows previously invalid values, care must be taken to support cluster downgrades safely. To accomplish this, the validation will distinguish between new resources and updates to existing resources:

- When the feature gate is disabled:
  - (a) New resources will no longer allow setting zero as the sleep duration second for the PreStop hook. (no change to current validation)
  - (b) Existing resources cannot be updated to have a sleep duration of zero seconds
  - (c) Existing resources with a PreStop sleep duration set to zero will continue to run and use a sleep duration of zero seconds. These can be updated and the zero sleep duration would continue to work.
- When the feature gate is enabled:
  - (c) New resources allow zero as a valid sleep duration.
  - (d) Updates to existing resources will allow zero as a valid sleep duration.

The proposed change adds another layer to the `validateSleepAction` function to allow zero as a valid sleep duration setting like shown:

```diff
-func validateSleepAction(sleep *core.SleepAction, gracePeriod *int64, fldPath *field.Path) field.ErrorList {
+func validateSleepAction(sleep *core.SleepAction, gracePeriod *int64, fldPath *field.Path, opts PodValidationOptions) field.ErrorList {
	allErrors := field.ErrorList{}
	// We allow gracePeriod to be nil here because the pod in which this SleepAction
	// is defined might have an invalid grace period defined, and we don't want to
	// flag another error here when the real problem will already be flagged.
-	if gracePeriod != nil && sleep.Seconds <= 0 || sleep.Seconds > *gracePeriod {
-		invalidStr := fmt.Sprintf("must be greater than 0 and less than terminationGracePeriodSeconds (%d)", *gracePeriod)
-		allErrors = append(allErrors, field.Invalid(fldPath, sleep.Seconds, invalidStr))
+	if opts.AllowPodLifecycleSleepActionZeroValue {
+		if gracePeriod != nil && sleep.Seconds < 0 || sleep.Seconds > *gracePeriod {
+			invalidStr := fmt.Sprintf("must be non-negative and less than terminationGracePeriodSeconds (%d)", *gracePeriod)
+			allErrors = append(allErrors, field.Invalid(fldPath, sleep.Seconds, invalidStr))
+		}
+	} else {
+		if gracePeriod != nil && sleep.Seconds <= 0 || sleep.Seconds > *gracePeriod {
+			invalidStr := fmt.Sprintf("must be greater than 0 and less than terminationGracePeriodSeconds (%d). Please enable PodLifecycleSleepActionAllowZero feature gate if you need a sleep of zero duration.", *gracePeriod)
+			allErrors = append(allErrors, field.Invalid(fldPath, sleep.Seconds, invalidStr))
+		}
	}
	return allErrors
}
```

Currently, the kubelet accepts `0` as a valid duration. There is no validation done at the kubelet level. All the validation for the duration itself is done at the kube-apiserver. The [runSleepHandler](https://github.com/AxeZhan/kubernetes/blob/3a96afdfefdf329c637623ae31a61d20dbdb0393/pkg/kubelet/lifecycle/handlers.go#L129-L141) in the kubelet uses the `time.After()` function from the [time](https://pkg.go.dev/time) package, which supports a `0` duration input. `time.After` also accepts negative values which are also returned immediately similar to zero. We don't support negative values however.

See the entire code changes in the WIP PR: [https://github.com/kubernetes/kubernetes/pull/127094](https://github.com/kubernetes/kubernetes/pull/127094)

### User Stories (Optional)

#### Story 1

As a Kubernetes user, I want to to be able to have a PreStop hook defined in my spec without needing to sleep during the execution of the PreStop hook. This no-op behaviour can be used for validation purposes with admission webhooks ([Reference](https://github.com/kubernetes/enhancements/issues/3960#issuecomment-2208556397)).

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

The change is opt-in, since it requires configuring a PreStop hook with sleep action of 0 second duration. So there is no risk beyond the upgrade/downgrade risks which are addressed in the Proposal section.

## Design Details

Refer to the Proposal section.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

Alpha:

- Test that the runSleepHandler function returns immediately when given a duration of zero.
- Test that the validation succeeds when given a zero duration with the feature gate enabled.
- Test that the validation fails when given a zero duration with the feature gate disabled.
- Test that the validation returns the appropriate error messages when given an invalid duration value (e.g., a negative value) with the feature gate disabled and enabled.
- Unit tests for testing the disabling of the feature gate after it was enabled and the feature was used.
- Unit tests for pod with zero grace period duration and zero sleep duration with zero value enabled.
- Unit test for pod with nil grace period with zero value disabled
- Unit test for pod with nil grace period with zero value enabled

Current coverages:

- `k8s.io/kubernetes/pkg/apis/core/validation` : 2024-09-20 - 84.3
- `k8s.io/kubernetes/pkg/kubelet/lifecycle/handlers` : 2024-09-20 - 86.4

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

N/A

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

Basic functionality

- Create a simple pod with a container that runs a long-running process.
- Add a preStop hook to the container configuration, using the new sleepAction with a sleep duration of `0`.
- Delete the pod and observe the time it takes for the container to terminate.
- Verify that the container terminates immediately without sleeping.

Additional e2e tests for beta:
  - Test that pods with sleep value of 0 in PreStop hook can be created
  - Test that pods with sleep value of 0 in PostStart hook can be created
  - Test that pods with sleep value of 0 in PreStop hook can be updated
  - Test that pods with sleep value of 0 in PostStart hook can be updated

### Graduation Criteria

#### Alpha
- Feature implemented behind a feature flag
- Initial unit/e2e tests completed and enabled

#### Beta
- Gather feedback from developers and surveys
- Additional e2e tests are completed
- No trouble reports from alpha release

#### GA
- No trouble reports with the beta release, plus some anecdotal evidence of it being used successfully.


### Upgrade / Downgrade Strategy

#### Upgrade

The previous PreStop Sleep Action behavior will not be broken. Users can continue to use their hooks as it is. To use this enhancement, users need to enable the feature gate, and set the sleep duration as zero in their prestop hook’s sleep action.

#### Downgrade

If the kube-apiserver is downgraded to a version where the feature gate is not supported (<v1.32), no new resources can be created with a PreStop sleep duration of zero seconds. Existing resources created with a sleep duration of zero will continue to function.

If the feature gate is turned off after being enabled, no new resources can be created with PreStop sleep duration of zero seconds. Existing resources will continue to run and use a sleep duration of zero seconds. These resources can be updated and the zero sleep duration would continue to work.

### Version Skew Strategy

Only the kube-apiserver will need to enable the feature gate for the full featureset to be present. This is because the implementation is already handled in the parent [KEP #3960](https://github.com/kubernetes/enhancements/issues/3960). The change introduced in this KEP is only to how the validation is done. If the feature gate is disabled, the feature will not be available. The feature gate does not apply to the kubelet logic since the time.After function used by the original KEP already supports zero as a valid duration.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: PodLifecycleSleepActionAllowZero
  - Components depending on the feature gate: kube-apiserver
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

The feature can be disabled in Alpha and Beta versions by restarting kube-apiserver with the feature-gate off. In terms of Stable versions, users can choose to opt-out by not setting the sleep field.

###### What happens if we reenable the feature if it was previously rolled back?

New pods with sleep action in prestop sleep duration of zero seconds can be created.

###### Are there any tests for feature enablement/disablement?

For the parent KEP, unit tests for the `switch` of the feature gate were added in `pkg/registry/core/pod/strategy_test`. We can add similar tests for the new feature gate as well.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

The change is opt-in, it doesn't impact already running workloads.

###### What specific metrics should inform a rollback?

I believe we don't need a metric here since the [parent KEP already has a metric](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/3960-pod-lifecycle-sleep-action#what-specific-metrics-should-inform-a-rollback) to inform rollbacks. This KEP only updates the validation to allow zero value.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This is an opt-in feature, and it does not change any default behavior. I manually tested enabling and disabling this feature by changing the kube-api-server config and restarting them in a kind cluster. The details of the expected behavior are described in the Proposal and Upgrade/Downgrade sections.


The manual test steps are as following:
1. Create a local 1.32 k8s cluster with kind, and create a test-pod in that cluster.
2. Enable PodLifecycleSleepActionAllowZero feature in the kube-apiserver and restart it.
3. Add a prestop hook with sleep action with duration of zero seconds to the test-pod and delete it. Observe the time cost.
4. Create another pod with sleep action duration of zero seconds.
5. Disable PodLifecycleSleepActionAllowZero feature in the kube-api-server and restart it.
6. Delete the pod created in step 4, and observe the time cost.


###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

Inspect the preStop hook configuration and also the feature gates

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [x] Other (treat as last resort)
  - Details: Check the logs of the container during termination, check the termination duration.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [x] Other (treat as last resort)
  - Details: Check the logs of the container during termination, check the termination duration.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

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

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

N/A. This is a change to validation within the API server.

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

Disable `PodLifecycleSleepActionAllowZero` feature gate, and restart the kube-apiserver.

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

- 2024-09-16: Alpha KEP PR opened for v1.32
- 2024-10-03: Summary, Motivation and Proposal sections merged
- 2024-09-03: [Alpha code implementation PR](https://github.com/kubernetes/kubernetes/pull/127094) opened
- 2024-11-01: Alpha code PR merged
- 2024-12-11: Kubernetes v1.32 release with PodLifecycleSleepActionAllowZero in alpha stage
- 2025-02-06: KEP updated targeting to beta in v1.33
- 2025-06-11: KEP updated targeting to stable in v1.34
- 2025-07-20: [Code implementation for GA graduation](https://github.com/kubernetes/kubernetes/pull/132595) merged into k/k
- 2025-10-20: k/enhancements PR opened updating KEP status as implemented

## Drawbacks

N/A

## Alternatives

Another way to run zero duration sleep in a container is to use the exec command in preStop hook like so `["/bin/sh","-c","sleep 0"]`. This requires a sleep binary in the image. Since the sleep action already exists as a PreStop hook, it is easier to allow a duration of zero seconds for the sleep action.

## Infrastructure Needed (Optional)

N/A