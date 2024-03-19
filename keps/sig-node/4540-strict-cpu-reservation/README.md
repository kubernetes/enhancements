# KEP-4540: Add CPUManager policy option to restrict reservedSystemCPUs to system daemons and interrupt processing

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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Starting with Kubernetes 1.22, a new `CPUManager` flag has facilitated the use of `CPUManager` Policy options(#2625) which enable users to customize their behavior based on workload requirements without having to introduce an entirely new policy.
These policy options work together to ensure an optimized cpu set is allocated for workloads running on a cluster.
The three policy options that already exist are `full-pcpus-only`(#2625) and `distribute-cpus-across-numa` (#2902) and `align-by-socket` (#3327).
With this KEP, a new `CPUManager` policy option is introduced which ensures that reservedSystemCPUs are strictly reserved for system daemons or interrupt processing and are not used by burstable and best-effort pods.


## Motivation

The static policy is used to reduce latency or improve performance. If you want to move system daemons or interrupt processing to dedicated cores, the obvious way is use the reservedSystemCPUs option. But in current implementation this isolation is implemented only for guaranteed pods not for burstable and best-effort pods.
Admission is only comparing the cpu requests against the allocatable cpus. Since the cpu limit are higher than the request, it allows burstable and best-effort pods to use up the capacity of reservedSystemCPUs option and cause the OS/Systemd services to starve in real life deployments.

### Goals
 * Ensure the reservedSystemCPUs are only used by system daemons or interrupt processing not by pods.

### Non-Goals

## Proposal

We propose to add a new `CPUManager` policy option called `strict-cpu-reservation` to the `static` policy of `CPUManager`.
With this policy option, we remove the reserved cores from the list of all available cores at the stage of calculation DefaultCPUSet. As a result, burstable and best-effort containers are launched with a cpuset in which the reserved cores are excluded. 

### Risks and Mitigations

The risks of adding this new feature are quite low.
It is isolated to a specific policy option within the `CPUManager`, and is protected both by the option itself, as well as the `CPUManagerPolicyAlphaOptions` feature gate (which is disabled by default).

## Design Details

When `strict-cpu-reservation` is enabled as a policy option, we remove the reserved cores from the list of all available cores at the stage of calculation DefaultCPUSet.

Feature impact can be illustrated as following: 

With the following Kubelet configuration:

```yaml
kind: KubeletConfiguration
apiVersion: kubelet.config.k8s.io/v1beta1
featureGates:
  CPUManagerPolicyOptions: true
  CPUManagerPolicyAlphaOptions: true
cpuManagerPolicy: static
cpuManagerPolicyOptions:
  strict-cpu-reservation: "true"
reservedSystemCPUs: "0,1,40,41,20,21,60,61"
```

When `strict-cpu-reservation` is disabled:
```console
# cat /var/lib/kubelet/cpu\_manager\_state
{"policyName":"static","defaultCpuSet":"0-79","checksum":1241370203}
```

When `strict-cpu-reservation` is enabled:
```console
# cat /var/lib/kubelet/cpu\_manager\_state
{"policyName":"static","defaultCpuSet":"2-19,22-39,42-59,62-79","checksum":3758876046}
```

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

- `k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/policy_static.go`: `03-18-2024` - `91.1`

##### Integration tests

- These cases will be added in the existing integration tests:
  - Feature gate enable/disable tests
  - `strict-cpu-reservation` policy option works as expected.
  - `strict-cpu-reservation` policy option works with existing polity options.

##### e2e tests

- These cases will be added in the existing e2e tests:
  - Feature gate enable/disable tests
  - `strict-cpu-reservation` policy option works as expected.
  - `strict-cpu-reservation` policy option works with existing polity options.

### Graduation Criteria

#### Alpha

- [X] Implement the new policy option.
- [X] Ensure proper unit tests are in place.
- [X] Ensure proper e2e node tests are in place.

#### Beta

- [X] Gather feedback from consumers of the new policy option.
- [X] Verify no major bugs reported in the previous cycle.

#### GA

- [X] Allow time for feedback (1 year).
- [X] Make sure all risks have been addressed.

### Upgrade / Downgrade Strategy

The new policy option is opt-in and orthogonal to the existing ones.

### Version Skew Strategy

No changes needed

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

The /var/lib/kubelet/cpu\_manager\_state needs be removed when changing the value of `strict-cpu-reservation`.

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `CPUManagerPolicyAlphaOptions`
  - Components depending on the feature gate: `kubelet`
- [X] Change the kubelet configuration to set a `CPUManager` policy of `static` and a `CPUManager` policy option of `strict-cpu-reservation`
  - Will enabling / disabling the feature require downtime of the control
    plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
        Yes -- removing /var/lib/kubelet/cpu\_manager\_state and restarting kubelet are required.


###### Does enabling the feature change any default behavior?

Yes. Reserved CPU cores for system usage will be strictly used for system daemons and interrupt processing not available for workloads.

The feature is only enabled when all following conditions are met:
1. The `CPUManagerPolicyAlphaOptions` feature gate must be enabled
2. The `static` `CPUManager` policy must be selected
3. The new `strict-cpu-reservation` policy option must be selected
4. The `reservedSystemCPUs` is not empty

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the feature can be disabled by either:
1. Disabling the `CPUManagerPolicyAlphaOptions` feature gate
2. Switching the `CPUManager` policy to `none`
3. Removing `strict-cpu-reservation` from the list of `CPUManager` policy options

###### What happens if we reenable the feature if it was previously rolled back?

The feature will be enabled regardless it is enabled for the first time or not.

###### Are there any tests for feature enablement/disablement?

- A specific e2e test will demonstrate that the default behaviour is preserved when the feature gate is disabled, or when the feature is not used (2 separate tests)

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Inspect the kubelet configuration -- check the presence of the feature gate and usage of the new policy option.

###### How can someone using this feature know that it is working for their instance?

Inspect the cgroup/cpuset configuration of burstable and best-effort pods -- check the reserved cores are not used by them.

Below is an example when Cgroup v1 is used:
```console
# cat /sys/fs/cgroup/cpuset/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod50ca196f\_866c\_4070\_844a\_16d466b957ae.slice/cri-containerd-b511f2ad6e959a0a369578d335c8f0c3f01a2108c9157bce8d239a8d8d6a0d64.scope/cpuset.cpus
2-19,22-39,42-59,62-79 
```

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This feature to protect infrastructure services, when they are restricted to run on limited number of CPU cores, from bursty workloads.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Run `top -H` to observe reservedSystemCPUs are used by system daemons and interrupt processing only.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

None

### Dependencies

None

###### Does this feature depend on any specific services running in the cluster?

No

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

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

Incease kubelet log level and check kubelet log for errors.

Below is how to check kubelet log when it runs as a systemd service:
```console
journalctl _SYSTEMD_INVOCATION_ID=`systemctl show -p InvocationID --value kubelet.service`
```


###### How does this feature react if the API server and/or etcd is unavailable?

There is no impact on this node local feature.

###### What are other known failure modes?

There is no known failure mode since this feature changes available CPU core for burstable and best-effort pods only.

###### What steps should be taken if SLOs are not being met to determine the problem?

You can safely disable the feature.

## Implementation History

- 2023-03-08: Initial KEP created

## Drawbacks

## Alternatives

## Infrastructure Needed (Optional)
