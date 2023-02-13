# KEP-3766: Move ReferenceGrant to SIG Auth API Group

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [No Default Implementation](#no-default-implementation)
    - [Potential for Variations Among Implementations](#potential-for-variations-among-implementations)
    - [Cross-Namespace References may Weaken Namespace Boundaries](#cross-namespace-references-may-weaken-namespace-boundaries)
- [Design Details](#design-details)
  - [General Notes](#general-notes)
    - [<code>ReferenceGrant</code> is half of a handshake](#-is-half-of-a-handshake)
    - [ReferenceGrant authors must have sufficient access](#referencegrant-authors-must-have-sufficient-access)
    - [<code>Resource</code> vs <code>Kind</code>](#-vs-)
    - [Revocation behavior](#revocation-behavior)
  - [Example Usage](#example-usage)
    - [Gateway API Gateway Referencing Secret](#gateway-api-gateway-referencing-secret)
    - [Gateway API HTTPRoute Referencing Service](#gateway-api-httproute-referencing-service)
    - [PersistentVolumeClaim using cross namespace data source](#persistentvolumeclaim-using-cross-namespace-data-source)
  - [API Spec](#api-spec)
  - [Outstanding questions and clarifications](#outstanding-questions-and-clarifications)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Open Questions](#open-questions)
  - [Do We Need Verbs?](#do-we-need-verbs)
  - [Do We Need Any or Wildcard Selectors?](#do-we-need-any-or-wildcard-selectors)
  - [Do We Need Label Selectors?](#do-we-need-label-selectors)
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
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

[ReferenceGrant](https://gateway-api.sigs.k8s.io/api-types/referencegrant/) was
developed by the [Gateway API subproject](https://gateway-api.sigs.k8s.io/) to
enable certain object references to cross namespaces. More recently, it has also
been [used by
sig-storage](https://kubernetes.io/blog/2023/01/02/cross-namespace-data-sources-alpha/)
to enable cross-namespace data sources.

This KEP proposes moving ReferenceGrant from its current
`gateway.networking.k8s.io` API group into the `authorization.k8s.io` API
group.

## Motivation

Any project that wants to enable cross-namespace references currently has to choose
between introducing a dependency on Gateway API's ReferenceGrant or creating a
new API that would be partially redundant (leading to confusion for users).
	
Recent interest between SIGs has made it clear that ReferenceGrant is wanted for use
cases other than Gateway API. We would like to move ReferenceGrant to a neutral home
(ideally, under SIG Auth) in order to make it the canonical API for managing references
across namespaces.

### Goals

* Move ReferenceGrant to an API Group that SIG Auth manages
* Clearly define how ReferenceGrant should be used, including both current use
  cases and guidance for future use cases
* Implement a library to ensure that ReferenceGrant is implemented consistently
  by all controllers

### Non-Goals

* Develop an authorizer that will automatically implement ReferenceGrant for all
  use cases. (It would be impossible to represent concepts like "all namespaces"
  or label selectors that have become important for this KEP).

## Proposal

Move the existing ReferenceGrant resource into a new
`authorization.k8s.io` API group, defined within the Kubernetes code base
as part of the 1.27 release.

We will take this opportunity to clarify and update the API after SIG Auth
feedback. This resource will start with v1alpha1 as the API version.


### Risks and Mitigations

#### No Default Implementation
Similar to the Ingress and Gateway APIs, this API will be dependent on
implementations by controllers that are not included by default in Kubernetes.
This could lead to confusion for users. We'll need to rely heavily on
documentation for this feature, tracking all uses of official Kubernetes APIs
that support ReferenceGrant in a central place.

#### Potential for Variations Among Implementations
Because this relies on each individual controller to implement the logic,
it is possible that implementations may become inconsistent. To avoid that,
we'll provide a standard library for implementing ReferenceGrant. We'll
also strongly recommend that every API that relies on ReferenceGrant 
includes robust conformance tests covering this functionality. Existing
Gateway API conformance tests can serve as a model for this.

#### Cross-Namespace References may Weaken Namespace Boundaries
Although we believe that the handshake required for cross-namespace references
with ReferenceGrant ensures these references will be safe, it does potentially
weaken existing namespace boundaries. We believe ReferenceGrant will have a net
benefit on the ecosystem as it will allow workloads, secrets, and configuration
to be deployed in separate namespaces that more clearly match up with desired
authorization.

## Design Details

### General Notes

#### `ReferenceGrant` is half of a handshake

When thinking about ReferenceGrant, it is important to remember that it does not
do anything by itself. It *Grants* the *possibility* of making a *Reference*
across namespaces. It's intended that _another object_ (that is, the From object)
complete the handshake by creating a reference to the referent object (the To
object).

#### ReferenceGrant authors must have sufficient access

Anyone creating or updating a ReferenceGrant MUST have read access to the resources
they are providing access to. If that authorization check fails, the update or
create action will also fail.

<<[UNRESOLVED Do we need checks beyond read access? ]>>
Previous Discussion: https://github.com/kubernetes/enhancements/pull/3767#discussion_r1084528657

Comment that started that thread:

does anything ensure the user creating the ReferenceGrant has permissions (read?
write?) on the object they are granting access to? Translating the existing
ReferenceGrant into an authz check means translating from Kind to Resource

since this is extending "half of a handshake", it seems important to ensure the
actor extending the handshake actually has permission to do so

<<[/UNRESOLVED]>>

#### `Resource` vs `Kind`

When creating a metaresource (that is, a resource that targets other resources)
like ReferenceGrant, it's important to consider if the metaresource uses the more
common `Kind` or the more correct `Resource`.

In the original Gateway API implementation, we chose to use `Kind` rather than
`Resource`, mainly to improve the user experience. That is, it's easier users
to take the value from the `kind` field at the top of the YAML they are already
using, and put it straight into these fields, rather than needing to do a
kind-resource lookup for every user's interaction with the API. @robscott even
ended up making https://github.com/kubernetes/community/pull/5973 to clarify
the API conventions.

However, in discussion on this KEP, it's clear that the more generic nature of
_this_ API requires the additional specificity that `Resource` provides.

The Gateway API ReferenceGrant looked like this:
```yaml
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  name: allow-gateways
  namespace: bar
spec:
  from:
    # Note that in Gateway API, Group is currently defaulted
    # to this, which means you to explicitly set the group to
    # the empty string for Core resources. We should definitely
    # change this.
    - group: "gateway.networking.kubernetes.io"
      kind: Gateway
      namespace: foo
  to:
   - group: ""
     kind: Secret
```

The new version will look like this instead:
```yaml
apiVersion: authorization.k8s.io/v1alpha1
kind: ReferenceGrant
metadata:
  name: allow-gateways
  namespace: bar
spec:
  from:
    # Assuming that we leave the default for Group to the empty
    # string, so that Core objects don't need additional config.
    - group: "gateway.networking.kubernetes.io" 
      resource: gateways
      namespace: foo
  to:
    - group: ""
      resource: secrets

```

The new version communicates the scope more clearly because `group`+`resource`
is unambiguous and corresponds to exactly one set of objects on the API Server.

This change also leaves room for an enhancement. Whether we have an in-tree or
CRD implementation, we can rely on the exact matching that the plural resource
name gives us, and [warn](https://kubernetes.io/blog/2020/09/03/warnings/) if
either side of the grant is for an API that's not served by this cluster.

#### Revocation behavior

Unfortunately, there's no way to be specific about what happens when a
ReferenceGrant is deleted in every possible case - the revocation behavior is
dependent on what access is being granted (and revoked). With that said, we
expect the following guidelines to be rules to apply to ALL implementations of
the API:

* Deletion of a ReferenceGrant means the granted access is revoked.
* ReferenceGrant controllers must remove any configuration generated by the
  granted access as soon as possible (eventual consistence permitting).
* Some actions that have already been enabled by the ReferenceGrant (such as
  forwarding requests or persisting data) cannot be undone, but no future
  actions should be allowed.

The examples below include information about what happens when the ReferenceGrant
is removed as data points.

### Example Usage

#### Gateway API Gateway Referencing Secret

In this example (from the Gateway API docs), we have a Gateway in the
`gateway-api-example-ns1` namespace, referencing a Secret in the
`gateway-api-example-ns2` namespace. The following ReferenceGrant allows this:

```yaml
apiVersion: gateway.networking.k8s.io/v1beta1
kind: Gateway
metadata:
  name: cross-namespace-tls-gateway
  namespace: gateway-api-example-ns1
spec:
  gatewayClassName: acme-lb
  listeners:
  - name: https
    protocol: HTTPS
    port: 443
    hostname: "*.example.com"
    tls:
      # There's a Kind/Resource mismatch here, which sucks, but it is not
      # easily fixable, since Gateway is already a beta, close to GA
      # object.
      certificateRefs:
      - kind: Secret
        group: ""
        name: wildcard-example-com-cert
        namespace: gateway-api-example-ns2
---
apiVersion: authorization.k8s.io/v1alpha1
kind: ReferenceGrant
metadata:
  name: allow-ns1-gateways-to-ref-secrets
  namespace: gateway-api-example-ns2
spec:
  from:
  - group: gateway.networking.k8s.io
    resource: gateways
    namespace: gateway-api-example-ns1
  to:
  - group: ""
    resource: secrets
```

For Gateway TLS references, if this ReferenceGrant is deleted (revoking, 
the grant), then the Listener will become invalid, and the configuration
will be removed as soon as possible (eventual consistency permitting).

#### Gateway API HTTPRoute Referencing Service

In this example, a HTTPRoute in the `baz` namespace is directing traffic
to a Service backend in the `quux` namespace.

```yaml
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: quuxapp
  namespace: baz
spec:
  parentRefs:
  - name: example-gateway
    sectionName: https
  hostnames:
  - quux.example.com
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /
    # BackendRefs are Services by default.
    backendRefs:
    - name: quuxapp
      namespace: quux
      port: 80
---
apiVersion: authorization.k8s.io/v1alpha1
kind: ReferenceGrant
metadata:
  name: allow-baz-httproutes
  namespace: quux
spec:
  from:
  - group: gateway.networking.k8s.io
    resource: httproutes
    namespace: baz
  to:
  - group: ""
    resource: services
```

For HTTPRoute objects referencing a backend in another namespace, if the
ReferenceGrant is deleted, the backend will become invalid (since the target
can't be found). If there was more than one backend, then the valid parts of the
HTTPRoute's config would persist in the data plane.

But in this case, the cross-namespace reference is the _only_ backend, so the
removal of the ReferenceGrant will also result in the removal of the HTTPRoute's
config from the data plane.

#### PersistentVolumeClaim using cross namespace data source

This example is taken from https://kubernetes.io/blog/2023/01/02/cross-namespace-data-sources-alpha/
and updated to use the proposed new spec.

It allows the PersistentVolumeClaim in the `dev` namespace to use a volume
snapshot from the `prod` namespace as its data source.

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: example-pvc
  namespace: dev
spec:
  storageClassName: example
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
  dataSourceRef:
    apiGroup: snapshot.storage.k8s.io
    kind: VolumeSnapshot
    name: new-snapshot-demo
    namespace: prod
  volumeMode: Filesystem
---
apiVersion: authorization.k8s.io/v1alpha1
kind: ReferenceGrant
metadata:
  name: allow-prod-pvc
  namespace: prod
spec:
  from:
  - resource: persistentvolumeclaims
    namespace: dev
  to:
  - group: snapshot.storage.k8s.io
    resource: volumesnapshots
    name: new-snapshot-demo
```

When a ReferenceGrant is deleted, any existing volumes created from the
cross-namespace datasource will still persist, but new volumes will be
rejected".

### API Spec

```golang
// ReferenceGrant identifies kinds of resources in other namespaces that are
// trusted to reference the specified kinds of resources in the same namespace
// as the policy.
//
// Each ReferenceGrant can be used to represent a unique trust relationship.
// Additional ReferenceGrants can be used to add to the set of trusted
// sources of inbound references for the namespace they are defined within.
//
// ReferenceGrant is a form of runtime verification allowing users to assert
// which cross-namespace object references are permitted. Implementations that
// support ReferenceGrant MUST NOT permit cross-namespace references which have
// no grant, and MUST respond to the removal of a grant by revoking the access
// that the grant allowed. 
//
// Implementation of ReferenceGrant is eventually consistent, dependent on
// watch events being received from the Kubernetes API. Although some processing
// delay is inevitable, any updates that could result in revocation of access MUST
// be considered high priority and handled as quickly as possible.
//
// Implementations of ReferenceGrant MUST treat all of the following scenarios
// as equivalent:
//
// * A reference to a Namespace that doesn't exist
// * A reference to a Namespace that exists and a Resource that doesn't exist
// * A reference to Namespace and Resource that exists but a ReferenceGrant
//   allowing the reference does not exist
//
// If any of the above occur, a generic error message such as "RefNotPermitted" 
// should be communicated, likely via status on the referring resource.
//
// Support: Core
type ReferenceGrant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of ReferenceGrant.
	Spec ReferenceGrantSpec `json:"spec,omitempty"`

	// Note that `Status` sub-resource has been excluded at the
	// moment as it was difficult to work out the design.
	// A `Status` sub-resource may be added in the future.
}

// +kubebuilder:object:root=true
// ReferenceGrantList contains a list of ReferenceGrant.
type ReferenceGrantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReferenceGrant `json:"items"`
}

// ReferenceGrantSpec identifies a cross namespace relationship that is trusted
// for Gateway API.
type ReferenceGrantSpec struct {
	// From describes the trusted namespaces and kinds that can reference the
	// resources described in "To". Each entry in this list MUST be considered
	// to be an additional place that references can be valid from, or to put
	// this another way, entries MUST be combined using OR.
	//
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	From []ReferenceGrantFrom `json:"from"`

	// To describes the resources in this namespace that may be referenced by
	// the resources described in "From". Each entry in this list MUST be
	// considered to be an additional set of objects that references can be
	// valid to, or to put this another way, entries MUST be combined using OR.
	//
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	To []ReferenceGrantTo `json:"to"`
}

```
<<[UNRESOLVED Are "To" and "From" the right field names? ]>>
Previous Discussion: https://github.com/kubernetes/enhancements/pull/3767#discussion_r1084671720

Comments from @thockin:

I would not argue for subject/object - that's confusing, and I love to take
analogies too far.

NetPol uses To/From but only for actual communications, and that still has been
critiqued as perhaps "too cute".

Subject/From isn't too bad, but not as symmetric. Subject/Referrer is correct
but decidedly uncute. Subject/Origin?

I hold opinions from an API review POV, but I'd like sig-auth to own the
decision :)

For reference, there was an [earlier
discussion](https://groups.google.com/g/kubernetes-api-reviewers/c/ldmrXXQC4G4)
on the kubernetes-api-reviewers mailing list that's also relevant to this.
<<[/UNRESOLVED]>>

```go

// ReferenceGrantFrom describes trusted namespaces and kinds.
type ReferenceGrantFrom struct {
	// Group is the group of the referent.
	// When empty, the Kubernetes core API group is inferred.
	Group Group `json:"group"`

	// Resource is the resource of the referent.
	Resource string `json:"resource"`

	// Namespace is the namespace of the referent.
	Namespace string `json:"namespace"`
}

// ReferenceGrantTo describes what Kinds are allowed as targets of the
// references.
type ReferenceGrantTo struct {
	// Group is the group of the referent.
	// When empty, the Kubernetes core API group is inferred.
	Group string `json:"group"`

	// Resource is the resource of the referent.
	Resource string `json:"resource"`

	// Name is the name of the referent. When unspecified, this policy
	// refers to all resources of the specified Group and Kind in the local
	// namespace.
	//
	// +optional
	Name *string `json:"name,omitempty"`
}
```

### Outstanding questions and clarifications

This section lists some of the outstanding questions people have had about
ReferenceGrant, and other items that we'll be clarifying in the godoc and other
documentation as part of the transition to the new API group, along with any
other changes we need to make that aren't already reflected in this document.

Also note that we don't consider any of these blockers for the general _idea_ of
moving ReferenceGrant to the new API group, just notes to save discussion time.

* Clarify that an implementation is required to reconcile ReferenceGrant for
  specific `To` Kinds.
* Corollary for future work, define how controllers interact. Is it a problem if
  multiple controllers consume the same ReferenceGrant?
* Status design is still pending, but it's currently expected that controllers
  will indicate status on the _referring_ resources, not on ReferenceGrant itself.
<<[UNRESOLVED do we need status? ]>>
Original Thread: https://github.com/kubernetes/enhancements/pull/3767#discussion_r1084670421

We want to be able to represent the following, in descending order of 
importance: 

1. Communicate that the ReferenceGrant is actively being used
2. Communicate which controllers are using this ReferenceGrant
3. Communicate how many times it's been used with sufficient granularity that I
   can see the effects of my changes (if I remove this reference, am I removing
   a dependency on this ReferenceGrant?)

We could introduce a status structure that allowed each implementing controller
to write 1 entry:

```yaml
status:
  referencesAllowed:
  - controllerName: gateway.example.com
    numReferences: 1
```

@thockin responded with:

If we think that the cardinality of controllers is low, we can put it in status.
The downsides are:

1. Could frequently require retries because of optimistic concurrency failures
   (I'm trying to increment my count, but so is everyone else)
1. If we're wrong about cardinality, there's not an easy way out
1. Lots of writes to a resource that will be watched fairly often (every
   controller which needs refs will watch all refgrants)
1. We need .status.
1. If we instead put that into a ReferenceGrantUse resource (just a tuple of
   controller-name and count), then we only have optimistic concurrency
   problems with ourselves, we have ~infinite cardinality, nobody will be
   watching them, and RefGrant doesn't need .status.

Downsides:

1. It's another new resource
1. It's a new pattern, untested in other places.

<<[/UNRESOLVED]>>
* Clarify that the expected operating model for implementations expects them to
  have broad, read access to both ReferenceGrant and the specific `To` Kinds they
  support, and then self-limit to only _use_ the relevant ones.
* Decide whether to formally add `*` as a special value for Namespace, to mean
  "all namespaces".

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

This is a net new resource to Kubernetes so it will not require any changes or
additions to existing tests.

##### Unit tests

Unit tests will be used to cover:

1. ReferenceGrant validation
2. ReferenceGrant implementation library

Test Cases of the ReferenceGrant implementation library will cover the
following:

* A reference to a Namespace that doesn't exist
* A reference to a Namespace that exists and a Resource that doesn't exist
* A reference to a Namespace and Resource that exists but a ReferenceGrant
  allowing the reference does not exist
* Multiple entries in both from and to entries within a ReferenceGrant
* A ReferenceGrant that allows references to kinds of resources that do not
  exist
* Multiple ReferenceGrants with partially overlapping grants
* Revocation of a ReferenceGrant with partially overlapping grants
* A ReferenceGrant that does not specify `to.name`
* A ReferenceGrant that includes overlapping grants for the same namespace both
  with and without the resource name specified
* A reference that has not been allowed by any ReferenceGrants
* A ReferenceGrant that is ineffective due to the wrong `from.namespace` value
* A ReferenceGrant that is ineffective due to the wrong `from.group` value
* A ReferenceGrant that is ineffective due to the wrong `from.resource` value
* A ReferenceGrant that is ineffective due to the wrong `to.group` value
* A ReferenceGrant that is ineffective due to the wrong `to.resource` value
* A ReferenceGrant that is ineffective due to the wrong `to.name` value
* A ReferenceGrant that is ineffective due to being in the wrong namespace

More details will be added as the details of the implementation library are
clarified.

##### Integration tests

Before this graduates to beta, we will provide a reference implementation with a
sample CRD that will be used to provide integration tests.

##### e2e tests

We will strongly encourage every API that uses ReferenceGrant to define
conformance tests for their use of ReferenceGrant.

### Graduation Criteria

#### Beta

[ ] Reference implementation with integration tests.
[ ] Almost all of the fields and behavior have conformance test coverage in at least one project (for example Gateway API).

#### GA

[ ] Conformance tests that exercise all ReferenceGrant API calls (not the actual implementation of the API).
[ ] Multiple implementations of this API passing all relevant conformance tests.

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

Version skew is a bit different here. Although we will provide a shared library
for implementing this API, this will only be used by third-party controllers,
not built-in components.

There will be some implementations that support both the API defined by Gateway
API and this API. Since these resources are entirely additive and can be
duplicative, we can copy Gateway API resources to the new API group and delete
the old Gateway API resources as part of a seamless migration. We expect that
many implementations will provide this recommendation to users, and we may even
provide tooling to simplify this process.

<<[UNRESOLVED open questions that don't clearly fit elsewhere ]>>
## Open Questions

This KEP was merged in a provisional state with a number of open questions
remaining. This section highlights the questions we have not resolved yet.

### Do We Need Verbs?

Previous Discussion: https://github.com/kubernetes/enhancements/pull/3767#discussion_r1084509958

We could add a `verbs` field to enable users to specify the kind of referential
access they want to grant. For example, we could define "read", "route", and
"backup" as well-known verbs to start. We could also allow implementations to
support additional domain-prefixed verbs.

This would enable more precise grants, and potentially more open ended fields
elsewhere in the resource, see the next item for more.

### Do We Need Any or Wildcard Selectors?

Previous Discussions:
* https://github.com/kubernetes/enhancements/pull/3767#discussion_r1086020464
* https://github.com/kubernetes/enhancements/pull/3767#discussion_r1086012665

We already allow "Name" to be optional in `To`, effectively resulting in
wildcard behavior. Should we expand that to allow any of the following?

1. References to any group or resource
1. References from any group or resource
1. References from any namespace

Historically, there has been pushback to allowing any group or resource
because it could enable unknown or unintended kinds of access. If we
introduced the concepts of "verbs" described above, this would become
a moot point.

### Do We Need Label Selectors?

Previous Discussions:
* https://github.com/kubernetes/enhancements/pull/3767#discussion_r1084492070
* https://github.com/kubernetes/enhancements/pull/3767#discussion_r1084674648

As a natural next step, instead of simply allowing all, we could use
label selectors to enable:

1. Access to resources with specific labels
1. Access from namespaces with specific labels

In both cases, this would significantly increase implementation
complexity.

As a potential middleground, we could explore a solution that left
room for namespace selectors without actually including them. For example: 

```go
type ReferenceGrantFrom struct {
  //...
  Peer ReferenceGrantPeer
}

// Exatcly one field should be set. If none are found, clients must
// assume that an unknown value was specified.
type ReferenceGrantPeer {
    Namespace *string
    // Future: Namespace selector
}
```

This assumes that group+resource supersedes namespaces - is that true? Or do we
really want:

```go
type ReferenceGrant struct {
    // ...
    From ReferenceGrantPeer
}

type ReferenceGrantPeer struct {
    // ...
    Namespace *string
    NamespaceResource *NamespaceResource
}

type NamespaceResource struct {
    Namespace string
    Group string
    Resource string
}
```
<<[/UNRESOLVED]>>


## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Other
  - Describe the mechanism: Enable alpha ReferenceGrant API
  - Will enabling / disabling the feature require downtime of the control
    plane?
    No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
    No

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes

###### What happens if we reenable the feature if it was previously rolled back?

The API would become accessible again, implementing controllers may need to be
restarted to pick up the presence of this API.

###### Are there any tests for feature enablement/disablement?

No

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

API enablement may not work, but that would not be unique to this API.

###### What specific metrics should inform a rollback?

N/A, this is just an API

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A, this is just an API

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

N/A, this is just an API

###### How can an operator determine if the feature is in use by workloads?
```
kubectl get referencegrants --all-namespaces
```

###### How can someone using this feature know that it is working for their instance?

This will be dependent on the API that ReferenceGrant is used with. In Gateway API,
each resource has clear status conditions that reflect the validity of a cross-namespace
reference.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Changes to ReferenceGrants are processed by the shared library within 10 seconds 99% over a quarter.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Changes to ReferenceGrants are processed by the shared library within 10 seconds.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

### Dependencies

N/A, this is just an API

###### Does this feature depend on any specific services running in the cluster?

- API Server

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes, users may install controllers that watch for changes to ReferenceGrants.
Users may also create ReferenceGrants to enable cross-namespace references.

###### Will enabling / using this feature result in introducing new API types?

API Type: ReferenceGrant
Supported Number of Objects per Cluster: No limit 
Supported Number of Objects per Namespace: No limit

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The API would not be accessible. We would likely recommend that controllers revoke
cross-namespace references if they could not find ReferenceGrants that allow them
so this could result in a disruption for anything that relied on cross-namespace
references.

###### What are other known failure modes?

N/A, this is just an API

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A, this is just an API

## Implementation History

* July 2021: [ReferencePolicy is proposed in Gateway API](https://github.com/kubernetes-sigs/gateway-api/pull/711)
* August 2021: [ReferencePolicy is added to Gateway API](https://github.com/kubernetes-sigs/gateway-api/pulls?page=2&q=is%3Apr+is%3Aclosed+ReferencePolicy)
* June 2022: [ReferencePolicy is renamed to ReferenceGrant](https://github.com/kubernetes-sigs/gateway-api/pull/1179)
* December 2022: [SIG-Storage uses ReferenceGrant for cross-namespace data storage sources](https://kubernetes.io/blog/2023/01/02/cross-namespace-data-sources-alpha/)
* December 2022: [ReferenceGrant graduates to beta in Gateway API v0.6.0](https://github.com/kubernetes-sigs/gateway-api/releases/tag/v0.6.0)

## Drawbacks

N/A

## Alternatives

1. ReferenceGrant could remain as a CRD
This would probably be fine, we just don't really have a good place for it to
live. This could also complicate installation of Gateway API and other APIs that
depended on this.

2. Every API that wanted to support cross-namespace references could maintain their own version of ReferenceGrant
This would be a confusing mess, we should avoid this at all costs.

