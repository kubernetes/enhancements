# KEP-4004: Deprecate the kubeProxyVersion field of v1.Node

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Versioned API Change: NodeStatus v1 core](#versioned-api-change-nodestatus-v1-core)
  - [NodeStatus Internal Representation](#nodestatus-internal-representation)
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

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
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

Deprecate the `status.nodeInfo.kubeProxyVersion` field of v1.Node

## Motivation

This field is not accurate, the field is set by kubelet, which does not actually know the kube-proxy version,
or even if kube-proxy is running.

### Goals

- Mark `status.nodeInfo.kubeProxyVersion` deprecated.
- Find any places use it and get them to stop.
- Make kubelet stop setting the field.

### Non-Goals

- Having kube-proxy itself set `status.nodeInfo.kubeProxyVersion`. Components should not
  be trying to figure out what Service features are available based on the kube-proxy
  version anyway, since some clusters may run an alternative service proxy implementation.
  If there is a need for a Service proxy feature discovery feature in the future, then
  that would need to be designed at that time.

- Changing `v1.Node` validation to *require* that the field be unset.

## Proposal

The proposal is to deprecate the `kubeProxyVersion` field of `NodeStatus`, and to have
kubelet set it to an empty string, rather than the kubelet version, in the future.

This field was used by the GCP cloud provider up until 1.28 for the legacy built-in cloud provider ([kubernetes #117806], and up until 1.29 for the external cloud-provider ([cloud-provider-gcp #533]). It may also be used by other components. Thus, we will use a feature gate to protect it until all components are fixed.

[kubernetes #117806]: https://github.com/kubernetes/kubernetes/pull/117806
[cloud-provider-gcp #533]: https://github.com/kubernetes/cloud-provider-gcp/pull/533

### Risks and Mitigations

## Design Details

### Versioned API Change: NodeStatus v1 core

Mark the kubeProxyVersion field as Deprecated.

```
// Deprecated: KubeProxy Version reported by the node.
KubeProxyVersion string `json:"kubeProxyVersion" protobuf:"bytes,8,opt,name=kubeProxyVersion"`
```

### NodeStatus Internal Representation

Mark the kubeProxyVersion field as Deprecated.

```
// Deprecated: KubeProxy Version reported by the node.
KubeProxyVersion string
```

### Test Plan

[x ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates


##### Unit tests

- Add a test to `TestVersionInfo` in `pkg/kubelet/nodestatus` to see if FeatureGate behaves as expected when it is turned on or off.
- If DisableNodeKubeProxyVersion FeatureGate is enabled:
  - `status.nodeInfo.kubeProxyVersion` will remain unset if it is initially unset.
  - `status.nodeInfo.kubeProxyVersion` will be cleared if it is set.
- Else:
  - `status.nodeInfo.kubeProxyVersion` will be set to the same value as `status.nodeInfo.kubeletVersion`

##### Integration tests

- N/A

##### e2e tests

 - N/A

### Graduation Criteria

#### Alpha

- Created the feature gate, disabled by default.
- Started looking for components that might be using the deprecated field.

#### Beta

- Make sure that upgrades to the Beta version will work across supported levels of [version skew](https://kubernetes.io/releases/version-skew-policy/).

#### GA

- No issues reported during two releases.

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DisableNodeKubeProxyVersion
  - Components depending on the feature gate: kubelet

###### Does enabling the feature change any default behavior?

Yes, it makes kubelet stop setting `node.status.nodeInfo.kubeProxyVersion`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Using the featuregate is the only way to enable/disable this feature.

###### What happens if we reenable the feature if it was previously rolled back?

The feature should continue to work just fine.

###### Are there any tests for feature enablement/disablement?

In addition to the unit tests, we will manually confirm that restarting kubelet with the
feature gate toggled has the expected behavior (that it sets `kubeProxyVersion` if the
feature gate is disabled, even if it doesn't also need to set `kubeletVersion`, and that
it clears `kubeProxyVersion` if the feature gate is enabled).

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

If there is a component that we didn't know about using kubeProxyVersion, then it may fail in some unknown way when the administrator upgrades to a version with that field disabled. It should be possible to just roll back in this case.

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

* N/A

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

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

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

\- 2023-05-15: Initial draft KEP

## Drawbacks

## Alternatives

