# KEP-4940: Add PSA to block setting `.host` field from ProbeHandler and LifecycleHandler

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

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

We have a `Host` field that can be set from `TCPSocketAction` and
`HTTPGetAction` fields which are part of the `ProbeHandler` and
`LifecycleHandler` structs in Kubernetes. The `Host` field is used
for allowing users to specify another entity other than the
podIP (which is the default value) to which Kubelet should
perform probes to. However this opens it up for security attacks
since the `Host` field can be set to pretty much any value in the
system including security sensitive external hosts or localhost on
the node. Kubelet will be probing this set `Host` value which can
lead to blind SSRF attacks.

### Goals

* Add PSA to enable admins to restrict users from creating
  probes with the `Host` field set.

### Non-Goals

N/A

## Proposal

There is a long term plan to deprecate the existing TCP and HTTP probe
types in the API to replace them with ones with slightly different semantics.
See [KEP-4559](https://github.com/kubernetes/enhancements/pull/4558) for more
details. Given the unsolvable security problems with the Host field,
we do not plan to offer it in the new types.

Meanwhile, the older API is never going to go away. So we also want to
add PSA to allow admins to be able to restrict users from creating
probes with the Host field set when using the (about to be deprecated) API.
This is implemented by [kubernetes PR 125271](https://github.com/kubernetes/kubernetes/pull/125271)
that does exactly that.

### Risks and Mitigations

There might be users who depend on the `Host` field in
their existing probes which will continue to work and if
newly created probes also need the `Host` field to point
to an external destination then the admin can avoid enforcing
the PSA to block it.

## Design Details

Add a Baseline APILevel Pod Security Admission policy to allow admins of the
cluster to block users from setting `.host` field in:

* container.LivenessProbe.ProbeHandler.HTTPGet.Host
* container.ReadinessProbe.ProbeHandler.HTTPGet.Host
* container.StartupProbe.ProbeHandler.HTTPGet.Host
* container.LivenessProbe.ProbeHandler.TCPSocket.Host
* container.ReadinessProbe.ProbeHandler.TCPSocket.Host
* container.StartupProbe.ProbeHandler.TCPSocket.Host
* container.Lifecycle.PostStart.TCPSocket.Host // Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for backward compatibility.
* container.Lifecycle.PreStop.TCPSocket.Host // Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for backward compatibility.
* container.Lifecycle.PostStart.HTTPGet.Host
* container.Lifecycle.PreStop.HTTPGet.Host

### Test Plan

* Unit and E2E tests will be added to ensure the PSA works as expected

##### Prerequisite testing updates

None

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `<package>`: `<date>` - `<test coverage>`

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

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

### Graduation Criteria

The PSA added will be done within a single release
and given there will be no feature gates for that,
there is no need for multi-release graduation criteria.
All related code will land within the same single release


#### Deprecation

TBD

### Upgrade / Downgrade Strategy

Any older pods with this field set should not be affected
with the above solution. Only newer pods getting created
with the field will be alerted.

Users who are using this field can switch to using exec
probes moving forward which should unblock them given exec
probes can provide the same functionality.


### Version Skew Strategy

N/A since its only within a single component: pod-security-admission
and doesn't cross multiple components.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

We decided to not go with feature gates and use PSA versioning.
So if the admin sets pod-security.kubernetes.io/enforce-version: v1.33
on a namespace this feature will get enabled.

###### Does enabling the feature change any default behavior?

* There is no effect on clusters where PSA is not enabled OR an older
PSA version is used.

* There is no effect on clusters where `.Host` probes are not used

* There is no effect on clusters where an older PSA versioning is being
used

* If users create pod probes with `.Host` field set and the admin
has set baseline PSA level to `enforce` mode then the request will be
actively blocked and rejected. Existing pods with `.Host` probes
that are upgrading will not be impacted.


###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

No

###### What happens if we reenable the feature if it was previously rolled back?

N/A

###### Are there any tests for feature enablement/disablement?

N/A since there is no feature gate

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

N/A

###### How can an operator determine if the feature is in use by workloads?

If pods have probes with `.Host` field set and PSA annotation is set on that pod's namespace
to a version where the new admission has been added, then it means the feature is enabled.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

TBD

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?
N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

None

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

N/A

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

###### How does this feature react if the API server and/or etcd is unavailable?

N/A

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

## Drawbacks

N/A

## Alternatives

The alternative is to remove this field from the API after
its deprecated, but that's not a supported API action.
