# KEP-4020: Unknown Version Interoperability Proxy

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Garbage Collector](#garbage-collector)
    - [Namespace Lifecycle Controller](#namespace-lifecycle-controller)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Aggregation Layer](#aggregation-layer)
    - [StorageVersion enhancement needed](#storageversion-enhancement-needed)
    - [Identifying destination apiserver's network location](#identifying-destination-apiservers-network-location)
    - [Proxy transport between apiservers and authn](#proxy-transport-between-apiservers-and-authn)
  - [Discovery Merging](#discovery-merging)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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
  - [Network location of apiservers](#network-location-of-apiservers)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
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

When a cluster has multiple apiservers at mixed versions (such as during an
upgrade/downgrade or when runtime-config changes and a rollout happens), not 
every apiserver can serve every resource at every version.

To fix this, we will add a filter to the handler chain in the aggregator which
proxies clients to an apiserver that is capable of handling their request.

## Motivation

When an upgrade or downgrade is performed on a cluster, for some period of time
the apiservers are at differing versions and are able to serve different sets of
built-in resources (different groups, versions, and resources are all possible).

In an ideal world, clients would be able to know about the entire set of
available resources and perform operations on those resources without regard to
which apiserver they happened to connect to. Currently this is not the case.

Today, these things potentially differ:
* Resources available somewhere in the cluster
* Resources known by a client (i.e. read from discovery from some apiserver)
* Resources that can be actuated by a client

This can have serious consequences, such as namespace deletion being blocked
incorrectly or objects being garbage collected mistakenly.

### Goals

* Ensure that a resource request is handled by an apiserver that is capable of serving that resource (if one exists)
* In the failure case (e.g. network not routable between apiservers), ensure
  that unreachable resources are served 503 and not 404.
* Ensure discovery reports the newest set of resources available in a cluster
* Ensure that a resource request is handled by an apiserver that is compatible with the client making the request. This means
  * Establish a way for clients to propagate compatibility version information in requests to servers
  * We use compatibility version information from the incoming request and proxy the request to an API server with a compatibility version >= client’s compatibility version (if such an apiserver exists)


### Non-Goals

* Lock particular clients to particular versions

## Proposal

We will use the existing [StorageVersion API](https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/2339-storageversion-api-for-ha-api-servers/README.md) to figure out which group, versions,
and resources an apiserver can serve.

API server change:
* A new handler is added to the stack:

  - If the request is for a group/version/resource the apiserver doesn't have
    locally (we can use the StorageVersion API), it will proxy the request to
    one of the apiservers that is listed in the [ServerStorageVersion](https://github.com/kubernetes/kubernetes/blob/release-1.27/pkg/apis/apiserverinternal/types.go#L64) object. If an apiserver fails
    to respond, then we will return a 503 (there is a small
    possibility of a race between the controller registering the apiserver
    with the resources it can serve and receiving a request for a resource
    that is not yet available on that apiserver).

* Discovery handling:

  - We will proxy the discovery request to the newest available apiserver in the cluster. In doing so, we will ensure that all discovery requests are handled keeping the future state of the cluster in mind. 

* Version Aware Proxying:

  - With [compatibility versions](https://github.com/kubernetes/enhancements/pull/4395) information available for kubernetes components, we can leverage that to perform version aware proxying between API servers, and ensure that a request is proxied to an eligible apiserver given that it was generated by a client compatible with the apiserver/s present in a cluster. Version aware proxying lets us ensure that requests from clients (possibly) expecting to use a feature introduced in later K8s versions,  are not incorrectly routed to apiservers at older versions

  - We will add a new header in client requests to specify the compatibility version of the client 
  
  - We will publish compatibility version, binary-version of an apiserver as annotations in the apiserver-identity lease

### User Stories (Optional)

#### Garbage Collector

The garbage collector makes decisions about deleting objects when all
referencing objects are deleted. A discovery gap / apiserver mismatch, as
described above, could result in garbage collector seeing a 404 and assuming an object has been
deleted; this could result in it deleting a subsequent object that it should
not.

This proposal will cause the garbage collector to behave in the following way for the scenarios described:

- Case 1: if the resource is being deleted in the newer version 
    - Any actions taken on this resource during the upgrade will render useless after the upgrade when these objects are orphaned so we can act fast and proxy this request to the newer apiserver. Garbage collection will be triggered if the list of resources returned by the newest apiserver has objects that have no owner references. Since the deleted resource is not going to be found in the discovery document, garbage collector will take no action on it and its children objects  
- Case 2: if the resource is being introduced in the newer version
    - The new resources are going to be included in the discovery document returned by the newest apiserver. Garbage collector will take the required actions on the new resources depending on whether any owner references were found for them. This scenario is handled safely
- Case 3: If the resource state remains the same before and after the upgrade
    - If the resource state remains the same before and after the upgrade, garbage collector's behavior is unchanged

#### Namespace Lifecycle Controller

This controller seeks to empty all objects from a namespace when it is deleted.
Discovery failures cause Namespace Lifecycle Controller to be unable to tell if objects of a given resource
are present in a namespace. It fails safe, meaning it refuses to delete the
namespace until it can verify it is empty: this causes slowness deleting
namespaces that is a common source of complaint.

Additionally, if the Namespace Lifecycle Controller knows about a resource that the apiserver it is talking
to does not, it may incorrectly get a 404, assume a collection is empty, and
delete the namespace too early, leaving garbage behind in etcd. This is a
correctness problem, the garbage will reappear if a namespace of the same name
is recreated.

This proposal will cause the Namespace Lifecycle Controller to behave in the following way for the scenarios described:

- Case 1: if the resource is being deleted in the newer version
  - if the resource is actually being deleted in the newer version, once namespace deletion is triggered, the namespace lifecycle controller will try to find all objects that belong to this namespace and delete them. It wont know a particular resource type if that resource is not returned by the newest apiserver's discovery document. This can cause the namespace lifecycle controller to delete the namespace leaving behind orphan objects  
- Case 2: If the resource is being introduced in the newer version
  - if namespace deletion is triggered in this case, the namespace lifecycle controller will find all the resources served by the newest apiserver and it will delete those objects (including the new ones introduced in the newer version) before deleting the namespace. This scenario is handled safely 
- Case 3: If the resource state remains the same before and after the upgrade 
  - If the resource state remains the same before and after the upgrade - namespace lifecycle behavior is unchanged

### Notes/Constraints/Caveats (Optional)

#### Version Aware Proxying

A feature can dynamically be turned on/off in a cluster, so there's no way to know just by a client request or even by apiservers’ compatibility versions, which feature that request is going to interact with. So if we have 2 apiservers at the same compatibility version but varying features enabled, MVP will not be able to distinguish between these apiservers to ensure that a request using a particular feature is proxied to the right apiserver

### Risks and Mitigations

1. **Network connectivity isues between apiservers**
    
    Cluster admins might not read the release notes and realize they should enable network/firewall connectivity between apiservers. In this case clients will receive 503s instead of transparently being proxied. 503 is still safer than today's behavior. We will clearly document the steps needed to enable the feature and also include steps to verify that the feature is working as intended. Looking at the following exposed metrics can help wth that 
    1. `kubernetes_apiserver_rerouted_request_total` to monitor the number of (UVIP) proxied requests. This metric can tell us the number of requests that were successfully proxied and the ones that failed
    2. `apiserver_request_total` to check the success/error status of the requests

2. **Increase in egress bandwidth**
    
    Requests will consume egress bandwidth for 2 apiservers when proxied. We can cap the number if needed, but upgrades aren't that frequent and few resources are changed on releases, so these requests should not be common. We will count them with a metric.

3. **Increase in request traffic directed at destination kube-apiserver**

    There could be a large volume of requests for a specific resource which might result in the identified apiserver being unable to serve the proxied requests. This scenario should not occur too frequently, since resource types which have large request volume should not be added or removed during an upgrade - that would cause other problems, too.

4. **Indefinite rerouting of the request**
    
    We should ensure at most one proxy, rather than proxying the request over and over again (if the source apiserver has an incorrect understanding of what the destination apiserver can serve). To do this, we will add a new header such as `X-Kubernetes-APIServer-Rerouted:true` to the  request once it is determined that the request cannot be served by the local apiserver and should therefore be proxied.  
    We will remove this header after the request is received by the destination apiserver (i.e. after the proxy has happened once) at which point it will be served locally.

5. **Putting IP/endpoint and trust bundle control in user hands in REST APIs**
    
    To prevent server-side request forgeries we will not give control over information about apiserver IP/endpoint and the trust bundle (used to authenticate server while proxying) to users via REST APIs.

## Design Details

### Aggregation Layer

![Alt text](https://user-images.githubusercontent.com/26771552/244544622-8ade44db-b22b-4f26-880d-3eee5bc1f913.png?raw=true "Optional Title")

1. A new filter will be added to the [handler chain] of the aggregation layer. This filter will maintain an internal map with the key being the group-version-resource and the value being a list of server IDs of apiservers that are capable of serving that group-version-resource
   1. This internal map is populated using an informer for StorageVersion objects. An event handler will be added for this informer that will get the apiserver ID of the requested group-version-resource and update the internal map accordingly

2. This filter will pass on the request to the next handler in the local aggregator chain, if:
   1. It is a non resource request
   1. The StorageVersion informer cache hasn't synced yet or if `StorageVersionManager.Completed()` has returned false. We will serve error 503 in this case
   1. The request has a header `X-Kubernetes-APIServer-Rerouted:true` that indicates that this request has been proxied once already. If for some reason the resource is not found locally, we will serve error 503
   1. the StorageVersion has an apiserverID that does not match one of the known peer apiserverIDs, indicating that the request is for an aggregated API or a CRD. For aggregated apiservers that have self registered storage version information, this results in the desired behavior- the apiserver handles the request normally, which in this case would entail matching the request with an APIService resource and delegating the request to that aggregated API server
   1. If the local apiserverID is found in the list of serviceable-by server IDs from the internal map and the local apiserver's compatibility version is equal to the compatibility version of the client

1. From the internal map that stores the list of apiserverIDs that can serve a GVR, we will fetch the list of apiservers that have the same compatibility version as the client 
  1. If the local apiserver ID is not found in the above list of serviceable-by server IDs, a random apiserver ID will be selected from the retrieved list and the request will be proxied to this apiserver
1. If there was no apiserverID retrieved for the requested GVR which had compatibility version matching the compatibility version of the client, we will shortlist apiservers that have compatibility versions greater than compatibility version of the client
  1. If the local apiserver ID is not found in the above list of serviceable-by server IDs, a random apiserver ID will be selected from the retrieved list and the request will be proxied to this apiserver
1. If there is no apiserver ID retrieved for the requested GVR at all, we will serve 404 with error `GVR <group_version_resource> is not served by anything in this cluster`

1. If the proxy call fails for network issues or any reason, we serve 503 with error `Error while proxying request to destination apiserver`

1. We will also add a poststarthook for the apiserver to ensure that it does not start serving requests until we are done creating/updating SV objects

[handler chain]:https://github.com/kubernetes/kubernetes/blob/fc8f5a64106c30c50ee2bbcd1d35e6cd05f63b00/staging/src/k8s.io/apiserver/pkg/server/config.go#L639

#### StorageVersion enhancement needed

StorageVersion API currently tells us whether a particular StorageVersion can be read from etcd by the listed apiserver. We will enhance this API to also include apiserver ID of the server that can serve this StorageVersion.

With the enhancement, the new [ServerStorageVersion](https://github.com/kubernetes/kubernetes/blob/release-1.27/pkg/apis/apiserverinternal/types.go#L62-L73) object will have this structure

```
type ServerStorageVersion struct {
  // The ID of the reporting API server.
  APIServerID string
	
  // The API server encodes the object to this version 
  // when persisting it in the backend (e.g., etcd).
  EncodingVersion string

  // The API server can decode objects encoded in these versions.
  // The encodingVersion must be included in the decodableVersions.
  DecodableVersions []string

  // Versions that can be served by the reporting API server
  ServedVersions []string
}
```

#### Identifying destination apiserver's network location

We will be performing dual writes of the ip and port information of the apiservers in:

1. A clone of the [endpoint reconciler's masterlease](https://github.com/kubernetes/kubernetes/blob/release-1.27/pkg/controlplane/reconcilers/lease.go) which would be read by apiservers to proxy the request to a peer. We will use a separate reconciler loop to do these writes to avoid modifying the existing endpoint reconciler

2. [APIServerIdentity Lease object](https://github.com/kubernetes/kubernetes/blob/release-1.27/pkg/controlplane/instance.go#L559-L577) for users to view this information for debugging

3. We will use an egress dialer for network connections made to peer kube-apiservers. For this, will create a new type for the network context to be used for peer kube-apiserver connections ([xref](https://github.com/kubernetes/kubernetes/blob/release-1.27/staging/src/k8s.io/apiserver/pkg/apis/apiserver/types.go#L56-L71))

#### Proxy transport between apiservers and authn

For the mTLS between source and destination apiservers, we will do the following

1. For server authentication by the client (source apiserver) : the client needs to validate the [server certs](https://github.com/kubernetes/kubernetes/blob/release-1.27/staging/src/k8s.io/apiserver/pkg/server/options/serving.go#L59) (presented by the destination apiserver), for which it will 
   1. look at the CA bundle of the authority that signed those certs. We will introduce a new flag --peer-ca-file for the kube-apiserver that will be used to verify the presented server certs. If this flag is not specified, the requests will fail with error 503
   2. look at the ServerName `kubernetes.default.svc` for SNI to verify server certs against

2. The server (destination apiserver) will check the client (source apiserver) certs to determine that the proxy request is from an authenticated client. We will use requestheader authentication (and NOT client cert authentication) for this. The client (source apiserver) will provide the [proxy-client certfiles](https://github.com/kubernetes/kubernetes/blob/release-1.27/cmd/kube-apiserver/app/options/options.go#L222-L233) to the server (destination apiserver) which will verify the presented certs using the CA bundle provided in the [--requestheader-client-ca-file](https://github.com/kubernetes/kubernetes/blob/release-1.27/staging/src/k8s.io/apiserver/pkg/server/options/authentication.go#L125-L128) passed to the apiserver upon bootstrap

### Discovery Merging
TODO: detailed description of discovery merging. (not scheduled until beta.)

### Version Aware Proxying
TODO


### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

In the first alpha phase, the integration tests are expected to be added for:

- The behavior with feature gate turned on/off
- Request is proxied to an apiserver that is able to handle it
- Validation where an apiserver tries to serve a request that has already been proxied once
- Validation where an apiserver tries to call a peer but actually calls itself (to simulate a networking configuration where this happens on accident), and the test fails with error 503
- Validation where a request that cannot be served by any apiservers is received, and is passed on locally that eventually gets handled by the NotFound handler resulting in 404 error
- Validation where apiserver is mis configured and is proxied to an incorrect peer resulting in 503 error 

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

We will test the feature mostly in integration test and unit test. We may add e2e test for spot check of the feature presence.

### Graduation Criteria

#### Alpha

- Proxying implemented (behind feature flag)
- mTLS or other secure system used for proxying
- Ensure proper tests are in place.

#### Beta

- Discovery routing to newest apiserver implemented
- Use egress dialer for network connections made to peer kube-apiservers
- Version aware proying implemented

#### GA

- TODO: wait for beta to determine any further criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

In alpha, no changes are required to maintain previous behavior. And the feature gate can be turned on to make use of the enhancement.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: UnknownVersionInteroperabilityProxy
  - Components depending on the feature gate: kube-apiserver

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Yes, requests for built-in resources at the time when a cluster is at mixed versions will be served with a default 503 error instead of a 404 error, if the request is unable to be served. 

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, disabling the feature will result in requests for built-in resources in a cluster at mixed versions to be served with a default 404 error in the case when the request is unable to be served locally.

###### What happens if we reenable the feature if it was previously rolled back?

The request for built-in resources will be proxied to the apiserver capable of serving it, or else be served with 503 error.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

Unit test and integration test will be introduced in alpha implementation.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

The proxy to remote apiserver can fail if there are network restrictions in place that do not allow an apiserver to talk to a remote apiserver. In this case, the request will fail with 503 error.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

- apiserver_request_total metric that will tell us if there's a spike in the number of errors seen meaning the feature is not working as expected

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Upgrade and rollback will be tested before the feature goes to Beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

The following metrics could be used to see if the feature is in use:
- kubernetes_apiserver_rerouted_request_total

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- Metrics like kubernetes_apiserver_rerouted_request_total can be used to check how many requests were proxied to remote apiserver

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

None have been identified.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [X] Metrics
  - Metric name: `kubernetes_apiserver_rerouted_request_total`
  - Components exposing the metric: kube-apiserver

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

No. We are open to input.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

No, but it does depend on 

- the `StorageVersion` feature that generates objects with a `storageVersion.status.serverStorageVersions[*].apiServerID` field which is used to find the remote apiserver's network location.
- `APIServerIdentity` feature in kube-apiserver that creates a lease object for APIServerIdentity which we will use to store the network location of the remote apiserver for visibility/debugging

<!--
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
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

When handling a request in the handler chain of the kube-aggregator, the StorageVersion informer will be used to look up which API servers can serve the requested resource. 

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

Requests will consume egress bandwidth for 2 apiservers when proxied. We can put a limit on this value if needed.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

If the API server/etcd is unavailable the request will fail with 503 error.

###### What are other known failure modes?

<!--
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
-->
None.

###### What steps should be taken if SLOs are not being met to determine the problem?

- The feature can be disabled by setting the feature-gate to false if the performance impact of it is not tolerable.
- The peer-to-peer connection between API servers should be checked to ensure that the remote API servers are reachable from a given API server

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### Discovery Merging

- During upgrade or downgrade, it may be the case that no apiserver has a
    complete list of available resources. To fix the problems mentioned, it's
    necessary that discovery exactly matches the capability of the system. So,
    we will use the storage version objects to reconstruct a merged discovery
    document and serve that in all apiservers.

Why so much work?
* Note that merely serving 503s at the right times does not solve the problem,
  for two reasons: controllers might get an incomplete discovery and therefore
  not ask about all the correct resources; and when they get 503 responses,
  although the controller can avoid doing something destructive, it also can't
  make progress and is stuck for the duration of the upgrade.
* Likewise proxying but not merging the discovery document, or merging the
  discovery document but serving 503s instead of proxying, doesn't fix the
  problem completely. We need both safety against destructive actions and the
  ability for controllers to proceed and not block.

### Network location of apiservers

1. Use [endpoint reconciler's masterlease](https://github.com/kubernetes/kubernetes/blob/release-1.27/pkg/controlplane/reconcilers/lease.go) 
    1. We will use the already existing IP in Endpoints.Subsets.Addresses of the masterlease by default
    2. For users with network configurations that would not allow Endpoints.Subsets.Addresses to be reachable from a kube-apiserver, we will introduce a new optional --bind-peer-ip flag to kube-apiserver. We will store its value as an annotation on the masterlease and use this to route the request to the right destination server
    3. We will also need to store the apiserver identity as an annotation in the masterlease so that we can map the identity of the apiserver to its IP
    4. We will also expose the IP and port information of the kube-apiservers as annotations in APIserver identity lease object for visibility/debugging purposes

* Pros
  1. Masterlease reconciler already stores kube-apiserver IPs currently
  2. This information is not exposed to users in an API that can be used maliciously
  3. Existing code to handle lifecycle of the masterleases is convenient

* Cons
  1. using masterlease will include making some changes to the legacy code that does the endpoint reconciliation which is known to be brittle

2. Use [coordination.v1.Lease](https://github.com/kubernetes/kubernetes/blob/release-1.27/staging/src/k8s.io/client-go/informers/coordination/v1/lease.go) 
    1. By default, we can store the [External Address](https://github.com/kubernetes/kubernetes/blob/release-1.27/staging/src/k8s.io/apiserver/pkg/server/config.go#L149) of apiservers as labels in the [APIServerIdentity Lease](https://github.com/kubernetes/kubernetes/blob/release-1.27/pkg/controlplane/instance.go#L559-L577) objects. 
    2. If `--peer-bind-address` flag is specified for the kube-apiserver, we will store its value in the APIServerIdentity Lease label
    3. We will retrieve this information in the new UVIP handler using an informer cache for these lease objects 

* Pros
  1. Simpler solution, does not modify any legacy code that can cause unintended bugs
  2. Since in approach 1 we decided we want to store the apiserver IP, port in the APIServerIdentity lease object anyway for visibility to the user, we will be just making this change once in the APIServerIdentity lease instead of both here and in masterleases

* Cons 
  1. If we take this approach, there is a risk of giving the user control of the apiserver IP, port information. This can lead to apiservers routing a request to a rogue IP:port specified in the lease object.

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
