# KEP-2161 : Immutable label selectors for all namespaces

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Today to select a Namespace by its name is not possible because the NamespaceSelector is only
possible to deal with labels.

The idea of this KEP is to propose that namespaces have an immutable label when created, created
by the APIServer that represents the metadata.name of the object and turn namespaces selectable 
by the name.

## Motivation

In https://github.com/kubernetes/enhancements/issues/2112 , we discussed several 
ways to implement network policies that select namespaces by name, rather than by 
relying on labels which may be unknown, or uneditable by people writing policies, 
or untrustworthy.

This was inspired by a broad community ask to target namespaces in this manner, 
as evidenced by issues such as [kubernetes #88253](https://github.com/kubernetes/kubernetes/issues/88253), 
and the larger set of user feedback obtained in  [user stories](https://github.com/jayunit100/network-policy-subproject/blob/master/p0_user_stories.md).

It turns out all solutions:
Break the old API in fundamental ways that are questionably WRT cost/benefit OR
Add nesting complexity to the network policy API, wherein certain “subsets” of 
the API were not usable together.  This is not “bad”, but it is suboptimal and 
introduces redundant semantics.  For example, you may need a way to declare 
namespaceAsName selectors, which is distinct from namespace label selectors.

It’s also worth noting that there are at least 2 other NamespaceSelector APIs in 
k/k - Validating and Mutating webhooks.  They will inevitably need similar API 
evolution, but we don't need to do that as part of this KEP.

Example:  The obvious API, when combined with a podSelector, decreases security 
of a pod - an old client would be more open than users would expect them to be. 
This is thoroughly debated in the KEP.  Fail-closed is a requirement.

If a podSelector is an enabled for a network policy peer, then currently, nil 
values for namespace selectors mean *any* namespace is allowed:

|                         | old api client   | new api client |
|-------------------------|------------------|----------------|
| namespaceAsNameSelector | ANY              | AND            |
| namespaceSelector       | AND              | AND            |
| ipBlock                 | NOT ALLOWED      | NOT ALLOWED    |

Thus, adding a *new* namespace selector field would result in  old api clients 
assuming that ANY namespace is allowed when namespaceAsNameSelectors are utilized 
instead of the namespaceSelector fields, due to the fact that *nil namespaceSelectors are promiscuous*.

Possible solutions to this divergent interpretation of policy might be:
- closed failures which break on old clients (making networkpolicy api evolution very hard)
- open failures with lots of warnings and docs about how old clients might 
overinterpret the allowed connectivity for a policy

Both of these are impractical due to the nature of security tools, which need to 
be robust and explicit.  Although failing in a closed manner is acceptable... it is obviously unfortunate,
as it would obsolete all network policy providers *not* eagerly adopting such a new semantic, which is
overly restrictive and against the spirit of community-friendly, safe API evolution which is so important
to a large project like Kubernetes with many external functional plugins.

Ideally, a fix here would “just work” for “back-rev” implementations of NetworkPolicy 
(which are out-of-tree - generally implemented by cni plugins) and would not 
introduce new surprises for users. But in practicality this is not the case 
because, by adding a new selection data structure, the logical inclusion/exclusion 
properties of namespace and pod selectors become drastically different, as shown in the table below:


In this table "policy-reader" might be something like the "calico-controller" or any other cni agent
that reads the API Servers network policy portfolio and responds to it over time.

|         input policy         | policy-reader (old)    | client (new)        |   |   |
|------------------------------|------------------------|---------------------|---|---|
| nsNameSelector + podSelector | ANY ns, pod restricted | ns + pod restricted |   |   |
| nsNameSelector               | invalid (empty peer)   | ns restricted       |   |   |
| podSelector                  | ANY namespace          | ANY namespace       |   |   |

In an older client (one which predated the addition of a new, "nsNameSelector" field, we find
that there is a scenario (`ANY ns, pod restricted`), wherein an old interpretation of the API
assumes that the ONLY restricting field is a pod label.  Thus, although the user intended to
restrict all traffic to a namespace/pod combination, the policy implementer does something **much**
more promiscuous.

This is all avoidable if we could assume, in all cases, that namespaces have a default label.

There may also be many other usecases where the a security or permissions boundary might be conveniently defined in terms
of a namepsace name, and that such a definition would be most easily referenced by a default label.

### Goals

- Add an immutable default label to ALL namespaces, as the namespace name so this 
can be used as selector by arbitrary components.

### Non-Goals

* Publishing name labels on all resources
* Exposing arbitrary fields as selectors

## Proposal

We propose to label of all namespaces, by default, with a reserved label 
(i.e. “kubernetes.io/metadata.name” or some such), to allow easy selection.

If this label is missing, it is added by the apiserver

If the label is incorrect or changed, the apiserver fails to validate the object


### User Stories (Optional)

See https://github.com/kubernetes/enhancements/pull/2113/files for relevant user stories.

### Notes/Constraints/Caveats (Optional)


### Risks and Mitigations

* Every namespace gets bigger in size. We think this is inconsequential
* Some user may already be using that label already. This is mitigated by the 
fact that the whole "kubernetes.io" namespace is reserved. If this is a 
sticking point, we can do one release where we log/event if we observe 
usage of this key that is not compatible.

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
And in the validation.go, we implement the guarantee of this namespace’s value 
being constant WRT namespace name (which is immutable).

```
func ValidateNamespace(namespace *core.Namespace) field.ErrorList {
	// Check if namespace.Labels[“kubernetes.io/metadata.name”] == namespace.Name
  // if not add an error
  // ...
  return allErrs
```

This solves not just NetworkPolicy, but all such namespace selector APIs without 
any disruption or version skew problems.

### Test Plan
* Add unit tests that creates a namespace and checks if the namespace contains 
the label
* Add a test that tries to select this namespace by the label (this should 
return also only that namespace)
* Try to modify the reserved label and this should return an error
* Check the WATCH operation against existing namespaces that do not have this 
label still work properly.

### Graduation Criteria
* This feature being enabled by default at least one release.

### Upgrade / Downgrade Strategy
* Check the WATCH situation described in test plan, otherwise populate the label 
of namespaces without it.
* In case of downgrade, as this is just a label it will work fine. Network Policies 
that relies on this label may fail-close, which might be acceptable.

### Version Skew Strategy
N/A

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

1.21:
- gate disabled by default
- if gate disabled, log or event incompatible uses of this
- if gate enabled, enforce
- release note

1.22:
- gate enabled by default
- release note

1.23:
- lock gate enabled

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  TBD

* **What specific metrics should inform a rollback?**
  TBD

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  TBD

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

* **Does this feature depend on any specific services running in the cluster?**
  None

### Scalability

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
  
* **Will enabling / using this feature result in any new calls to the cloud 
provider?**: No

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  Describe them, providing:
  - API type(s): Namespace
  - Estimated increase in size: New label with the same size of the namespace name
  - Estimated amount of new objects: N/A

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
* 2020-11-22 - Creation of the issue and the KEP 

## Drawbacks

* "It's kind of a special hack. The fact that it isn't generalized or generalizable 
makes it feel special." - @thockin

## Alternatives

* The original proposal was to create a new kind of selector for NetworkPolicies
