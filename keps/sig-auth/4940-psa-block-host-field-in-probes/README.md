# KEP-4940: Add Pod Security Admission (PSA) to block setting `.host` field from ProbeHandler and LifecycleHandler

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
`LifecycleHandler` structs in Kubernetes that are used in
`InitContainers` and `Containers` structs of `PodSpec`.
The `Host` field is used for allowing users to specify
another entity other than the podIP (which is the default value) to
which Kubelet should perform probes to.
However this opens it up for security attacks since the `Host`
field can be set to pretty much any value in the system including
security sensitive external hosts or localhost on the node.
Kubelet will be probing this set `Host` value which can
lead to blind SSRF attacks.

### Goals

* Add  Pod Security Admission (PSA) to enable admins to restrict
  users from creating probes with the `Host` field set to disallowed
  values. The only allowed values will be `127.0.0.1` and `::1`.
* The Baseline Pod Security Standard (PSS) will be updated to enforce
  blocking this field so that it helps with easier adoption for
  workload operators given this is a known issue we want to prevent.

### Non-Goals

* Removing `.Host` field from the API and dropping support (It is
  unsaid rule that nothing can get removed from core Kubernetes API)

## Proposal

There is a long term plan to deprecate the existing TCP and HTTP probe
types in the API to replace them with ones with slightly different semantics.
See [KEP-4559](https://github.com/kubernetes/enhancements/pull/4558) for more
details. Given the unsolvable security problems with the Host field,
we do not plan to offer it in the new types.

Meanwhile, the older API is never going to go away. So we also want to
add PSA to allow admins to be able to restrict users from creating
probes with the Host field set with disallowed values when using the
(about to be deprecated) API. This is implemented by
[kubernetes PR 125271](https://github.com/kubernetes/kubernetes/pull/125271)
that does exactly that.

Given there is still a use case where admins might be deploying the apiserver
or any controlplane host-networked pod service to have probes with `.Host` field set to
localhost (127.0.0.1). This is because there could be firewall rules blocking access to public nodeIP
for good reasons. Hence we would continue to allow for this use case meaning the only values
allowed on the `.Host` field would be `127.0.0.1` and `::1`. See [this snippet] for example.

[this snippet]: https://github.com/kubernetes/kops/blob/5dd2f468b46fda43f3a63ba1e6dc7c55c21919eb/nodeup/pkg/model/kube_apiserver.go#L603

### Risks and Mitigations

There might be users who depend on the `Host` field in
their existing probes which will continue to work and if
newly created probes also need the `Host` field to point
to an external destination then the admin can avoid enforcing
the PSA to block it.

## Design Details

Add a Baseline APILevel Pod Security Admission policy to allow admins of the
cluster to block users from setting `.host` field to disallowed values in the
following fields. The only allowed values will be `127.0.0.1` and `::1` and everything
else will be blocked.

* spec.containers[*].LivenessProbe.ProbeHandler.HTTPGet.Host
* spec.containers[*].ReadinessProbe.ProbeHandler.HTTPGet.Host
* spec.containers[*].StartupProbe.ProbeHandler.HTTPGet.Host
* spec.containers[*].LivenessProbe.ProbeHandler.TCPSocket.Host
* spec.containers[*].ReadinessProbe.ProbeHandler.TCPSocket.Host
* spec.containers[*].StartupProbe.ProbeHandler.TCPSocket.Host
* spec.containers[*].Lifecycle.PostStart.TCPSocket.Host // Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for backward compatibility.
* spec.containers[*].Lifecycle.PreStop.TCPSocket.Host // Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for backward compatibility.
* spec.containers[*].Lifecycle.PostStart.HTTPGet.Host
* spec.containers[*].Lifecycle.PreStop.HTTPGet.Host
* spec.initContainers[*].LivenessProbe.ProbeHandler.HTTPGet.Host
* spec.initContainers[*].ReadinessProbe.ProbeHandler.HTTPGet.Host
* spec.initContainers[*].StartupProbe.ProbeHandler.HTTPGet.Host
* spec.initContainers[*].LivenessProbe.ProbeHandler.TCPSocket.Host
* spec.initContainers[*].ReadinessProbe.ProbeHandler.TCPSocket.Host
* spec.initContainers[*].StartupProbe.ProbeHandler.TCPSocket.Host
* spec.initContainers[*].Lifecycle.PostStart.TCPSocket.Host // Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for backward compatibility.
* spec.initContainers[*].Lifecycle.PreStop.TCPSocket.Host // Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for backward compatibility.
* spec.initContainers[*].Lifecycle.PostStart.HTTPGet.Host
* spec.initContainers[*].Lifecycle.PreStop.HTTPGet.Host

### Test Plan

* Unit and E2E tests will be added to ensure the PSA works as expected

##### Prerequisite testing updates

None

##### Unit tests

Necessary unit tests will be added to the [PSA package] for
testing the new code.
Current test coverage status for the package is:
- `k8s.io/pod-security-admission/policy`: `2025-05-06` - `89.9%`
- `k8s.io/pod-security-admission/test`: `TBD` - `TBD`

[PSA package]: https://github.com/kubernetes/kubernetes/tree/master/staging/src/k8s.io/pod-security-admission/policy

##### Integration tests

The following integration tests will be added to verify the PSA validation logic:

1. Test that pods with `.host` field set to disallowed values in probes are rejected when PSA is enabled with baseline level
2. Test that pods with `.host` field set to allowed values in probes are accepted when PSA is enabled with baseline level
3. Test that pods without `.host` field set in probes are allowed when PSA is enabled with baseline level
4. Test that existing pods with `.host` field set continue to work when PSA is enabled
5. Test that pods with `.host` field set are allowed when PSA is disabled or using an older version

These tests will be added to:
- `test/integration/auth/podsecurity_test.go`
https://storage.googleapis.com/k8s-triage/index.html?test=TestPodSecurity

The integration tests will verify the PSA policy validation logic by:
- Creating test cases for each probe type (HTTPGet, TCPSocket) in a pod
- Testing each probe location (LivenessProbe, ReadinessProbe, StartupProbe, LifecycleHandler)
- Verifying the PSA policy enforcement at the baseline level
- Testing the behavior with different PSA configurations

##### e2e tests

There are no Pod Security specific E2E tests (we rely on integration test coverage instead),
but the Pod Security admission controller is enabled in E2E clusters,
and all E2E test namespaces are labeled with the enforcement label for Pod Security.

### Graduation Criteria

The PSA added will be done within a single release
and given there will be no feature gates for that,
there is no need for multi-release graduation criteria.
All related code will land within the same single release

### Upgrade / Downgrade Strategy

Any older pods with this field set to allowed values should
not be affected with the above solution.

Any older pods with this field set to disallowed values should
not be affected with the above solution. Only newer pods getting
created with the field will be alerted.

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
So if the admin sets pod-security.kubernetes.io/enforce-version: v1.34
along with pod-security.kubernetes.io/enforce: <LEVEL>
on a namespace this feature will get enabled.

###### Does enabling the feature change any default behavior?

* There is no effect on clusters where PSA is not enabled OR an older
PSA version is used.

* There is no effect on clusters where `.Host` probes are not used

* There is no effect on clusters where an older PSA versioning is being
used

* If users create new pod with `.Host` probes field set and the admin
has set baseline PSA level to `enforce` mode then the request will be
actively blocked and rejected. Existing pods with `.Host` probes
that are upgrading will not be impacted unless PSA level is set to
`enforce` mode.


###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

No

###### What happens if we reenable the feature if it was previously rolled back?

N/A

###### Are there any tests for feature enablement/disablement?

N/A since there is no feature gate

### Rollout, Upgrade and Rollback Planning


###### How can a rollout or rollback fail? Can it impact already running workloads?

* Running workloads/deployments that have `.Host` probes set when upgraded to
  the latest version where they get rolled-out, if the PSA enforce label is
  placed on the namespace of the workload, then the workload will fail to get created.
* If pod security label is not enabled on the namespace, then there is no
  impact on running workloads

###### What specific metrics should inform a rollback?

If your workloads are not rolling out due to the policy rejecting the request,
then cluster admins can use the [PSA denial metrics]. Example, the `pod_security_evaluations_total`
can indicate how many "deny" decisions were done based on number of policy evaluations that
occurred.

[PSA denial metrics]: https://kubernetes.io/docs/concepts/security/pod-security-admission/#metrics

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

N/A

### Monitoring Requirements

N/A

###### How can an operator determine if the feature is in use by workloads?

If pods have probes with `.Host` field set and PSA label is set on that pod's namespace
to a version where the new admission has been added, then it means the feature is enabled.

###### How can someone using this feature know that it is working for their instance?

Trying to create a pod with `.Host` field set in the probes will fail
like this:
```
Error from server (Forbidden): error when creating "psa/fail-case-pod.yaml": pods "liveness-http-pass" is forbidden: violates PodSecurity "restricted:latest": probeHost (container "liveness" uses probeHost 135.45.63.4)
```
Trying to rollout a deployment with `.Host` field set in probes will fail with the following status:
```
    - lastTransitionTime: "2025-06-17T06:17:36Z"
      lastUpdateTime: "2025-06-17T06:17:36Z"
      message: 'pods "hello-world-577c86d6dd-bs7nt" is forbidden: violates PodSecurity
        "restricted:latest": probeHost (container "hello-world" uses probeHost 135.45.63.4)'
      reason: FailedCreate
      status: "True"
      type: ReplicaFailure
    observedGeneration: 1
    unavailableReplicas: 1
```

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?
N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

cluster admins can use the [PSA denial metrics] to determine if something is
wrong with their workloads and services are not serving properly due to policy
enforcement.

[PSA denial metrics]: https://kubernetes.io/docs/concepts/security/pod-security-admission/#metrics

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

So if the admin sets `pod-security.kubernetes.io/enforce-version: v1.34`
on a namespace this feature will get enabled and workloads rolling out
with `.Host` probes set will be impacted. One of the remediation procedures to
get workloads into a healthy state would be:

* To pin the the [PSA namespace label] to a version prior to the version where this
  field is introduced (example set it to v1.33)
* Restart your workloads.

[PSA namespace label]: https://kubernetes.io/docs/concepts/security/pod-security-admission/#pod-security-admission-labels-for-namespaces

## Implementation History

## Drawbacks

N/A

## Alternatives

The alternative is to remove this field from the API after
its deprecated, but that's not a supported API action.
