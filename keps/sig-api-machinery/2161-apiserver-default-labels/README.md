# KEP-2161 : Immutable label selectors for all namespaces

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  * [Goals](#goals)
  * [Non-Goals](#non-goals)
- [Proposal](#proposal)
  * [User Stories (Optional)](#user-stories--optional-)
  * [Notes/Constraints/Caveats (Optional)](#notes-constraints-caveats--optional-)
  * [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  * [Test Plan](#test-plan)
  * [Graduation Criteria](#graduation-criteria)
  * [Upgrade / Downgrade Strategy](#upgrade---downgrade-strategy)
  * [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  * [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  * [Rollout, Upgrade and Rollback Planning](#rollout--upgrade-and-rollback-planning)
  * [Monitoring Requirements](#monitoring-requirements)
  * [Dependencies](#dependencies)
  * [Scalability](#scalability)
  * [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed--optional-)
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
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
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

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

## Motivation

In https://github.com/kubernetes/enhancements/issues/2112 , we discussed several ways to implement network policies that select namespaces by name, rather than by relying on labels which may be unknown, or uneditable by people writing policies, or untrustworthy.

This was inspired by a broad community ask to target namespaces in this manner, as evidenced by issues such as https://github.com/kubernetes/kubernetes/issues/88253, and the larger set of user feedback obtained in  https://github.com/jayunit100/network-policy-subproject/blob/master/p0_user_stories.md.

It turns out all solutions:
Break the old API in fundamental ways that are questionably WRT cost/benefit OR
Add nesting complexity to the network policy API, wherein certain “subsets” of the API were not usable together.  This is not “bad”, but it is suboptimal and introduces redundant semantics.  For example, you may need a way to declare namespaceAsName selectors, which is distinct from namespace label selectors.
Example:  The obvious API, whe combined with a podSelector, decreases security of a pod - an old client would be more open than users would expect them to be.  This is thoroughly debated in the KEP.  Fail-closed is a requirement.

If a podSelector is an enabled for a network policy peer, then currently, nil values for namespace selectors mean *any* namespace is allowed:

|                         | old api client   | new api client |
|-------------------------|------------------|----------------|
| namespaceAsNameSelector | ANY              | AND            |
| namespaceSelector       | AND              | AND            |
| ipBlock                 | NOT ALLOWED      | NOT ALLOWED    |

Thus, adding a *new* namespace selector field would result in  old api clients assuming
that ANY namespace is allowed when namespaceAsNameSelectors are utilized instead of the namespaceSelector fields, due to the fact that *nil namespaceSelectors are promiscuous*.

Possible solutions to this divergent interpretation of policy might be:
- closed failures which break on old clients (making networkpolicy api evolution very hard)
- open failures with lots of warnings and docs about how old clients might overinterpret the allowed connectivity for a policy

Both of these are impractical due to the nature of security tools, which need to be robust and explicit.


Ideally, a fix here would “just work” for “back-rev” implementations of NetworkPolicy (which are out-of-tree - generally implemented by cni plugins) and would not introduce new surprises for users… But in practicality this is not the case because, by adding a new selection data structure, the logical inclusion/exclusion properties of namespace and pod selectors become drastically different, as shown in the table below:



### Goals

- Add an immutable default label to ALL namespaces, as the namespace name so this can be used as selector by arbitrary components.

### Non-Goals

It’s also worth noting that there are at least 2 other NamespaceSelector APIs in k/k - Validating and Mutating webhooks.  They will inevitably need similar API evolution, but we don't  need to do that as part of this KEP.

## Proposal

We propose to label of all namespaces, by default, with a reserved label (i.e. “kubernetes.io/metadata.namespace” or some such), to allow easy selection.
If this label is missing, it is added by the apiserver
If the label is incorrect or changed, the apiserver fails to validate the object


### User Stories (Optional)

See https://github.com/kubernetes/enhancements/pull/2113/files for relevant user stories.

### Notes/Constraints/Caveats (Optional)


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


This can be implemented by modifying the defaults.go and validation.go components for the namespace API, i.e., in the defaults.go file for api/core:

```
func SetDefaults_Namespace(obj *v1.Namespace) {
if obj.Labels == nil {
   		obj.Labels = map[string]string{}
}
if _, found := obj.Labels[“kubernetes.io/metadata.name”]; !found {
   		obj.Labels[“kubernetes.io/metadata.name”] = obj.Name
}
}
```
And in the validation.go, we implement the guarantee of this namespace’s value being constant WRT namespace name (which is immutable).

```
func ValidateNamespace(namespace *core.Namespace) field.ErrorList {
	// ...
   	// Check if namespace.Labels[“kubernetes.io/metadata.name”] == namespace.Name
// if not add an error
   	// ...
return allErrs
```

This solves not just NetworkPolicy, but all such namespace selector APIs without any disruption or version skew problems.

### Test Plan

### Graduation Criteria


### Upgrade / Downgrade Strategy


### Version Skew Strategy


## Production Readiness Review Questionnaire


### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Also set `disable-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?).

* **What happens if we reenable the feature if it was previously rolled back?**

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling or disabling feature
  gates. However, unit tests in each component dealing with managing data, created
  with and without the feature, are necessary. At the very least, think about
  conversion tests if API types are being modified.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  Think about both cluster-level services (e.g. metrics-server) as well
  as node-level agents (e.g. specific version of CRI). Focus on external or
  optional services that are needed. For example, if this feature depends on
  a cloud provider API, or upon an external software-defined storage or network
  control plane.

  For each of these, fill in the following—thinking about running existing user workloads
  and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
  For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History
 

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives



We considered doing this for all resources, but some k8s resources allow object names that are not valid label values.  Given that we have a clear and concrete use-case in namespaces, it seemed proper to start here.  If other kinds need similar treatment (which seems unlikely) they can use the same label key if their name format allows.

We considered telling users to DIY with Validating and Mutating admission controllers.  This is unpleasant at best and seems particularly egregious given that a) multiple APIs benefit from this; and b) the implementation is very simple.


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

# https://github.com/kubernetes/enhancements/issues/2161
