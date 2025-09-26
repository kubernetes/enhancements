# KEP-5573: Move cgroup v1 to unsupported

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Enable fail-cgroup-v1 by default](#enable-fail-cgroup-v1-by-default)
  - [Update warning messages and events](#update-warning-messages-and-events)
  - [Documentation updates](#documentation-updates)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [x] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
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

Formally move cgroup v1 into an unsupported state in Kubernetes, building upon the maintenance mode introduced in KEP-4569. This enhancement will disable cgroup v1 support by default and provide clear messaging that cgroup v1 is no longer supported.

## Motivation

Following the transition of cgroup v1 support to maintenance mode in KEP-4569, the next logical step is to move cgroup v1 to an unsupported state. This aligns with the broader ecosystem's migration to cgroup v2, including major Linux distributions and the Linux kernel community's focus on cgroup v2 for new features and improvements.

The motivation builds on the rationale established in KEP-4569:
- The Linux kernel community has made cgroup v2 the focus for new features
- Major Linux distributions are phasing out cgroup v1 support
- systemd and other critical components are moving beyond cgroup v1
- cgroup v2 offers better functionality, more consistent interfaces, and improved scalability

By formally moving cgroup v1 to unsupported status, Kubernetes provides a clear signal to the community about the deprecation path and encourages migration to the more secure and efficient cgroup v2 technology.

### Goals

1. **Disable cgroup v1 support by default**: Set the kubelet flag `--disable-cgroupv1-support` to `true` by default, effectively making cgroup v1 unsupported unless explicitly enabled.

2. **Clear messaging**: Update warning messages and events to reflect that cgroup v1 is now unsupported rather than in maintenance mode.

3. **Documentation updates**: Update all relevant documentation to reflect the unsupported status of cgroup v1 and provide migration guidance.

4. **Preparation for removal**: This change prepares the codebase for eventual removal of cgroup v1 support in future releases.

5. **Community alignment**: Provide clear signals to the Kubernetes community about the deprecation timeline and encourage adoption of cgroup v2.

6. **Remove testing cgroup v1**: Moving to unsupported means that Kubernetes will no longer run tests on cgroup v1 nodes.

### Non-Goals

- Complete removal of cgroup v1 code (this will be addressed in a future KEP)
- Breaking existing clusters that explicitly enable cgroup v1 support
- Removing the ability to override the default behavior

## Proposal

This proposal builds upon the foundation laid by KEP-4569 (Moving cgroup v1 support into maintenance mode) and formally transitions cgroup v1 from maintenance mode to unsupported status.

### Risks and Mitigations

The primary risks involve potential disruptions for users who have not yet migrated to cgroup v2:

1. **Existing clusters running cgroup v1**: Users running Kubernetes on hosts with cgroup v1 will need to either:
   - Migrate their hosts to cgroup v2 (recommended)
   - Explicitly set `--disable-cgroupv1-support=false` to continue using cgroup v1 (not recommended)

2. **Workload compatibility**: Users depending on technologies that require specific versions for cgroup v2 support:
   - OpenJDK / HotSpot: jdk8u372, 11.0.16, 15 and later
   - NodeJs 20.3.0 or later
   - IBM Semeru Runtimes: jdk8u345-b01, 11.0.16.0, 17.0.4.0, 18.0.2.0 and later
   - IBM SDK Java Technology Edition Version (IBM Java): 8.0.7.15 and later
   - Third-party monitoring and security agents need to support cgroup v2

**Mitigations**:
- Provide comprehensive migration documentation and guidance
- Maintain the ability to override the default behavior with `--disable-cgroupv1-support=false`
- Clear warning messages when cgroup v1 is detected
- Community support through migration period
- Advance notice through multiple release cycles

## Design Details

This enhancement primarily involves configuration changes and messaging updates, building on the infrastructure already implemented in KEP-4569.

### Enable fail-cgroup-v1 by default

The key technical change is to modify the default value of the kubelet flag `--disable-cgroupv1-support` from `false` to `true`. This change will be implemented in the kubelet configuration types.

Current behavior:
```go
// Default: false (cgroup v1 support enabled by default)
DisableCgroupV1Support: false,
```

Proposed behavior:
```go
// Default: true (cgroup v1 support disabled by default)
DisableCgroupV1Support: true,
```

### Update warning messages and events

Update the warning messages and events introduced in KEP-4569 to reflect the new unsupported status:

From (maintenance mode):
```golang
klog.Warning("cgroup v1 detected. cgroup v1 support has been transitioned into maintenance mode, please plan for the migration towards cgroup v2. More information at https://git.k8s.io/enhancements/keps/sig-node/4569-cgroup-v1-maintenance-mode")
```

To (unsupported):
```golang
klog.Warning("cgroup v1 detected. cgroup v1 support is unsupported and will be removed in a future release. Please migrate to cgroup v2. More information at https://git.k8s.io/enhancements/keps/sig-node/5573-cgroup-v1-unsupported")
```

Similar updates will be made to corresponding events.

### Documentation updates

Update all relevant documentation across the Kubernetes ecosystem:
- Kubernetes.io documentation
- Kubelet configuration documentation  
- Migration guides
- Release notes
- Blog posts about the transition

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.
All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.
[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

All existing cgroup v2 test jobs must continue to pass. Tests should verify that:
1. The default behavior correctly disables cgroup v1 support
2. The override flag `--disable-cgroupv1-support=false` continues to work
3. Appropriate warning messages are displayed when cgroup v1 is detected

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

Unit tests should cover:
- Default configuration values
- Warning message generation
- Event creation for cgroup v1 detection
- Configuration override behavior

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

Integration tests will verify the end-to-end behavior of the configuration changes and ensure proper interaction between kubelet components.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

1. Continue monitoring cgroup v2 CI jobs to ensure stability
2. Add specific tests for the new default behavior
3. Ensure all new tests use cgroup v2 hosts
4. Maintain minimal testing for override scenarios where cgroup v1 is explicitly enabled

### Graduation Criteria

#### Alpha

- Default value for `--disable-cgroupv1-support` changed to `true`
- Updated warning messages and events for unsupported status
- Documentation updates in kubernetes/enhancements repository
- Basic test coverage for new default behavior

#### Beta

- Comprehensive testing across multiple scenarios
- Updated documentation in kubernetes/website
- Community feedback incorporated
- Stable behavior across different environments

#### GA

- All tests passing consistently
- Complete documentation coverage
- Community migration guidance available
- Blog post announcing the change
- Preparation for future removal KEP

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

**Upgrade considerations**:
- Clusters upgrading to Kubernetes v1.35+ on cgroup v1 hosts will fail to start kubelet unless `--disable-cgroupv1-support=false` is explicitly set
- Administrators should migrate to cgroup v2 before upgrading or explicitly set the override flag
- Clear documentation and communication about this breaking change

**Downgrade strategy**:
- Downgrading to versions prior to this change will restore the previous default behavior
- No additional configuration changes needed for downgrade

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

This change only affects kubelet behavior and does not involve coordination with other control plane components. The change is backward compatible for users who explicitly configure cgroup v1 support.

## Production Readiness Review Questionnaire

<!--
Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

This is a default configuration change. The feature can be controlled via the kubelet flag:
- To disable cgroup v1 support (default): `--disable-cgroupv1-support=true` 
- To enable cgroup v1 support (override): `--disable-cgroupv1-support=false`

###### Does enabling the feature change any default behavior?

Yes, this change modifies the default behavior. Previously, cgroup v1 support was enabled by default. After this change, cgroup v1 support will be disabled by default.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, users can set `--disable-cgroupv1-support=false` to re-enable cgroup v1 support.

###### What happens if we reenable the feature if it was previously rolled back?

Re-enabling cgroup v1 support will restore the previous behavior, allowing kubelet to run on cgroup v1 hosts.

###### Are there any tests for feature enablement/disablement?

Yes, unit and integration tests will cover both the default behavior and the override scenarios.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout fail? Can it impact already running workloads?

**Potential failure scenarios**:
1. Kubelet fails to start on cgroup v1 hosts without the override flag
2. Existing clusters running on cgroup v1 experience service disruption during upgrade

**Impact on running workloads**:
- Existing workloads on upgraded nodes will be impacted if the node uses cgroup v1 and the override flag is not set
- The kubelet will fail to start, causing the node to become unavailable

###### What specific metrics should inform a rollback?

- `kubelet_cgroup_version` metric showing unexpected cgroup version distribution
- Increased node failures or unavailability
- Kubelet startup failures indicating cgroup v1 detection

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Testing will include:
- Upgrade scenarios on both cgroup v1 and cgroup v2 hosts
- Rollback to previous versions
- Override flag functionality

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

This change moves cgroup v1 from maintenance mode to unsupported status but does not remove any APIs or flags. The `--disable-cgroupv1-support` flag remains available for override purposes.

### Monitoring Requirements

###### How can someone using this feature know that it is working for their instance?

Users can monitor:
- Kubelet logs for warnings about cgroup v1 detection
- Events related to cgroup v1 unsupported status
- The `kubelet_cgroup_version` metric to verify cgroup version usage

###### How can an operator determine if the feature is in use by workloads?

Operators can use the `kubelet_cgroup_version` metric to determine cgroup version distribution across their cluster and monitor logs/events for cgroup v1 warnings.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- Node availability and kubelet health status
- `kubelet_cgroup_version` metric distribution
- Absence of cgroup v1 related warning messages

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

- 99.9% of nodes should be running cgroup v2
- Zero cgroup v1 related warnings in production clusters
- Node availability should remain consistent after migration

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

The existing `kubelet_cgroup_version` metric from KEP-4569 provides sufficient observability for this change.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No external dependencies. This change only affects kubelet configuration defaults.

### Scalability

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No, this is a configuration default change that does not impact resource usage.

###### Will enabling / using this feature result in any new API calls?

No new API calls are introduced.

###### Will enabling / using this feature result in introducing new API types?

No new API types are introduced.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No new cloud provider calls are made.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No impact on existing API objects.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No impact on existing operation timing.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No increase in resource usage. The change may actually improve performance by defaulting to the more efficient cgroup v2.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

This feature operates at the kubelet level and does not depend on API server or etcd availability.

###### What are other known failure modes?

**Failure mode**: Kubelet fails to start on cgroup v1 hosts
- **Detection**: Kubelet startup logs and node status
- **Mitigation**: Set `--disable-cgroupv1-support=false` or migrate to cgroup v2
- **Diagnostics**: Kubelet logs will clearly indicate cgroup v1 detection and unsupported status
- **Testing**: Covered in unit and integration tests

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check kubelet logs for cgroup-related error messages
2. Verify cgroup version on affected nodes using `kubelet_cgroup_version` metric
3. If cgroup v1 is detected, either migrate to cgroup v2 or set override flag
4. Monitor node availability and kubelet health status

## Implementation History

- **2025-09-26**: KEP for moving cgroup v1 to unsupported status created

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

1. **Breaking change**: This represents a breaking change for clusters running on cgroup v1 hosts that upgrade without preparation.

2. **Migration burden**: Users who have not yet migrated to cgroup v2 will be forced to either migrate or explicitly override the default behavior.

3. **Ecosystem readiness**: Some users may still rely on environments or workloads that are not fully ready for cgroup v2.

4. **Support burden**: Increased support requests from users who encounter issues during the transition.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough detail to
express the idea and why it was not acceptable.
-->

1. **Continue maintenance mode longer**: Keep cgroup v1 in maintenance mode for additional releases to provide more migration time. This was ruled out because it delays the necessary ecosystem transition and maintains technical debt.

2. **Immediate removal**: Completely remove cgroup v1 support without an unsupported phase. This was ruled out as too aggressive and would break existing clusters without providing a migration path.

3. **Opt-in cgroup v2**: Require explicit configuration to enable cgroup v2 instead of disabling cgroup v1 by default. This was ruled out because it doesn't provide clear signals about the deprecation path and slows adoption of the preferred technology.

4. **Feature gate approach**: Use a feature gate instead of a kubelet flag. This was ruled out because kubelet flags provide more direct control over the behavior and are more appropriate for this type of configuration change.