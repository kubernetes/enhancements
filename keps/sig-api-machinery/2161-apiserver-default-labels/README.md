# KEP-2161 : Immutable label selectors for all namespaces

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Avoid labels misuse](#avoid-labels-misuse)
    - [Lack of permissions](#lack-of-permissions)
    - [Simplicity](#simplicity)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Why we are directly enabling this feature in beta](#why-we-are-directly-enabling-this-feature-in-beta)
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
  - [Add new select-by-name capabilities to policy APIs](#add-new-select-by-name-capabilities-to-policy-apis)
    - [API Machinery modifications and workarounds](#api-machinery-modifications-and-workarounds)
    - [API modifications at the NetworkPolicy level](#api-modifications-at-the-networkpolicy-level)
  - [Improve NetworkPolicy API and deprecate old fields](#improve-networkpolicy-api-and-deprecate-old-fields)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
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

Namespaces are not guaranteed to have any 'identifying' labels by the Kubernetes API.  This means that to select a namespace by name, on would have to use a field selector.  This encumberance makes it hard to write default NetworkPolicies, and also makes its tricky to create other kinds of "label-driven" namespace functionality in the K8s api.

The idea of this KEP is to propose that namespaces are always ensured, by the API server, to have a *immutable label matching the metadata.name of the namespace*, which thus allows one to build label selectors to include/exclude namespaces by name, without having write access to a namespace (which is required to label it manually). 

This was initially envisioned as a workaround to a limitation in the network policy API, but has emerged as a potential to other problems which may be solved by default namespace labels.

## Motivation

In https://github.com/kubernetes/enhancements/issues/2112 , we discussed several 
ways to implement network policies that select namespaces by name, rather than by 
relying on labels which may be unknown, or uneditable by people writing policies, 
or untrustworthy.

This was inspired by a broad community ask to target namespaces in this manner, 
as evidenced by issues such as [kubernetes #88253](https://github.com/kubernetes/kubernetes/issues/88253), 
and the larger set of user feedback obtained in  [user stories](https://github.com/jayunit100/network-policy-subproject/blob/master/p0_user_stories.md).

It’s also worth noting that there are at least 2 other NamespaceSelector APIs in 
k/k - Validating and Mutating webhooks.  They will inevitably need similar API 
evolution, but we don't need to do that as part of this KEP.

It turns out that alternate [solutions](#alternatives):
Break the old API in fundamental ways that are questionable in terms of cost/benefit, 
add nesting complexity to the network policy API, wherein certain “subsets” of
the API were not usable together.  This is not “bad”, but it is suboptimal and
introduces redundant semantics. This is avoidable if we could assume, in all
cases, that namespaces have a default label, thereby leveraging existing
functionalities offered by label selectors.

There may also be many other usecases where the security or permissions boundary
might be conveniently defined in terms of a namepsace name, and that such a namespace
definition would be most easily referenced by a default label.

### Goals

- Add the ability to select namespaces by name reliably using traditional label selector methods.

### Non-Goals

* Publishing name labels on all resources, this is an option we can explore later for a larger subset of fields in the Kubernetes API.
* Exposing arbitrary fields to selectors.

## Proposal

We propose to label all namespaces, by default, with a reserved label of
“kubernetes.io/metadata.name” and set it to the namespace's name. This will allow
any resource with a `namespaceSelector` field (such as NetworkPolicy, MutatingWebhookConfiguration
etc) to select (or exclude with `notIn`) these namespaces by explicitly matching on
the namespace's name with traditional label selector mechanisms.

- If this label is missing, it is added by the apiserver as a default.
- If this label is deleted, it is added by the apiserver as a default.
- defaults.go mutates namespaces on read, to always have the label `kubernetes.io/metadata.name=obj.Name`, this effectively means a mutation would be an allowed no-op, since the apiserver would always overwrite the value of this field.
- api clients always see this default label, because of this mutation.


### User Stories (Optional)

#### Avoid labels misuse

NetworkPolicies allow users to open up traffic to/from their namespaces with the help of `namespaceSelector` field.
However, there is a concern that arbitrary labels may not be the most secure way to open up traffic when it comes
to namespace resource. In case a namespace's labels are known by others, any user with wrote access to a namespace can add labels at any time to said namespace, and thus send traffic to a pod belonging to this namespace.

Matching a namespace by its name, on the other hand, is a more reliable way to whitelist namespaces as it's much easier to specifically allow only namespaces user control since user knows their own namespace names.

#### Lack of permissions

Users want to allow DNS traffic to coreDNS pods which are created in system generated namespaces like kube-system.
Since these system-generated namespaces do not come up with labels, users may not be authorized to add labels to
such namespaces and thereby accessing some essential services is not straight forward.

#### Simplicity

It is being reported that users find it difficult to work with mutation and validation webhooks as they are not
able to select a single namespace to enforce those controllers. Similarly, it is quite common and intuitive for
users to remember the names of the namespaces rather than labels associated with those namespaces while defining
NetworkPolicies or writing admission controller configurations.

Thus, regardless of wether such a user story is valid, it is clearly 'simpler' to implement (as are other similar stories) when a default mechanism for namespace selection is available.

### Notes/Constraints/Caveats (Optional)

* The name of the namespace must satisfy label value [requirements](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set).
Currently, the name of the namespace must be a valid [DNS label](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-label-names), and this requirement will not change going forward.

### Risks and Mitigations

* Every namespace gets bigger in size. We think this is inconsequential
* Some user may already be using that label already. This is mitigated by the 
fact that the whole "kubernetes.io" label prefix is [reserved](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set). However, since there
is no validation of reserved label prefixes, an upgrade would mutate such a label (this is not a major concern).
* any clients/applications expecting this label (i.e. for a NetworkPolicy), wouldnt see it, and thus they might not be able to find namespaces using this label technique.  In the case of network policies, this would result in traffic being blocked due to unavailable namespace labels.

For this reason, we can do one alpha release where we log/event if we observe usage of this key that is not compatible.

## Design Details

This can be implemented by modifying the defaults.go and validation.go components for the namespace API, i.e., in the defaults.go file for api/core.  Note that we will support DISABLING this via a feature gate (named `NamespaceDefaultLabelName`), per the versioning specification in this KEP.

```
		obj.Labels["kubernetes.io/metadata.name"] = obj.Name
```

In validation.go, we'll go on to validate this namespace, like so, again, allowing disablement of the feature per the versioning specification in this KEP.
```
		if namespace.Labels[v1.LabelNamespaceName] != namespace.Name {
			allErrs = append(allErrs, field.Invalid(field.NewPath("metadata", "labels").Key(v1.LabelNamespaceName), namespace.Labels[v1.LabelNamespaceName], fmt.Sprintf("must be %s", namespace.Name)))
		}
```


This solves not just NetworkPolicy, but all such namespace selector APIs without  any disruption or version skew problems.  Note that this feature will be defaulted to on in its Beta phase, since it is expected to be benign in all practical scenarios.  

### Why we are directly enabling this feature in beta

This feature is generally thought of us non-disruptive, but reliance on it will likely be taken up quickly due to its simplicity and its ability to solve a common user story for network security.  Thus, if we have a complex process around disabling it initially, we could confuse applications which rely on this feature - which have no API driven manner of determining the status of the automatic labelling which we are doing.  

Consider the scenario where one uses a `not in` selector to exclude namespaces with specific names.  In this scenario, such a selector would not be able to locate such namespaces, and would thus fail.  There is no clear way a user would be able to reconcile this failure, due to the fact that its an apiserver default.  Thus, its simpler to default to this feature being 'on' for any release after its introduction.

### Test Plan


* Add unit tests that creates a namespace and checks if the namespace contains the label
* Add a test that tries to select this namespace by the label (this should  return also only that namespace)
* Add a e2e test that deletes / mutates this label but confirms that on subsequent read, it still is there.
* Try to modify the reserved label and this should be a no-op- the label is guaranteed to be there.
* Check the WATCH operation against existing namespaces that do not have this label still work properly.

### Graduation Criteria
* This feature being enabled by default at least one release.

### Upgrade / Downgrade Strategy

* Check the WATCH situation described in test plan, otherwise populate the label 
of namespaces without it.
* In case of downgrade, as this is just a label it will work fine. Network Policies 
that rely on this label w.r.t. their `matchLabels` criteria within namespaceSelector
to enforce rules will fail-close, as they no longer will match these namespaces,
thereby fail to add such namespaces to their whitelist rules.
On the other hand, network policies with `matchExpressions` within a
namespaceSelector, specifically with a `notIn` operator as shown below, will
inadvertently select this namespace instead of excluding it, which is an undesired
side effect.

```
namespaceSelector:
  matchExpressions:
  - key: kubernetes.io/metadata.name
    operator: NotIn
    values:
    - kube-system
```

Hence, it must be noted that namespaces are not guaranteed to have this label on
downgrades to release which do not support this feature. Users may choose to label
their namespaces by themselves in order to restore their expected behaviour in
terms of network policy rules.

### Version Skew Strategy

It is assumed that losing data under a `kubernetes.io` label is acceptable, since its a reserved label and should not generally be used by non-k8s core components:

```
The kubernetes.io/ and k8s.io/ prefixes are reserved for Kubernetes core components.
```

which is published in https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/.

Since this has been reserved since 2017 (if not earlier), no backwards guarantees are made for clusters using the `kubernetes.io/metadata.name` label, but we'll do our best to not break people who coincidentally may have used this label (debatable wether this is a realistic concern or not).

- Old versions will be assumed to never have this label.  If they do have this label, then they will eventually lose its contents.
- New versions always will have this label, lazily added on read or write.
- During upgrades, a policy created may be able to "get away" with temporarily writing to the aformentioned "kubernetes.io" metadata.name label, however, this write will eventually be lost.
- It's generally agreed that the kubernetes.io prefix is reserved, and thus, there is no need to warn or carefully migrate its presence into releases. We will thus simply have a feature gate, which can be disabled for one release.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

1.21:
- enable by default
- allow gate to disable the feature
- release note

1.22:
- gate enabled by default
- release note

1.23:
- remove the gate

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Yes, namespaces which are already targeted via this label may break, although such policies would be in violation of the standard that k8s.io labels are reserved.

* **What specific metrics should inform a rollback?**
  No e2e tests or metrics exist to inform this specific action, but if a net loss of healthy pods occurred and a user is actively using networkpolicies that impersonate on the k8s.io label, that would be a hint.
 
  Indirectly, this could be discovered by inspection of APIServer rejection events associated with the namespace API resource.

For example, the apisesrver_request_total metric can be inspected for "non 201" events.  In rejection scenarios, the change in metrics is evident as follows...
```
apiserver_request_total{code="201",component="apiserver",contentType="application/json",dry_run="",group="",resource="namespaces",scope="resource",subresource="",verb="POST",version="v1"} 9
apiserver_request_total{code="422",component="apiserver",contentType="application/json",dry_run="",group="",resource="namespaces",scope="resource",subresource="",verb="POST",version="v1"} 3
```

Thus, one can monitor for `code != 201` increases, and thereby infer a potential issue around namespace admission to the APIServer, and investigate/rollback as needed.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Not yet. TBD.
  
* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
  Any user relying on namespaces with **no** labels, or expecting **some** namespaces to not have labels including the `k8s.io`prefix defined here, would observe a potential changes in namespace selection behaviour with this, however, that is not a guaranteed API feature in any way, so no actual feature is being deprecated. 
  
### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**

By running `kubectl get ns --show-labels` and inspecting the `kubernetes.io` values.

* **What are the SLIs (Service Level Indicators) an operator can use to determine** 

Since this just an immutable field, we don't expect any such indicators to be of relevance.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

Since this just an immutable field, we don't expect any such indicators to be of relevance.

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**

Since this just an immutable field, we don't expect any such indicators to be of relevance.


### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  None

### Scalability

* **Will enabling / using this feature result in any new API calls?**

No, because it is just mutating the incoming created namespaces at the APIServer level.

* **Will enabling / using this feature result in introducing new API types?**

No.

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**:

No

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**

Yes. Each namespace object will marginally increase in size to accommodate an extra label.

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**

No

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**

No

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
There are no catastrophic failure scenarios of note, but there might be some user facing scenarios which are theoretically worth mentioning here.
- If this label already existed, a user will get unforeseen consequences from it.
- If administrators don't allow any labels on namespaces, or have special policies about labels that are allowed, this introduction could violate said poliices.

* **What steps should be taken if SLOs are not being met to determine the
  problem?** 

No SLOs are proposed.

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History
* 2020-11-22 - Creation of the issue and the KEP 
* 2020-11-30 - https://github.com/kubernetes/kubernetes/pull/96968/files example PR implementing this

## Drawbacks

* "It's kind of a special hack. The fact that it isn't generalized or generalizable 
makes it feel special." - @thockin

## Alternatives

### Add new select-by-name capabilities to policy APIs

#### API Machinery modifications and workarounds

* Add an 'in memory' virtual label that is defaulted by the apiserver but not stored in etcd, this has the drawback of diverging the apiserver from data in an etcd backup, making inspection and recovery of offline clusters more complicated.
* Add a new syntax for selectors on labels that allow them to dynamically select things in fields,  This has the drawback of being prohibitively complicated from a perspective of community convergence on what the boundaries of such a syntax might actually be.

#### API modifications at the NetworkPolicy level

* The original proposal was to create a new kind of selector for NetworkPolicies, this of course has many drawbacks, including most of all, the
need for new API semantics in the networkpolicy specification, which is already tricky to understand for newcomers.

For example, you may need a way to declare namespaceAsName selectors, which is distinct from namespace label selectors.
Introducing a new field has its own drawbacks: The obvious API, when combined with a podSelector, decreases security
of a pod - an old client would be more open than users would expect them to be (in the most obvious implementations).  We note that there are workarounds involving special "always off" selectors, which can avoid this scenario, but they come with their own obvious inherent technical debt and costs.

This is thoroughly debated in the [KEP](https://github.com/kubernetes/enhancements/issues/2112).  Fail-closed is a requirement.

If a podSelector is enabled for a network policy peer, then currently, nil
values for namespace selectors mean *any* namespace is allowed:

|                         | old api client | new api client |
|-------------------------|----------------|----------------|
| namespaceAsNameSelector | ANY            | AND            |
| namespaceSelector       | AND            | AND            |
| ipBlock                 | NOT ALLOWED    | NOT ALLOWED    |

Thus, adding a *new* namespace selector field would result in  old api clients
assuming that ANY namespace is allowed when namespaceAsNameSelectors are utilized
instead of the namespaceSelector fields, due to the fact that *nil
namespaceSelectors are promiscuous*.

Possible solutions to this divergent interpretation of policy might be:
- closed failures which break on old clients (making networkpolicy api evolution very hard)
- open failures with lots of warnings and docs about how old clients might
overinterpret the allowed connectivity for a policy

Both of these are impractical due to the nature of security tools, which need to
be robust and explicit.  Although failing in a closed manner is acceptable, it is
obviously unfortunate, as it would obsolete all network policy providers *not*
eagerly adopting such a new semantic, which is overly restrictive and against
the spirit of community-friendly, safe API evolution which is so important to
a large project like Kubernetes with many external functional plugins.

Ideally, a fix here would “just work” for “back-rev” implementations of NetworkPolicy
(which are out-of-tree - generally implemented by cni plugins) and would not
introduce new surprises for users. But in practicality this is not the case
because, by adding a new selection data structure, the logical inclusion/exclusion
properties of namespace and pod selectors become drastically different, as shown
in the table below:

|         input policy         | policy-reader (old)    | client (new)        |   |   |
|------------------------------|------------------------|---------------------|---|---|
| nsNameSelector + podSelector | ANY ns, pod restricted | ns + pod restricted |   |   |
| nsNameSelector               | invalid (empty peer)   | ns restricted       |   |   |
| podSelector                  | ANY namespace          | ANY namespace       |   |   |

In this table "policy-reader" might be something like the "calico-controller"
or any other cni agent that reads the API Servers network policy portfolio and
responds to it over time.

In an older client (one which predated the addition of a new, "nsNameSelector"
field, we find that there is a scenario (`ANY ns, pod restricted`), wherein an
old interpretation of the API assumes that the ONLY restricting field is a pod
label.  Thus, although the user intended to restrict all traffic to a
namespace/pod combination, the policy implementer does something **much** more
promiscuous.

### Improve NetworkPolicy API and deprecate old fields

Another potential solution is to reorganize the NetworkPolicy API and make it more extensible.
As discussed in the above alternate proposal, adding new fields to the NetworkPolicy API breaks
assumptions and fail open in certain scenarios. This can be avoided by evolving the NetworkPolicy
API to accommodate new fields by updating semantics of the existing fields and/or by deprecating
older fields, replaced by newer more extensible fields.

The above can be released as a new v2 API for NetworkPolicy. This new NetworkPolicy v2 API must
then be implemented by network plugin vendors. The network plugin vendors are encouraged to
log and warn users of any usage of deprecated fields. A more extensible v2 NetworkPolicy API
can then be extended to include fields which would allow selection of namespaces by name. This
whole effort would at least take a year to materialize, subject to network plugin vendors
catching up on the v2 API. In addition to this, it also does not solve the issue for other
resources (eg. MutatingWebhookConfiguration) to select namespaces by name.
