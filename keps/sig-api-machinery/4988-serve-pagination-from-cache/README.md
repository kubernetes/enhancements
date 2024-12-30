# KEP-4988 Serve pagination from cache

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Client setting limit while not supporting pagination](#client-setting-limit-while-not-supporting-pagination)
    - [Memory overhead](#memory-overhead)
    - [Pagination request hitting another apiserver](#pagination-request-hitting-another-apiserver)
    - [Delegating slow pagination to etcd](#delegating-slow-pagination-to-etcd)
    - [Increased watch contention](#increased-watch-contention)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

Enabling pagination typically improves performance by reducing response sizes.
However, the kube-apiserver's caching mechanism (watchcache) does not currently
support pagination. When a client requests a paginated list, the limit parameter
might be ignored, or the request might bypass the cache and be served directly
from etcd, leading to performance issues. This document proposes an enhancement
to the kube-apiserver's caching layer to enable efficient serving of paginated
lists from the cache, improving performance and predictability.

## Motivation

The kube-apiserver's watchcache is an eventually consistent cache of the cluster
state. It allows the API server to serve read requests without constantly hitting
etcd. However, watchcache is not designed for pagination.

Pagination allows clients to request large lists of resources in smaller chunks.
The client specifies a limit (maximum number of objects per page) and receives
a continue token in the response. This token is then used in subsequent requests
to fetch the next page.

To serve paginated requests correctly, the API server needs access to a consistent
snapshot of the data from which the initial page was generated. Watchcache does
not store historical data. Therefore, currently, the API server does the following:

- For `resourceVersion=""`: Serve paginated requests directly from etcd, bypassing the cache.
- For `resourceVersion="0"`: Ignore the limit parameter and return the entire list in a single response.

Both options are problematic:

- Serving from etcd negates the benefits of the cache,
  leading to increased load on both the API server and etcd.
- Ignoring the limit parameter defeats the purpose of pagination,
  potentially returning massive responses that can overwhelm clients.

This situation leads to unexpected behavior for users,
where enabling pagination can degrade performance instead of improving it.

### Goals

- Improve the performance of paginated list requests to be comparable to non-paginated lists.
- Make pagination behavior predictable and consistent.
- Reduce load on etcd caused by paginated requests.

### Non-Goals

- Serve `resourceVersion="N"` request from watch cache
- Support indices when paginating.
- Eliminate all paginated list requests to etcd.

## Proposal

Leveraging the recent rewrite of the watchcache storage layer to use a B-tree
(https://github.com/kubernetes/kubernetes/pull/126754), we propose to utilize
B-tree snapshots to serve paginated lists.

Mechanism:
1. **Snapshot Creation:** When a paginated list request (with a limit parameter
   set) is received, the API server will create a snapshot using the efficient
   [Clone()](https://pkg.go.dev/github.com/google/btree#BTree.Clone) method.
   This clone is a lazy copy, only duplicating necessary nodes, resulting in
   minimal overhead.
2. **Snapshot Storage:** The snapshot will be stored in memory, keyed by
   resourceVersion.
3. **Serving Subsequent Pages:** When a subsequent request with a continue token
   arrives, the API server will:
   - Extract the resourceVersion from the continue token.
   - Retrieve the corresponding snapshot using the resourceVersion as the key.
   - Use the snapshot to serve the requested page, or pass the request to etcd is snapshot is not available.
4. **Snapshot Cleanup:** Snapshots will be subject to a Time-To-Live (TTL)
   mechanism. We will reuse the existing watch event cleanup logic, which has a
   75s TTL. This ensures that snapshots don't accumulate indefinitely.

As still some pagination requests will be delegated to etcd, we will monitor the
success rate by measuring the pagination cache hit vs miss ratio.

Consideration: Should we start respecting the limit parameter?

### Risks and Mitigations

#### Client setting limit while not supporting pagination

#### Memory overhead

No, B-tree only store pointers the actual objects, not the object themselves.
The objects are already cached to serve watch, so it should only add a small
overhead for the B-tree structure itself, which is negligible compared to the
size of the cached objects.

#### Pagination request hitting another apiserver

When connecting directly expectation is that a client will stay connected to the
same apiserver. What happens when apiserver is behind loadbalancer?

For setups with L4 loadbalancer apiserver can be configured with Goaway, which
requests client reconnects periodically, however per request probability should
be configured around 0.1%.

For L7 loadbalancer the default algorithm usually is round-robin. For most LBs
it should be possible to switch the algorithm to be based on source IP hash.
Even if that is not possible, stored snapshots will never be used and user will
not be able to benefit from the feature.

#### Delegating slow pagination to etcd

To avoid breaking users the proposal still allows pagination requests older than
75s to pass to etcd. This can have a huge performance impact if the resource is
large. However, this seems still safer than:
* Increasing the watch cache size 4 times to match etcd.
* Block requests older than 75s

#### Increased watch contention

As pagination is not served from the main watch cache storage, but from separately
stored snapshots, pagination request can be served without taking the main lock,
and just depend on watch cache

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

- Ensure the pagination is well tested

##### Unit tests

- `k8s/apiserver/pkg/storage/cache`: `2024-12-12` - `<test coverage>`

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

#### Alpha

- Feature implemented behind a feature gate
- Feature is covered with unit and integration tests

#### Beta

- Feature is enabled by default

#### GA

TODO

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
  - Feature gate name: PaginationFromCache
  - Components depending on the feature gate: kube-apiserver
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

Yes, kube-apiserver paginating LIST requests will no longer require request to etcd.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, via disabling the feature-gate in kube-apiserver.

###### What happens if we reenable the feature if it was previously rolled back?

The feature is purely in-memory so it will just work as enabled for the first time.

###### Are there any tests for feature enablement/disablement?

The feature is purely in-memory so feature enablement/disablement will not provide
additional value on top of feature tests themselves.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?


###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

NO

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

This is control-plane feature, not a workload feature.

###### How can someone using this feature know that it is working for their instance?

This is control-plane feature, not a workload feature.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

[API call latency SLO](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/api_call_latency.md)

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

[API call latency SLI](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/api_call_latency.md)

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No, we expect the [API call latency SLI](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/api_call_latency.md) to improve.


###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Overall we expect that cost of serving pagination will go down, however caching
might increase RAM usage, if the client reads the first page, but never
paginates. We expect that most controllers will read all pages.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature is kube-apiserver feature - it just doesn't work if kube-apiserver is unavailable.

###### What are other known failure modes?

No

###### What steps should be taken if SLOs are not being met to determine the problem?

Disabling the feature-gate.

## Implementation History

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

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
