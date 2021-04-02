# KEP-2595: Expanded DNS Configuration

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
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
- [ ] (R) Graduation criteria is in place
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

Allow kubernetes to have expanded DNS(Domain Name System) configuration.

## Motivation

Kubernetes today limits DNS configuration according to [the obsolete
criteria](https://access.redhat.com/solutions/58028). As recent DNS resolvers
allow an arbitrary number of search paths, a new feature gate
`ExpandedDNSConfig` will be introduced. With this feature, kubernetes allows
more DNS search paths and longer list of DNS search paths to keep up with recent
DNS resolvers.

Confirmed that expanded DNS configuration is supported by
- `glibc 2.17-323`
- `glibc 2.28`
- `musl libc 1.22`
- `pure Go 1.10 resolver`
- `pure Go 1.16 resolver`

### Goals

- Make `kube-apiserver` allow expanded DNS configuration when validating Pod's
or PodTemplate's `DNSConfig`
- Make `kubelet` allow expanded DNS configuration when validating `resolvConf`
- Make `kubelet` allow expanded DNS configuration when validating actual DNS
resolver configuration composed of `cluster domain suffixes`(e.g.
    default.svc.cluster.local, svc.cluster.local, cluster.local), kubelet's
`resolvConf` and Pod's `DNSConfig`

### Non-Goals

- Remove limitation on DNS search paths completely
- Let cluster administrators limit the number of search paths or the length of
DNS search path list to an arbitrary number

## Proposal

- Expand `MaxDNSSearchPaths` to 32
- Expand `MaxDNSSearchListChars` to 2048

### User Stories (Optional)

#### Story 1

#### Story 2

### Notes/Constraints/Caveats (Optional)

This enhancement relaxes the validation of `Pod` and `PodTemplate`. Once the
feature is activated, it must be carefully disabled. Otherwise, the objects left
over from the previous version which have the expanded DNS configuration can be
problematic.

### Risks and Mitigations

There may be some environments(DNS resolver or others) that break without
current limitations. At this point, it is fair to open a bug, so they can fix it.

## Design Details

- Declare and define `MaxDNSSearchPathsExpanded` to `32` and
`MaxDNSSearchListCharsExpanded` to `2048`
- Add the feature gate `ExpandedDNSConfig` (see [Feature Enablement and
Rollback](#feature-enablement-and-rollback))
- If the feature gate `ExpandedDNSConfig` is enabled, replace
`MaxDNSSearchPaths` with `MaxDNSSearchPathsExpanded` and replace
`MaxDNSSearchListChars` with `MaxDNSSearchListCharsExpanded` to allow expanded
DNS configuration

### Test Plan

- Add unit tests of validating expanded DNS config

### Graduation Criteria

#### Alpha -> Beta Graduation

- Address feedback from alpha
- Sufficient testing

#### Beta -> GA Graduation

- Address feedback from beta
- Sufficient number of users using the feature
- Close on any remaining open issues & bugs

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

In clusters with a replicated control plane, all kube-apiservers should enable
or disable the expanded DNS configuration feature.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

- **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `ExpandedDNSConfig`
    - Components depending on the feature gate:
      - `kubelet`
      - `kube-apiserver`
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

- **Does enabling the feature change any default behavior?**

Enabling this feature allows kubernetes to have objects(Pod or PodTemplate) with
the expanded DNS configuration.

- **Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?**

Yes, the feature can be disabled by disabling the feature gate.

Once the feature is disabled, kube-apiserver will reject the pod having expanded
DNS configuration and kubelet will create a resolver configuration excluding the
overage.

- **What happens if we reenable the feature if it was previously rolled back?**

It should continue to work as expected.

- **Are there any tests for feature enablement/disablement?**

We will add unit tests.

### Rollout, Upgrade and Rollback Planning

- **How can a rollout fail? Can it impact already running workloads?**

N/A

- **What specific metrics should inform a rollback?**

N/A

- **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

N/A

- **Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?**

N/A

### Monitoring Requirements

- **How can an operator determine if the feature is in use by workloads?**

N/A

- **What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

N/A

- **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

N/A

- **Are there any missing metrics that would be useful to have to improve observability of this feature?**

N/A

### Dependencies

- **Does this feature depend on any specific services running in the cluster?**

N/A

### Scalability

- **Will enabling / using this feature result in any new API calls?**

N/A

- **Will enabling / using this feature result in introducing new API types?**

N/A

- **Will enabling / using this feature result in any new calls to the cloud provider?**

N/A

- **Will enabling / using this feature result in increasing size or count of the existing API objects?**

N/A

- **Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?**

N/A

- **Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?**

N/A

### Troubleshooting

- **How does this feature react if the API server and/or etcd is unavailable?**

- **What are other known failure modes?**

- **What steps should be taken if SLOs are not being met to determine the problem?**

## Implementation History

- 2021-03-26: [Initial
discussion at #100583](https://github.com/kubernetes/kubernetes/pull/100583)

## Drawbacks

## Alternatives

- Remove the limitation of DNS search paths completely
- Make `MaxDNSSearchPaths` and `MaxDNSSearchListChars` configurable

## Infrastructure Needed (Optional)
