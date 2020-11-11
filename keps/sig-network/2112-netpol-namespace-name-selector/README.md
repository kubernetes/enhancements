<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Generic motivation](#generic-motivation)
  - [Community feedback on the namespace.Name selector](#community-feedback-on-the-namespacename-selector)
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
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA Graduation](#ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Alternative API implementation change](#alternative-api-implementation-change)
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

The ability to target namespaces using names, when building NetworkPolicy 
objects, has been a request from the broader k8s community for quite some 
time.  In the sig-network networkpolicy API sub project, it was one of the 
most popular policies which was voted on.

## Motivation

NetworkPolicies are under utilized by developers when defining applications, 
and the most obvious tenancy boundary for a developer is that of **the Namespace**. 
We thus aim to make this boundary **extremely** obvious and easy for users 
to target in a maximally secure manner.

NetworkPolicies are also used by administrators to make specific default 
policies (for example, the common `kube-system` namespaces might be one which 
an administrator wants to protect from unwanted traffic).

Although service-mesh and other technologies have been slated to obviate the 
need for developer driven security boundaries, these technologies aren't 
available in most clusters, and aren't supported by the Kubernetes API.

The ability to provide granular and intuitive network boundaries between apps 
is part of the broader vision to make the NetworkPolicy API a universal 
security construct implemented in all production applications.  Many 
conversations have come up around this topic, with [kubernetes #88253](https://github.com/kubernetes/kubernetes/issues/88253) 
being one of the most recent ones.

### Generic motivation

- Embrace the immutable nature of a namespace name as a fundamental security 
construct in the Kubernetes ecosystem.
- Making network policies more secure by making it impossible to "impersonate" 
a bespoke namespace.
- Making network policy tenancy boundaries more declarative to use by *not* 
requiring developers to copy/duplicate namespace labels

These three motivating factors are mostly self explanatory, but in the next 
section we outline concrete feedback in these areas.

### Community motivationg factors

- In 2016, the argument for namespace as name selectors was first made in
the [mailing list](https://groups.google.com/g/kubernetes-sig-network/c/GzSGt-pxBYQ/m/Rbrxve-gGgAJ), based on the fact that labels can be 
retroactively added to namespaces pretty easily, even if your namespace isn't 
intended to be able to send traffic somewhere: 

```
we need to clarify how namespaces are matched. I'm pretty 
uncomfortable with using labels here, since anyone can add labels at 
any time to any namespace and thus send traffic to your pod if they 
know the label you're using. If it were simply a namespace name, it's 
much easier to specifically allow only namespaces you control since you 
know your own namespace names. 
```

- Another recent argument was made on the basis of targeting namespaces 
using names, rather then labels, on the basis of **sheer convenience**.

```
While matching things like pods etc by label is certainly worthwhile, when 
matching a namespace I suspect the majority of the time you only want to match 
a single namespace. It been great match against the name rather than just a 
label. I suspect most people don't think to add labels to the namespaces.
```

The latter comment received 9 likes as a github issue - indiciating the 
general popularity of this as a request, and that correlates well to feedback 
we've seen in the broader community as well.

We thus conclude that, even though targeting an object using its name is not 
normal in K8s, in the case of NetworkPolicies, the overwhelming need for 
universal, easy to define security boundaries, makes a strong case for 
amending the API to support a "special" selector for namespace names that is 
**independent** of labels.

### Goals

- Add a `namespaceNames` option (which is additive to the current `matchLabels` selector).
- Allow multiple `namespaceNames` values, so that many namespaces can be 
allowed using the same field 

### Non-Goals

- Support matching of wildcard namespaces
- Changing the internal details of how namespaceSelector works with arbitrary syntax
- Depending on more sophisticated features (like virtual labels) to accomplish 
this feature without changing the API (note that, if someone were 
to write a virtualLabels KEP, however, it might change the trajectory of this 
KEP, which is worth discussing)

### User Stories (Optional)

See motivation section for user stories.  This KEP collects concrete data and opinions from several individuals over the past few years, hence we do not include generic user stories.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

- NetworkPolicy providers may **opt-out**, initially to support this construct, and dilligent communication with CNI providers will be needed to make sure its widely adopted, thus, this feature needs to be backwards compatible with the existing v1 api.
- We thus need to make sure that hidden defaults don't break the meaning of existing policys, for example:
  - if `namespaceNames` field is retrieved by an api client which doesn't yet support it, the client doesnt crash (and the plugin doesnt crash, either).
  - if `namespaceNames` is nil, the policy should behave identically to any policy made before this field was added.
    - in other words, there is no "deny all" semantic that can be enforced by this namespace being missing.
  - if `namespaceNames` is empty, it has IDENTICAL semantics as if it were nil.
    - in other words, there is no "allow all" namespaces semantic which corresponds to emptyness

Overall, the `NetworkPolicyPeer` modifications should be unobtrusive with regard to how other fields interplay, regardless of its absence or presence, notwithstanding the fact that it potentially broadens the selection of the `namespaceSelector` field.

## Proposal

In NetworkPolicy specification, inside `NetworkPolicyPeer` specify a new `namespaceNames` field.  

1) One simple way to implement this feature is to modify the NetworkPolicyPeer 
object, like so.  Since this doesn't involve modifying a go type (i.e. were 
not replacing a labelSelector with a different type, we expect it 
to be a cleaner implementation:

```
type NetworkPolicyPeer struct {
  // new field
  namespaceNames []string

  // existing fields...
  PodSelector *metav1.LabelSelector `json:"podSelector,omitempty" protobuf:"bytes,1,opt,name=podSelector"`
  NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty" protobuf:"bytes,2,opt,name=namespaceSelector"`
  IPBlock *IPBlock `json:"ipBlock,omitempty" protobuf:"bytes,3,rep,name=ipBlock"`
}
```

## Design Details

- Add a new selector to the network policy peer data structure which can switch between allowing a `namespaceNames`, supporting a policy that is expressed like this:

- A list of conventional namespace names (i.e. no regular expressions or any other fancy definining syntaxes)

The "namespaceNames" allow directive would look like so:
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
      namespaceNames:
      -  my-frontend
      -  my-frontend-2
```

- As a more sophisticated example: The following would:
  - allow all traffic from `podName:xyz` living in `my-frontend` or `my-frontend-2`
  - deny traffic from `podName:xyz` in `my-frontend-3`, since theres no selector for that namespace
  - allow things in ipblock 100.1.2.0/16

This is because the presence of EITHER a namespaceNames rule or a namespaceSelector, makes the podSelector subject to an **AND** filter operation alongside the pod selector.

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
    - namespaceNames:
      -  my-frontend
      namespaceSelector:
        matchLabels:
          app: my-frontend-2
      podSelector:
        matchLabels:
          podName: xyz
    -  ipBlock:
        cidr: 100.1.2.0/16
        except:
        - 100.1.2.3/24
```


### Test Plan

We will add tests for this new api semantic into the exting test/e2e/ network 
policy test suites in upstream which cover both of these scenarios, using the 
validation framework outlined in https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/20200204-NetworkPolicy-verification-rearchitecture.md#motivation.

### Graduation Criteria

All new field introductions need gates for at least one release, thus, we 
would add this feature gate in alpha, so we must use feature gates 
idiomatically to implement this change.  

Gone are the days of changing the API with reckless abandon.

#### Alpha 
- Communicate CNI providers about the new field.
- Add validation tests in API which confirm several positive / negative 
scenarios in the test matrix for when this field is present/absent
- All new field introductions need gates for at least one release, thus, we 
would add this feature gate in alpha. 


#### Beta
- The name selector has been supported for at least 1 minor release.
- Four commonly used NetworkPolicy (or CNI providers) implement the new field, 
with generally positive feedback on its usage.
- Feature Gate is enabled by Default.

#### GA Graduation

- At least **four** NetworkPolicy providers (or CNI providers) support the The 
name selector field.
- The name selector has been enabled by default for at least 1 minor release.

### Upgrade / Downgrade Strategy

If upgraded no impact should happen as this is a new matching option and not 
colliding with old ones.  There will potentially be some golang magic required 
to convert objects at the type level to be flexible enough to support 
different inputs, but this will use the K8s API Translation layer.

If downgraded the CNI wont be able to look into the new field, as this does 
not exists and network policies using this field will stop working.

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
  Nothing. Just need to check if the data is persisted in ``etcd`` after the 
  feature is disabled and reenabled or if the data is missed

* **Are there any tests for feature enablement/disablement?**
 
 TBD
 
### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  By looking at the kubernetes networkpolicys in the cluster

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**

   CNI Provider metrics can be usedto confirm that creation of a new policy 
   targeting a namespace name is working

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  Policy latency is currently not measured because its implemented by CNI providers 

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
 No

## Implementation History

None

## Drawbacks

We dont want to have > 1 way to select namespaces.

## Alternatives

A network policy operator could be created which translated a CRD into many networkpolicys on the fly, by watching namespaces and updating labels dynamically.  This would be a privileged container in a cluster and likely would not gain much adoption.

### Alternative API implementation change

Another (possibly more disruptive ) possible implementation of this would be.  We include this for completeness, but
are proposing the 1st, simpler option for this KEP.  This is interesting because it "folds" into the existing API, thus 
paving the way for symmetrical implementation for podSelectors, and so on.  However, its increased complexity due to the 
fact that it breaks the existing type system by changing the `labels` implementation, makes it less attractive.

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

## Infrastructure Needed (Optional)

A CNI provider that supports network policys
