# KEP-1967: Downward API HugePages

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
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This KEP exposes hugepages in the downward API.

## Motivation

Pods are unable to know their hugepage request or limits via the downward API.  HugePages
are a natively supported resource in Kubernetes and should be visible in downward API
consistent with other resources like cpu, memory, ephemeral-storage.

### Goals

- Add support for hugepage requests and limits for all page sizes in downward API

### Non-Goals

- Change any other aspect of hugepage support

## Proposal

Define a new feature gate: `DownwardAPIHugePages`.

If enabled, the `kube-apiserver` will allow pod specifications to make use
of hugepages in downward API when the feature gate is enabled.  The `kubelet`
will add support for hugepages in the downward API independent of the feature
gate.

### Risks and Mitigations

The primary risk for this proposal is that it loosens validation for Pods.

The mitigation proposed is as follows:

- Add support for the new fields in `kubelet` by default.  This is considered
low risk as the code is inert when pods do not use the tokens, and the subsystem
in the kubelet is localized.
- The `kube-apiserver` will have the feature gate disabled by default for 2
releases until we know all supported skew scenarios result in all kubelets having
the supported code present.

When the gate is enabled, the `kube-apiserver` will permit the newly allowed
values in all creation and update scenarios.  When the gate is disabled, the
new values are permitted only in updates of objects which already contain
the new values.  Use in creation of in updates of objects which do not
already use the new values will fail validation.

## Design Details

Add support for `requests.hugepages-<pagesize>` and `limits.hugepages-<pagesize>`
to downward API consistent with cpu, memory, and ephemeral storage.  Enable the
support by default in the kubelet, but gate its usage by default in the `kube-apiserver`
for 2 releases to ensure all nodes in the cluster have been proper support.

It is important to remember that `hugepages-<pagesize>` is not a resource
that is subject to overcommit.  A pod must have a matching request and limit
for an explicit `hugepages-<pagesize>` in order to consume hugepages.  Absent
an explicit request, no `hugepages-<pagesize>` is provided to a pod.

The `kube-apiserver` will not require pods to make an explicit `hugepages-<pagesize>`
request in its pod spec in order to use the field in the downward API.  The rationale
for this behavior is that pod templates for specific workload types may support
running with or without `hugepages-<pagesize>` made available to them and as a result,
it may include both memory and hugepages in the downward API in order to know how to adjust.
The `kubelet` will ensure that the downward API value projected into the container for
a specific `hugepages-<pagesize>` will match what is provided with its bounding pod
and or container cgroup.

### Test Plan

Unit and e2e testing will be added consistent with other resources in downward API.

e2e testing will only function if a node in the cluster exposes hugepages, otherwise,
it will gracefully skip (as expected).

### Graduation Criteria

#### Alpha

- Feature gate is present and enforced in kube-apiserver
- Validation logic is in-place in kube-apiserver
- Kubelet has support for projecting the value in the pod
- unit testing for downward API enhancement

#### Alpha -> Beta Graduation

- Added support in kube-apiserver protected by feature gate
- Added support in kubelet for 2 releases.
- e2e testing for hosts with hugepages enabled

#### Beta -> GA Graduation

- Enable support by default one release after kube-apiserver feature gate is enabled in beta.

### Upgrade / Downgrade Strategy

The kubelet will have the support for 2 releases before its
enabled in the kube-apiserver.  This ensures that pods cannot
get accepted in the platform for which nodes do not have support.

### Version Skew Strategy

The kubelet will have the support for 2 releases before its
enabled in the kube-apiserver.  This ensures that pods cannot
get accepted in the platform for which nodes do not have support.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: DownwardAPIHugePages
    - Components depending on the feature gate: kube-apiserver
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? No

* **Does enabling the feature change any default behavior?**
Yes, the kube-apiserver will admit pods that use the new downward API support.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?** Yes
Only if pods were not admitted that used the feature.

* **What happens if we reenable the feature if it was previously rolled back?**
Nothing.  New pods will now accept the new fields in admission.

* **Are there any tests for feature enablement/disablement?**
No, this will be handled by coordinating support in the kubelet.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**
If all kubelets in a cluster do not have support for hugepages enabled
prior to accepting pods in the kube-apiserver that use it in the downward api,
a node may not start with the downward api information made available.  It would
impact the operating environment for the application and not the cluster.

* **What specific metrics should inform a rollback?**
None.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
I do not believe this is applicable.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.
No, validation is loosened but coordinated across N-2 releases.

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**
An operator could audit pods that use the new downward API tokens.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
This does not seem relevant to this feature.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
This does not seem relevant to this feature.

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
No.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
No

### Scalability

* **Will enabling / using this feature result in any new API calls?**
No.

* **Will enabling / using this feature result in introducing new API types?**
No

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
No

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
No

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
No

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
No

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**
No impact.

* **What are other known failure modes?**
Not applicable.

* **What steps should be taken if SLOs are not being met to determine the problem?**
Not applicable

## Implementation History

## Drawbacks

None.

## Alternatives

None.

## Infrastructure Needed (Optional)

None.