<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [X] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [X] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [X] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [X] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [X] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [X] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-xxxx: Add Namespace as Name field to NetworkPolicy

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
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA Graduation](#ga-graduation)
    - [Removing a Deprecated Flag](#removing-a-deprecated-flag)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)

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

The ability to target namespaces using names, when building NetworkPolicy objects, has been a request from the broader k8s community for quite some time.  In the sig-network-networkpolicy API group, it was one of the most popular policies which was voted on.

## Motivation

NetworkPolicies are under utilized by developers when defining applications, and the most obvious tenancy boundary for a developer is that of **the Namespace**.  We thus aim to make this boundary **extremely** obvious and easy for users to target in a maximally secure manner.

Although service-mesh and other technologies have been slated to obviate the need for developer driven security boundaries, these technologies aren't available in most clusters, and aren't supported by the Kubernetes API.

The ability to provide granular and intuitive network boundaries between apps is part of the braoder vision to make the NetworkPolicy API a universal security construct implemented in all production applications. 

### Generic motivation

- Embrace the immutable nature of a namespace name as a fundamental security construct in the Kubernetes ecosystem.
- Making network policies more secure by making it impossible to "impersonate" a bespoke namespace.
- Making network policy tenancy boundaries more declarative to use by *not* requiring developers to copy/duplicate namespace labels

These two motivating factors are mostly self explanatory, but in the next section we outline concrete feedback in these areas.

### Community feedback on the namespace.Name selector

- In 2016, the argument for namespace as name selectors was first made... https://groups.google.com/g/kubernetes-sig-network/c/GzSGt-pxBYQ/m/Rbrxve-gGgAJ, based on the fact that labels can be retroactively added to namespaces pretty easily, even if your namespace isn't intended to be able to send traffic somewhere: 

```
 we need to clarify how namespaces are matched. I'm pretty 
uncomfortable with using labels here, since anyone can add labels at 
any time to any namespace and thus send traffic to your pod if they 
know the label you're using. If it were simply a namespace name, it's 
much easier to specifically allow only namespaces you control since you 
know your own namespace names. 
```

- Another recent argument was made on the basis of targetting namespaces using names, rather then labels, on the basis of **sheer convenience**.

```
While matching things like pods etc by label is certainly worthwhile, when matching a namespace I suspect the majority of the time you only want to match a single namespace. It been great match against the name rather than just a label. I suspect most people don't think to add labels to the namespaces.
```

The latter comment received 9 likes as a github issue - indiciating the general popularity of this as a request, and that correlates well to feedback we've seen in the broader community as well.

We thus conclude that, even though targetting an object using its name is not normal in K8s, in the case of NetworkPolicies, the overwhelming need for universal, easy to define security boundaries, makes a strong case for amending the API to support a "special" selector for namespace names that is **independent** of labels.

### Goals

- Add a `matchName` option (which is additive to the current `matchLabels` selector).
- Add an additional semantic mechanism to "allow all" namespaces while excluding a finite, specific, list of namespaces (canonically, this might be `kube-system`, as this is an obvious security boundary which most clusters would prefer).

### Non-Goals

- Support matching of wildcard namespaces

## Proposal

In NetworkPolicy specification, inside `namespaceSelector` specify a new `Name` field.  One possible implementation of this would be:

```
    type NamespaceSelector struct {
      names []string
      labels *metav1.LabelSelector
    }
```

which is referenced from the namespaceSelector:

```
    + NamespaceSelector *NamespaceSelector
    - NamespacesSelector *metav1.LabelSelector
```

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As an administrator I want to block all users from accessing the kube-system namespace as a fundamental default policy.

#### Story 2

As an user I want to "just add" a namespace to my allow list without having to manage the labels which might get added/removed over time from said namespace.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

- CNIs may not choose, initially, to support this construct, and diligent communication with CNI providers will be needed to make sure it's widely adopted.
- Fractionating the Kubernetes API idiom by introducing a separate way to select objects.  We accept this cost because security is a fundamentally important paradigm that justifies breaking other paradigms, at times. 

## Design Details

- Add a new selector to the network policy peer data structure which can switch between allowing a "matchName" or "matchLabels" implementation, supporting a policy that is expressed like this:

This selector has two, EXCLUSIVE, possible input types:

- A list of conventional namespaces OR 
- A `*` with an `exclude` rule as in input to this selector

The "conventional namespaces" will look like so:
```
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: mysql-allow-app
spec:
  podSelector:
    matchLabels:
      app: mysql
  ingress:
  - from:
    - namespaceSelector:
         matchName:
         -  my-frontend
```

Meanwhile, the "exclude" implementation, will look like this:

```
  ingress:
  - from:
    - namespaceSelector:
         matchName:
         -  *
         exclude:
         - kube-system
```


### Test Plan

We will add tests for this new api semantic into the exting test/e2e/ network policy test suites in upstream which cover both of these scenarios, using the validation framework outlined in https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/20200204-NetworkPolicy-verification-rearchitecture.md#motivation 

### Graduation Criteria

#### Alpha 
- Add a feature gated new field to NetworkPolicy
- Communicate CNI providers about the new field
- Add validation tests in API

#### Beta
- The name selector has been supported for at least 1 minor release
- At least one CNI provider implements the new field
- Feature Gate is enabled by Default

#### GA Graduation

- At least two CNIs support the The name selector field
- The name selector has been enabled by default for at least 1 minor release

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

### Upgrade / Downgrade Strategy

If upgraded no impact should happen as this is a new matching option and not colliding with old ones.  There will potentially be some golang magic required to convert objects at the type level to be flexible enough to support different inputs, but this will use the K8s API Translation layer.

If downgraded the CNI wont be able to look into the new field, as this does not exists and network policies using this field will stop working.

### Version Skew Strategy


## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: NetworkPolicyNamespaceAsName
    - Components depending on the feature gate: Kubernetes API Server
  
* **Does enabling the feature change any default behavior?**
  No

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes, but CNIs relying on the new field wont recognize it anymore

* **What happens if we reenable the feature if it was previously rolled back?**
  Nothing. Just need to check if the data is persisted in ``etcd`` after the feature is disabled and reenabled or if the data is missed

* **Are there any tests for feature enablement/disablement?**
 
 TBD
 
### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  By looking at the kubernetes networkpolicys in the cluster

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**

   CNI Provider metrics can be usedto confirm that creation of a new policy targetting a namespace name is working

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  Policy latency is currently not measured because its implemented by CNI providers 

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

## Implementation History

None

## Drawbacks

We dont want to have > 1 way to select namespaces.

## Alternatives

A network policy operator could be created which translated a CRD into many networkpolicys on the fly, by watching namespaces and updating labels dynamically.  This would be a privileged container in a cluster and likely would not gain much adoption.

## Infrastructure Needed (Optional)

A CNI provider that supports network policys
