# KEP-4764: Kubectl supports WatchList to list resources

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
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
    - [API server does not support initial events for watches](#api-server-does-not-support-initial-events-for-watches)
    - [<code>ConsistentListFromCache</code> or <code>WatchFromStorageWithoutResourceVersion</code> feature gates not enabled](#consistentlistfromcache-or-watchfromstoragewithoutresourceversion-feature-gates-not-enabled)
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

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Update `kubectl` to support fetching collections using watch lists. This will allow the API server to serve
those requests from its watch cache, which boosts performance and cuts the impact of those request
on the API server.

## Motivation

Currently `client-go` could enable **watchlist** by env and it works great, but `kubectl` is still a pain. 
And `kubectl` doest not have a cache mechanism like `client-go`, so it will do **list**  whenever it is used. Furthermore,
request with pager params can not hit cache of `kube-apiserver`, every request will go through into `etcd`,
which could push up the load of `kube-apiserver` and make system unstable.

In practice, our users have been writing script to unrestrainedly call `kubectl`, which could lead serious performance
issue. As cluster admininstor, we have few method to eliminate it. APF maybe is an option, but it's hard to configure
and don't solve the problem from the root. In the other side, if we are standing in the shoes of user, most of those 
scripts are simple Bash or Python which just simply glue a few `kubectl` commands. It is too heavy and "not worth" 
for user to refactor just "a few commands" into "a batch of complex code". So we are seeking an optimization for `kubectl`.
With this optimization, all users of `kubectl` can get free performance improvement.


### Goals

* Without breaking any user interface, add support to `kubectl` for fetching collections using **watch**.
* Make `kubectl` hit the cache, thus reducing impact on `kube-apiserver`.

### Non-Goals

* Change `kubectl` functionality.
* Deprecate **list** and plan to remove it from `kubectl`.

## Proposal

Existing `kubectl` subcommands like `get`, `describe` should be able to use a _watch list_ to fetch collections 
(list resources), rather than directly list resources page by page.

At this stage, we will adopt a similar approach to `client-go`, using an environment variable to control whether
to enable this experimental feature, and ensuring that it can revert to the previous behavior if it fails.

This proposal focuses on adding support for **watchlist** for better performance. Although **list** will be replaced
by **watchlist** in the long run, deprecation and removal of **list** is not part of this proposal.

### User Stories (Optional)

#### Story 1

Given that there is a large collection in the control plane, a user wants to fetch that collection using `kubectl get`.

As a user, I shouldn't need to worry about the underlying details. I should simply update the version of `kubectl`
to gain the performance improvements for free.

#### Story 2

As a cluster administrator, after upgrading the cluster, I should observe a significant reduction in the requests
penetrating into etcd on the monitoring dashboard. Ideally, the majority of the requests should be cached by the 
apiserver, rather than penetrating into etcd.

### Risks and Mitigations

For users, there are no visible changes to the user interface. However, since different APIs are called, 
some users with restricted permissions may encounter authorization issues. Specifically, users might need
to be granted additional **watch** permissions for the corresponding resources to ensure this mechanism 
works properly. Fortunately, we have a fallback mechanism in this place. Even if users do not have the 
additional permission, the old behavior should still work properly. In the worst case, some extra API 
calls might cause a slight delay.

Regarding security, users who already have **list** permissions inherently have access to all objects of this type.
Therefore, I don't believe this change will introduce any additional security issues.

## Design Details

Since support for watch lists was merged in version 1.27, the API server allows clients to include the 
`sendInitialEvents` query parameter when performing **watch** operations. This parameter instructs the 
kube-apiserver to first send the existing members of the collection as `ADDED` events before pushing subsequent 
objects changed events. A special `BOOKMARK` event with annotation `k8s.io/initial-events-end=true` will be sent 
at the end to indicate the completion of the initial object push.

This mechanism has been integrated into `client-go` in 1.31, and users who use typed or dynamic client can get
this for free by setting env `KUBE_FEATURE_WatchListClient=true`. But unfortunately, in current implementation, 
`kubectl` doesn't not use dynamic or typed client to list resource, it use raw `rest` client. So the auto switching
mechanism implemented in `client-go` can not work with `kubectl` for now.

To address the issue, a relatively simple method is to borrow the mechanism from `client-go`. Specifically, 
by introducing a new environment variable `KUBECTL_WATCHLIST`, and when this environment variable is enabled, 
`kubectl` will use the newly implemented watchlist method to list resources instead of the list method. 
The request parameters will look like this:

```go
  // rely on the `ConsistentListFromCache` feature. When this parameter is an empty string, 
  // the apiserver will ensure that the resource version is the latest version currently in etcd.
  //
  // Additionally, the `WatchFromStorageWithoutResourceVersion` feature is disabled by default in 1.30.
  // In this case, the Watch request will correctly return results from the cache.
	listOptions.ResourceVersion = ""

  // allow the sending of bookmark events. Only when this parameter is enabled will 
  // the aforementioned Bookmark event, used to mark the end of the event sequence, be sent.
	listOptions.AllowWatchBookmarks = true

  // enable ****watchlist****, ask kube-apiserver send initial event.
	listOptions.SendInitialEvents = ptr.To(true)
	listOptions.ResourceVersionMatch = metav1.ResourceVersionMatchNotOlderThan
```

The above parameters can be easily set by reusing the `watchlist.PrepareWatchListOptionsFromListOptions` function, 
thereby maintaining the same behavior as in `client-go`.

When the **watch** request is initiated and begins responding normally, `kubectl` will aggregate the output events. 
A detail to note here is that depending on whether the `ServerPrint` is enabled, the output event types may vary.
They may be output as `metav1.Table` or as the original object types. Upon encountering the `k8s.io/initial-events-end`
event, `kubectl` will proactively close the `watch` and pass the aggregated results to the `VisitorFunc` in one go.
However, if an exception occurs during the above events, kubectl will directly fallback to the original logic, making
a new **list** request.

The above process has been well implemented in `rest.WatchList`, and we can also reuse the implementation. 

In summary, since `kubectl` can't simply use the wrappers provided in `client-go`, it can't benefit from them for free. 
However, we can port the switching mechanism in `client-go` to `kubectl` to make it work.

In addition, there is an additional enhancement that can be applied to this switching mechanism. And this mechanism
will be applied to both `client-go` and `kubectl`. In order to prevent multiple requests to the API server during
a **watchlist** process, and also to make the semantics clear, a hint can be included in the returned error
when the **watch** fails. This hint tells the client that there is no need to fallback to using **list**, 
because **list** shall fail even client does the fallback. Also to be clear, this hint is not guaranteed to be
included in the error. In current considerations, the hint will be included only when the user have no permission
to **list** resources.

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

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

### Graduation Criteria

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

Provided they follow the guidance about permitted version skew, cluster administrators and end users can
upgrade via any supported route. The cluster administrator and the person using `kubectl` do not need to
agree or coordinate about when the client- and server-side features are enabled.

### Version Skew Strategy

This feature will not be enabled by default in `kubectl` until the dependent featuregate in the apiserver 
are enabled by default. However, if users actively enable it while using an older version of the apiserver,
the following scenarios may occur:

#### API server does not support initial events for watches

When the API server is on an older version or the dependent features have not yet been enabled,
the `sendInitialEvent` parameter in the **watch** request will not be correctly processed.
An API server prior to version 1.27 will ignore that query parameter and treat the request as a standard
**watch** request, which may lead to unexpected behavior in `kubectl` unless correctly handled.
In kube-apiserver version 1.27 and later, where the `WatchList` feature gate is disabled, the API server will 
respond with an error as expected; `kubectl` can then fallback to the original **list** verb and associated
client side logic.

#### `ConsistentListFromCache` or `WatchFromStorageWithoutResourceVersion` feature gates not enabled

This will cause requests to directly penetrate into `etcd`, resulting in increased server-side load.

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

- [ ] Feature gate (also fill in values in `kep.yaml`)
- [x] Other
  - When the environment variable `KUBECTL_WATCHLIST` is set to true, kubectl
    uses a **watch** to fetch collections where possible. When the environment variable
    `KUBECTL_WATCHLIST` is set to false, kubectl uses the **list** verb (legacy behavior).
    If `KUBECTL_WATCHLIST` is unset, the kubectl tool uses its compiled-in default.
    
    Once the feature is generally available, `kubectl` will always attempt to use a **watch**
    first.
  - Will enabling / disabling the feature require downtime of the control
    plane?
    No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
    No

###### Does enabling the feature change any default behavior?

`kubectl` will use the _watch list_ mechanism and the **watch** verb to fetch collections, 
only falling back to **list** if a **watch** fails.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

The features can be disabled for a single user by setting the `kubectl` environment
variable associated with the feature to **false**.

###### What happens if we reenable the feature if it was previously rolled back?

The feature does not depend on state, and can be disabled/enabled at will.

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

- There will be unit tests for the `kubectl` environment variable `KUBECTL_WATCHLIST`.

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

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

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

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

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

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

No

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
Yes; `kubectl` will use **watch** rather than **list** to fetch resources list.
Sometimes `kubectl` will then issue a **list** (for example, when the API server has denied the **watch**).

When **watch** API server fails, a hint could be included in response to tell client **list** will also fail (eg. client
have no **list** permission) client don't have to give a try by using **list**, because client know the subsequent 
**list** will fail too.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
No

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

Fetching collections using **watch** may introduce some minor delays due to the need to wait for the arrival of the
`initial-events-end` event. According to the current implementation, this event can be delayed by up to 100 milliseconds.

For small datasets, this delay might be more noticeable. However, due to the performance improvements provided
by the _watch list_ mechanism, the delays caused by the reasons mentioned above may be offset when dealing
with large amounts of data.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No

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

###### What steps should be taken if SLOs are not being met to determine the problem?

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

N/A

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

1. We can continue to use the old previous behavior

   When retrieving large amounts of data based on a list, pagination is necessary. However, during the pagination process,
   the view retrieved in subsequent pages must be consistent with the version at the time of the first page. If such 
   requests are to hit the cache, the API needs to implement a multi-version caching mechanism internally. 
   Currently, only the `EventBuffer` in the `WatchCache` within the API offers a similar mechanism. Implementing a 
   multi-version cache for **list** requests would undoubtedly not only require more memory but also make the implementation 
   more complex.
