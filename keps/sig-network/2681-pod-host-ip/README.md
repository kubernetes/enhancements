# KEP-2681: Field status.hostIPs added for Pod

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Versioned API Change: PodStatus v1 core](#versioned-api-change-podstatus-v1-core)
  - [PodStatus Internal Representation](#podstatus-internal-representation)
  - [Maintaining Compatible Interworking between Old and New Clients](#maintaining-compatible-interworking-between-old-and-new-clients)
  - [Container Environment Variables](#container-environment-variables)
  - [Test Plan](#test-plan)
    - [Expected behavior](#expected-behavior)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
    - [Removing a Deprecated Flag](#removing-a-deprecated-flag)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
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

The proposal aims to improve the Pod's ability to obtain the address of the node

## Motivation

### Goals

- Field `status.hostIPs` added for Pod
- Downward API support for `status.hostIPs`

### Non-Goals

## Proposal

### User Stories

#### Story 1

Applications that originally used IPv4 migrated to IPv6 during the dual-stack transition phase, 
For smooth migration, IP-related attributes should have both IPv4 and IPv6.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

### Versioned API Change: PodStatus v1 core

In order to maintain backwards compatibility for the core V1 API, this proposal
retains the existing (singular) "HostIP" field in the core V1 version of the
PodStatus V1 core API and adds a new array of structures that store host IPs
along with associated metadata for that IP.

``` go
    // HostIP represents the IP address of a host.
    // IP address information. Each entry includes:
    //    IP: An IP address allocated to the host.
    type HostIP struct {
      // ip is an IP address (IPv4 or IPv6) assigned to the host
      IP string
    }

    // HostIPs holds the IP addresses allocated to the host. If this field is specified, the 0th entry must
    // match the hostIP field. This list is empty if no IPs have been allocated yet.
    // +optional
    HostIPs []HostIP
```

### PodStatus Internal Representation

PodStatus internally indicates that the original use of HostIP will remain unchanged,
and HostIPs is added for subsequent use

``` go
    // HostIP address information for entries in the (plural) HostIPs field.
    // Each entry includes:
    //    IP: An IP address allocated to the host.
    type HostIP struct {
      // ip is an IP address (IPv4 or IPv6) assigned to the host
      IP string `json:"ip,omitempty" protobuf:"bytes,1,opt,name=ip"`
    }

    // HostIPs holds the IP addresses allocated to the host. If this field is specified, the 0th entry must
    // match the hostIP field. This list is empty if no IPs have been allocated yet.
    // +optional
    // +patchStrategy=merge
    // +patchMergeKey=ip
    HostIPs []HostIP `json:"hostIPs,omitempty" protobuf:"bytes,14,rep,name=hostIPs" patchStrategy:"merge" patchMergeKey:"ip"`
```

### Maintaining Compatible Interworking between Old and New Clients

[See the making a singular field plural](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#making-a-singular-field-plural)

### Container Environment Variables

The Downward API [status.hostIP](https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#capabilities-of-the-downward-api)
will preserve the existing single IP address, and will be set to the default IP for each pod.
A new pod API field named `status.hostIPs` will contain a list of IP addresses.
The new pod API will have a slice of structures for the additional IP addresses. 
Kubelet will translate the pod structures and return `status.hostIPs` as a comma-delimited string.

Here is an example of how to define a pluralized `MY_HOST_IPS` environmental
variable in a pod definition yaml file:

``` yaml
- name: MY_HOST_IPS
  valueFrom:
    fieldRef:
      fieldPath: status.hostIPs
```

This definition will cause an environmental variable setting in the pod similar
to the following:

```
# PodHostIPs FeatureGate is enabled
MY_HOST_IPS=fd00:10:20:0:3::3,10.20.3.3

# PodHostIPs FeatureGate is disabled
MY_HOST_IPS=
```

### Test Plan

Test whether FeatureGate behaves as expected when it is turned on or off
Test whether Downward API supports `status.hostIPs`

#### Expected behavior

- If PodHostIPs FeatureGate is enabled:
  - `status.hostIPs` there will be all host IPs (IPv4 and IPv6)
- Else:
  - `status.hostIPs` will be empty and the ApiServer will reject Downward API `status.hostIPs`, 
    if Pods already existing Downward API `status.hostIPs`, ensure to ignore it and not affect others.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
  - Add `status.hostIPs` to the PodStatus API
  - Downward API support for `status.hostIPs`
- Basic units and e2e tests completed and enabled

#### Alpha -> Beta Graduation

- Gather feedback from developers and users
- Expand the e2e tests with more scenarios
- Tests are in Testgrid and linked in the KEP

#### Beta -> GA Graduation

- 2 examples of end users using this field
- Allowing time for feedback

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: PodHostIPs
  - Components depending on the feature gate: `kube-apiserver`, `kubelet`
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Using the featuregate is the only way to enable/disable this feature.

###### What happens if we reenable the feature if it was previously rolled back?

The feature should continue to work just fine.

###### Are there any tests for feature enablement/disablement?

No, these will be introduced in the Alpha phase.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

If the dependent [KEP 563-dual-stack](https://github.com/kubernetes/enhancements/issues/563) is wrong, or could not get IP of host, or setting the field is crashing, this feature may fail to rollout/rollback.
The field is only informative, it doesn't affect running workloads.

###### What specific metrics should inform a rollback?

The `status.hostIPs` field in Pod is empty, or frequently updated, or cause any other to crash.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

TBD.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Pod has a `status.hostIPs` field and use it in downwardAPI to expose it. 

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: 
- [x] API .status
  - Other field: `pod.status.hostIPs` is not empty.
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- The feature is only informative, no increased failure rates during use the feature.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- TBD

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No additional metrics needed for this new API field.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

N/A

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes.
  
- API type(s): Pod
- Estimated increase in size:
  - New field in Status about 8 bytes, additional bytes based on whether IPv4(about 4 bytes) or IPv6(about 16 bytes) exists

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

### Troubleshooting

N/A

## Implementation History

- 2021-05-06: Initial KEP

## Drawbacks

N/A

## Alternatives

N/A

## Infrastructure Needed (Optional)

N/A
