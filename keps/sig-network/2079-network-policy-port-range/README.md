<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA Graduation](#ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
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

Today the ``ports``field in ingress and egress network policies is an array 
that needs a declaration of each single port to be contemplated. This KEP 
proposes to add a new field that allows a declaration of a port range, 
simplifying the creation of rules with multiple ports.

## Motivation

NetworkPolicy object is a complex object, that allows a developer to specify 
what's the traffic behavior expected of the application and allow/deny 
undesired traffic.

There are a number of user issues like [kubernetes #67526](https://github.com/kubernetes/kubernetes/issues/67526) 
and [kubernetes #93111](https://github.com/kubernetes/kubernetes/issues/93111) 
where users expose the need to create a policy that allow a range of ports but some 
specific port, or also cases that a user wants to create a policy that allows 
the egress to other cluster to the NodePort range (eg 32000-32768) and in this case, 
the rule should be created specifying each port separately, as:

```
spec:
  egress:
  - ports:
    - protocol: TCP
      port: 32000
    - protocol: TCP
      port: 32001
    - protocol: TCP
      port: 32002
    - protocol: TCP
      port: 32003
[...]
    - protocol: TCP
      port: 32768
```

So for the user, actually:
* To allow a range of ports, each of them must be declared as an item from 
``ports`` array
* To make an exception needs a declaration of all ports but the exception

Adding a new ``PortRange`` field inside the ``ports`` will allow a simpler 
creation of NetworkPolicy to the user.

### Goals

Add Range field to Ports in NetworkPolicy

### Non-Goals

Specify exceptions to the range. An exception can at this time be specified 
as multiple ranges not contemplating the exception.

## Proposal

In NetworkPolicy specification, inside ``NetworkPolicyPort`` object struct 
specify a new ``Range`` field composed of the minimum and maximum ports 
inside the range.

If both ``Port`` and ``Range`` are specified, they are cumulative and no further
validation might occur. It's up to the CNI to summarize both of the fields in a 
single rule.

### User Stories (Optional)

#### Story 1

I have an application that communicates with NodePorts of a different cluster 
and I want to allow the egress of the traffic only the NodePort range 
(eg. 30000-32767) as I don't know which port is going to be allocated on the 
other side, but don't want to create a rule for each of them.

### Notes/Constraints/Caveats

*  The technology used by the CNI provider might not support port range in a 
trivial way as described in [#drawbacks]

### Risks and Mitigations

CNIs will need to support the new field in their controllers. For this case 
we'll try to make broader communication with the main CNIs so they can be aware
of the new field.

## Design Details

API changes to NetworkPolicy:
* Add a new struct called ``NetworkPolicyPortRange`` as the following:
```
type NetworkPolicyPortRange struct {
    From uint16
    To   uint16
}
```
* Add a new field ``spec.ingress|egress.ports.range`` that points to the
new struct:
```
// NetworkPolicyPort describes a port to allow traffic on
type NetworkPolicyPort struct {
    Range NetworkPolicyPortRange
}
```

The range will need to be validated, with the following scenarios:
* If there's a ``From`` or a ``To`` field defined, the other one must be defined.
* ``From`` needs to be less than or equal to ``To``


### Test Plan

TBD

### Graduation Criteria

#### Alpha 
- Add a feature gated new field to NetworkPolicy
- Communicate CNI providers about the new field
- Add validation tests in API

#### Beta
- ``PortRanges`` has been supported for at least 1 minor release
- At least one CNI provider implements the new field
- Feature Gate is enabled by Default

#### GA Graduation

- At least two CNIs support the ``PortRanges`` field
- ``PortRanges`` has been enabled by default for at least 1 minor release

### Upgrade / Downgrade Strategy

If upgraded no impact should happen as this is a new field.

If downgraded the CNI wont be able to look into the new field, as this does not 
exists and network policies using this field will stop working.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: NetworkPolicyPortRange
    - Components depending on the feature gate: Kubernetes API Server
  
* **Does enabling the feature change any default behavior?**
  No

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes, but CNIs relying on the new field wont recognize it anymore

* **What happens if we reenable the feature if it was previously rolled back?**
  Nothing. Just need to check if the data is persisted in ``etcd`` after the 
  feature is disabled and reenabled or if the data is missed

* **Are there any tests for feature enablement/disablement?**
 
 TBD

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  
  Operators can determine if NetworkPolicies are making use of Range creating
  an object specifying the range and validating if the traffic is allowed within 
  the specified range

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  Operators would need to monitor the traffic of the Pods to verify if a 
  specified port range is applied and allowed in their workloads

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
 N/A

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
 N/A


### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  No


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  TBD

* **Will enabling / using this feature result in introducing new API types?**
  No, unless the new ``NetworkPolicyPortRange`` is considered a new API type

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
  No

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**

  - API type(s): NetworkPolicy / NetworkPolicyPorts
  - Estimated increase in size: New struct inside the object with two fields of 
  16 bits each
  - Estimated amount of new objects: N/A

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  N/A

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  It might get some increase of resource usage by the CNI while parsing the 
  new field.

### Troubleshooting

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
  As this feature is mainly used by CNI providers, the reaction with API server 
  and/or etcd being unavailable will be the same as before.

* **What are other known failure modes?**
  N/A

* **What steps should be taken if SLOs are not being met to determine the problem?**
  N/A

## Implementation History
- 2020-10-08 Initial [KEP PR](https://github.com/kubernetes/enhancements/pull/2079)

## Drawbacks

*  The technology used by the CNI provider might not support port range in a 
trivial way. As an example, OpenFlow does not have a way to specify port range
and this might be a problem for OVS based CNIs as commented in 
[kubernetes #67526](https://github.com/kubernetes/kubernetes/issues/67526#issuecomment-415170435).
The same way, eBPF based CNIs will need to populate their maps in a different way.
