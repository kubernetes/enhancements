# KEP-4568: Resilient watchcache initialization

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [Reduce the number of requests during initialization](#reduce-the-number-of-requests-during-initialization)
    - [Reject hanging watches](#reject-hanging-watches)
    - [Delegate get requests to etcd](#delegate-get-requests-to-etcd)
    - [Adjust what lists are delegated to etcd](#adjust-what-lists-are-delegated-to-etcd)
    - [Reject the rest of list requests](#reject-the-rest-of-list-requests)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
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

There are several issues that can lead to significant overload of kube-apiserver and
etcd (or worst case even bringing it down) during its initialization or reinitialization
of the watchcache layer. The KEP proposes a solution for mitigating these issues.

## Motivation

Watchcache is a caching layer inside kube-apiserver that is eventually consistent
cache of cluster state stored in etcd - both the state itself as well as the most
recent history of transaction log. It is critical to achieve the control plane
scalability thanks to its built-in indexing and deduplicating of the same work
needed e.g. to distribute the same watch even to multiple watches.

However, there are several issues when watchcache is either not yet initialized
on kube-apiserver startup or requires re-initialization due to not keeping up with
load later on. Both of these cases may lead to significant overload of the whole
control plane (including etcd), in worst case even bringing it down.

The goal of this KEP is to describe the current issues and provide the path for
addressing or at least mitigating them to avoid those control plane outages.

### Goals

- Describe issues that can be faced during watchcache initialization
- Provide and analyze tradeoffs of potential mitigations and decide for path forward

### Non-Goals

- Redesigning watchcache or the storage layer itself


## Proposal

Watchcache is mostly delegating all the write operations to the next layer (being
etcd3 implementation of storage interface). The only thing it is doing (for Update
and Delete calls) is trying to get the current state of the object from its cache
to avoid the need of reading it from etcd (in case of conflict it may still be
needed, but in huge amount of situations it allows to avoid this roundtrip).
What happens during initialization is that the cache is empty, so we're not able
to retrieve the object from cache, in which case we simply avoid passing the
current object down and it's then retrieved from etcd. We didn't observe nor
heard about this causing real issues and we're not going to change this behavior
as part of this KEP.

For watch requests, unless it's a consistent watch request (setting RV="") - which
isn't really used by none of our core components, it is always handled by the
watchcache. For all such requests, watch is hanging until watchcache finishes
initialization. The problem here is that for clusters with a large amount of data
of a given type, initialization (being effectively a list from etcd) can take
tens of seconds or even above a minute and in the meantime this watch is consuming
seats of API Priority&Fairness. In many cases at most tens of such watches (which
is a relatively small number in clusters with thousands of nodes) can effectively
completely starve a given PriorityLevel and block any load on it (both write and
read requests).

For get/list requests that explicitly opt-in for being served from watchcache
(namely those marked as `NotOlderThan` or `Any` in [the documentation]) - which
amongst others is what Reflector/Informers framework is using in most of cases,
if watchcache is not initialized there are two cases:
- if ResourceVersion=0, these are blindly delegated to etcd
- if ResourceVersion>0, they hang waiting for initialization similarly as watch
requests

The first case is particularly problematic for lists that hugely benefit from
watchcache indexing (e.g. listing pods assigned to a given node). When served
from watchcache, these requests are really efficient, but serving them from etcd
requires reading a large amount of data, deserializing it, filtering out objects
not matching selector and sending only those to the client. Passing too many of
such requests down to etcd, can easily bring down the whole control plane.
Additionally, in this case, when served from watchcache a LIMIT parameter is
ignored and the whole result is returned in one go. If served from etcd, the
LIMIT parameter is honored, so for large collections we end-up paging the
response (which for large collections and limit=500 of reflector/informer
frameworks can result in hundreds of consecutive requests for next pages).

The second case is suffering from the same problem as watches, as those requests
may effectively starve the whole PriorityLevel.

In order to mitigate the above problem, we propose a couple different changes.

[the documentation]: https://kubernetes.io/docs/reference/using-api/api-concepts/#semantics-for-get-and-list

#### Reduce the number of requests during initialization

As a first step, we will try to reduce the number of requests that are even reaching
kube-apiserver with unitialized watchcache. To achieve it, we will introduce a new
PostStartHook that will wait for watchcache of all *builtin* resources to be initialized.
Given that the initialization may include KMS usage (e.g. to decrypt secrets), we
will implement a timeout, which will be configurable by the operator via a flag.

Why this should help with reducing the number of requests? The whole idea behind
kube-apiserver readiness is that Kubernetes installations should be configured so that
requests are not sent to it until kube-apiserver becomes ready. The above will allow us
to delay this moment until watchcache is initialized.

Why PostStartHook instead of /readyz check?
The way that PostStartHook work is that it is started exacty [once on startup] and is
[hanging until success]. As soon as it succeeds it [starts reporting success] and never
changes until kube-apiserver stops. We use that mechanism (instead of regular readyz check)
because watchache is per-type layer and we want to avoid marking the whole kube-apiserver
as `not-ready` if watchcache for one of the resource types requires reinitialization,
because requests for all other resource types can still be handled properly.
To handle those cases, we will use a different mechanisms described below (rejection).

[once on startup]: https://github.com/kubernetes/kubernetes/blob/release-1.30/staging/src/k8s.io/apiserver/pkg/server/hooks.go#L155-L168
[hanging until success]: https://github.com/kubernetes/kubernetes/blob/release-1.30/staging/src/k8s.io/apiserver/pkg/server/hooks.go#L194-L206
[starts reporting success]: https://github.com/kubernetes/kubernetes/blob/release-1.30/staging/src/k8s.io/apiserver/pkg/server/hooks.go#L238-L245

Finally, we suggest starting just with builtin resources, as CRDs can be created at
any time, making it harder to implement with unclear gain. If it appears to not be
enough (or alternatively we realize that only a subset of builtin resource types is
needed), this decision can easily be revisited in the future.

#### Reject hanging watches

To mitigate the problem of starving certain PriorityLevels, instead of having the
watch hang and wait for watchcache initialization, we will simply reject it with
`Too Many Requests` 429 http code.
The reason for using this code is that it's already [properly handled] by our client
libraries (e.g. reflector) and will not risk all of them suddenly falling back
to list requests. Moreover, given that 429 is a long supported http code, we expect
that other clients should also properly handle it.

[properly handled]: https://github.com/kubernetes/kubernetes/blob/41e706600aea7468f486150d951d3b8948ce89d5/staging/src/k8s.io/client-go/tools/cache/reflector.go#L910-L920

#### Delegate get requests to etcd

Since GET requests have bounded cost, we always estimate their weight the same
no matter if they are served from etcd or from cache (the latency would be
different in those cases though, so the actual seat-seconds cost too) and
finally given we already did some work to process this request, we would
simply delegate those requests to etcd.

#### Adjust what lists are delegated to etcd

It's tempting to reject all list requests with 429 the same way as watches.
However, that would also have a negative consequences, e.g. by slowing down
kube-apiserver initialization (kube-apiserver on startup need to initialize
its own informers by loop-client, and by rejecting all lists until watchcache
is initialized, we would effectively block that initialization).

While we didn't decide to reject _all_ list requests with 429, we will start with
an approach where we only delegate to etcd the requests that (a) are not
setting any selectors AND (b) are setting a limit.
The first condition means that every object that we process will also be
returned, so we're protecting from requests very selective list requests that
are likely order(s) of magnitude more expensive when served from etcd.
The second condition effectively bounds the cost of the request.

We may decide to adjust it based on further feedback and/or experiments.

#### Reject the rest of list requests

To mitigate the problem of starving certain PriorityLevels, all other list
requests will be rejected with 429 http code similarly to watch requests.


### Risks and Mitigations

Given we're changing the existing behavior, there is a risk of breaking some
clients. To mitigate it, we will introduce all the logic behind two separate
feature gates [to allow for disablement if needed]:
- WatchCacheInitializationPostStartHook (Beta, enabled by default since 1.36):
  handling the logic of the new post-start hook. There is a risk of
  kube-apiserver not initializing if this hook has issues, hence it started
  as Beta disabled by default. By 1.36 we managed to collect enough production data to justify enabling it by default.
- ResilientWatchCacheInitialization (Beta, enabled by default since 1.31):
  Handles the changes to returning 429 errors instead of keeping the requests hanging.
  The risk is visibly lower (and 429 errors were already returned
  before, even in small clusters where priority-levels are small enough
  to often admit only 1-2 inflight requests at once anyway) so the feature was enabled by default.
  This feature has proven stable and is targeting GA in 1.34.

## Design Details

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

- Ensure that k8s.io/apiserver/pkg/storage/cache has time slack for accommodating test
  changes
- Test that TerminateAllWatchers doesn't result in watches ending with an error
- Test that Reflector doesn't get to relisting on TerminateAllWatchers in kube-apiserver

##### Unit tests

- `pkg/controlplane/`: `2024-04-04` - `<test coverage>`
- `k8s/apiserver/pkg/storage/cache`: `2024-04-04` - `<test coverage>`

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

Given we're only modifying kube-apiserver, integration tests are sufficient.


### Graduation Criteria

#### Beta

- Feature implemented behind a feature gate
- Feature is covered with unit and integration tests

#### GA

- Feature was enabled by default allowing us to collect production data.
- No critical issues reported during the period that would block graduation.
- Any further tuning of the specific request delegation logic is considered an incremental improvement and can be addressed post-GA.

### Upgrade / Downgrade Strategy

The feature is purely in-memory so update/downgrade doesn't require any
specific considerations. 

### Version Skew Strategy

Feature touches only kube-apiserver and coordination between individual
instances is not needed.

## Production Readiness Review Questionnaire


### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ResilientWatchCacheInitilization, WatchCacheINitializationPostStartHook
  - Components depending on the feature gate: kube-apiserver
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

Yes:
- kube-apiserver has a new PostStartHook that may block its readiness for longer time
- some GET/LIST/WATCH requests may now return 429 error when storage layer is not
  initialized (or is re-initializing) [429 error was possible before too, this change
  is only extending the number of situations when this may happen]

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, via disabling the feature-gate in kube-apiserver.

###### What happens if we reenable the feature if it was previously rolled back?

The feature is purely in-memory so it will just work as enabled for the first time.

###### Are there any tests for feature enablement/disablement?

The feature is purely in-memory so feature enablement/disablement will not provide
additional value on top of feature tests themselves.


### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

In case of bugs:
- kube-apiserver may not initialize correct due to buggy implementation of the
  introduced post-start hook
- kube-apiserver may incorrectly reject some requests with 429

###### What specific metrics should inform a rollback?

- kubeapiserver continues to fail /readyz (responds with code other than 200) significantly
   after startup should have completed
- unexpectedly high number of requests finishing with 429 code - check kube-apiserver metric
   `apiserver_request_total` on `code=429` label

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No need for tests, this feature doesn't have any persistent side effects.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

This is control-plane feature, not a workload feature.

###### How can someone using this feature know that it is working for their instance?

The feature is not workload-specific, it only affects if certain API calls will be rejected.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

[API call latency SLO](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/api_call_latency.md)

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

[API call latency SLI](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/api_call_latency.md)

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

Not directly.
However, misbehaving clients may retry requests rejected with 429 without exponential
backoff.

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature is kube-apiserver feature - it just doesn't work if kube-apiserver is unavailable.

###### What are other known failure modes?

- Misbehaving clients not handling 429 `retryAfter` correctly
  - Detection: significant increase in  `apiserver_request_total` metric
  - Mitigations: Identify the faulty client via logs and create a dedicated
      APF PriorityLevel for them.
  - Diagnostics: Request logs should be used to identify faulty clients
  - Testing: No new testing - 429 can already be returned for requests.

###### What steps should be taken if SLOs are not being met to determine the problem?

Disabling the feature-gate.

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

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

