# KEP-2079: Network Policy to support Port Ranges

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1 - Opening communication to NodePorts of other cluster](#story-1---opening-communication-to-nodeports-of-other-cluster)
    - [Story 2 - Blocking the egress for not allowed insecure ports](#story-2---blocking-the-egress-for-not-allowed-insecure-ports)
    - [Story 3 - Containerized Passive FTP Server](#story-3---containerized-passive-ftp-server)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Validations](#validations)
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


## Summary

Today the `ports` field in ingress and egress network policies is an array 
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

So for the user:
* To allow a range of ports, each of them must be declared as an item from 
`ports` array
* To make an exception needs a declaration of all ports but the exception

Adding a new `endPort` field inside the `ports` will allow a simpler 
creation of NetworkPolicy to the user.

### Goals

Add an endPort field in `NetworkPolicyPort`

### Non-Goals

* Support specific `Exception` field.
* Support `endPort` when the starting `port` is a named port.

## Proposal

In NetworkPolicy specification, inside `NetworkPolicyPort` specify a new 
`endPort` field composed of a numbered port that defines if this is a range
and when it ends.

### User Stories

#### Story 1 - Opening communication to NodePorts of other cluster

I have an application that communicates with NodePorts of a different cluster 
and I want to allow the egress of the traffic only the NodePort range 
(eg. 30000-32767) as I don't know which port is going to be allocated on the 
other side, but don't want to create a rule for each of them.

#### Story 2 - Blocking the egress for not allowed insecure ports
As a developer, I need to create an application that scrapes informations from 
multiple sources, being those sources databases running in random ports, web 
applications and other sources. But the security policy of my company asks me 
to block communication with well known ports, like 111 and 445, so I need to create
a network policy that allows me to communicate with any port except those two and so
I can be compliant with the company's policy.

#### Story 3 - Containerized Passive FTP Server
As a Kubernetes User, I've received a demand from my boss to run our FTP server in an 
existing Kubernetes cluster, to support some of my legacy applications. 
This FTP Server must be acessible from inside the cluster and outside the cluster, 
but I still need to keep the basic security policies from my company, that demands 
the existence of a default deny rule for all workloads and allowing only specific ports.

Because this FTP Server runs in PASV mode, I need to open the Network Policy to ports 21 
and also to the range 49152-65535 without allowing any other ports.


### Notes/Constraints/Caveats

*  The technology used by the CNI provider might not support port range in a 
trivial way as described in [#drawbacks]

### Risks and Mitigations

CNIs will need to support the new field in their controllers. For this case 
we'll try to make broader communication with the main CNIs so they can be aware
of the new field.

## Design Details

API changes to NetworkPolicy:
* Add a new field called `EndPort` inside `NetworkPolicyPort` as the following:
```
// NetworkPolicyPort describes a port to allow traffic on
type NetworkPolicyPort struct {
	// The protocol (TCP, UDP, or SCTP) which traffic must match. If not specified, this
	// field defaults to TCP.
	// +optional
	Protocol   *v1.Protocol `json:"protocol,omitempty" protobuf:"bytes,1,opt,name=protocol,casttype=k8s.io/api/core/v1.Protocol"`

	// The port on the given protocol. This can either be a numerical or named 
  // port on a pod. If this field is not provided, this matches all port names and
  // numbers, whether an endPort is defined or not.
	// +optional
	Port       *intstr.IntOrString `json:"port,omitempty" protobuf:"bytes,2,opt,name=port"`

	// EndPort defines the last port included in the port range.
	// Example:
  //    endPort: 12345
	// +optional
	EndPort    int32 `json:"port,omitempty" protobuf:"bytes,2,opt,name=endPort"`
}
```

### Validations
The `NetworkPolicyPort` will need to be validated, with the following scenarios:
* If an `EndPort` is specified a `Port` must also be specified
* If `Port` is a string (named port) `EndPort` cannot be specified
* `EndPort` must be equal or bigger than `Port`

### Test Plan

Unit tests:
* test API validation logic
* test API strategy to ensure disabled fields

E2E tests:
* Add e2e tests exercising only the API operations for port ranges. Data-path 
validation should be done by CNIs.


### Graduation Criteria

#### Alpha 
- Add a feature gated new field to NetworkPolicy
- Communicate CNI providers about the new field
- Add validation tests in API

#### Beta
- `EndPort` has been supported for at least 1 minor release
- Four commonly used NetworkPolicy (or CNI providers) implement the new field, 
with generally positive feedback on its usage.
- Feature Gate is enabled by Default.

#### GA Graduation

- At least **four** NetworkPolicy providers (or CNI providers) support the `EndPort` field
- `EndPort` has been enabled by default for at least 1 minor release

### Upgrade / Downgrade Strategy

If upgraded no impact should happen as this is a new field.

If downgraded the CNI wont be able to look into the new field, as this does not 
exists and network policies using this field will stop working correctly and
start working incorrectly. This is a fail-closed failure, so it is acceptable.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: NetworkPolicyEndPort
    - Components depending on the feature gate: Kubernetes API Server
  
* **Does enabling the feature change any default behavior?**
  No

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes, but CNIs relying on the new field wont recognize it anymore

* **What happens if we reenable the feature if it was previously rolled back?**
  Nothing. Just need to check if the data is persisted in `etcd` after the 
  feature is disabled and reenabled or if the data is missed

* **Are there any tests for feature enablement/disablement?**
 
 TBD

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  
  Operators can determine if NetworkPolicies are making use of EndPort creating
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
  No, unless the new `EndPort` is considered a new API type

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
  No

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**

  - API type(s): NetworkPolicyPorts
  - Estimated increase in size: 2 bytes for each new `EndPort` specified
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
trivial way. As an example, OpenFlow did not supported to specify port range
for a while as commented in [kubernetes #67526](https://github.com/kubernetes/kubernetes/issues/67526#issuecomment-415170435). 
While this has changed in Open vSwitch v1.6, this still might be a caveat
for other CNIs, like eBPF based CNIs will need to populate their maps in a 
different way.

For this cases, CNIs will have to iteract through the Port Range and
populate their packet filtering tables with each port.

## Alternatives

During the development of this KEP there was an alternative implementation 
of the `NetworkPolicyPortRange` field inside the `NetworkPolicyPort` as the following:

```
// NetworkPolicyPort describes a port or a range of ports to allow traffic on
type NetworkPolicyPort struct {
  // The protocol (TCP, UDP, or SCTP) which traffic must match. If not specified, this
  // field defaults to TCP.
  // +optional
  Protocol *api.Protocol
  
  // The port on the given protocol. This can either be a numerical or named 
  // port on a pod. If this field is not provided but a Range is 
  // provided, this field is ignored. Otherwise this matches all port names and
  // numbers.
  // +optional
  Port *intstr.IntOrString

  // A range of ports on a given protocol and the exceptions. If this field 
  // is not provided, this doesn't matches anything
  // +optional
  Range *NetworkPolicyPortRange
}
```

But the main design suggested in this Kep seems more clear, so this alternative
has been discarded. 

Also it has been proposed that the implementation contains an `Except` array and a new 
struct to be used in Ingress/Egress rules, but because it would bring much more complexity 
than desired the proposal has been dropped right now:

```
// NetworkPolicyPortRange describes the range of ports to be used in a 
// NetworkPolicyPort struct
type NetworkPolicyPortRange struct {
    // From defines the start of the port range
    From         uint16
    
    // To defines the end of the port range, being the end included within the
    // range
    To           uint16
    // Except defines all the exceptions in the port range
    +optional
    Except       []uint16
``` 
