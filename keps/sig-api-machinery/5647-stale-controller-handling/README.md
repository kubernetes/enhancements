# KEP-5647: Stale Controller Detection and Mitigation
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
  - [Informer and Cache Update](#informer-and-cache-update)
  - [Staleness Mitigation in Controllers](#staleness-mitigation-in-controllers)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
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

We propose a way to mitigate staleness within Kubernetes controllers,
particularly the ones located in `kube-controller-manager` (KCM). When operating
at a large scale, controllers can fall behind the state of the apiserver,
leading to decisions based on outdated information. This proposal introduces a
generalized mechanism for controllers to detect how stale their local cache is
and prevent reconciliation until the cache catches up to provide consistency.

## Motivation

Every controller operates on a local cache that is populated by watching for
changes from the apiserver. By its nature, this watch stream is eventually
consistent and provides no guarantee of how far behind the "live" state of the
apiserver it is. A change event might arrive within milliseconds, or under other
circumstances, could be delayed by seconds or even minutes. This means every
controller, at all times, is operating on a potentially outdated view of the
cluster's state. 

The issue this KEP addresses is that operators currently have no visibility into
this lag. It's impossible to distinguish between normal delays and a controller
that has fallen dangerously out of sync. A controller may continue trying to
enact its desired state based on an outdated view of the world, causing spurious
reconciles that do not reflect the desired intentions of the user. 

This proposal aims to provide solutions to mitigate issues that may arise out of
the controller falling behind. With the ability to [compare resource
version](https://github.com/kubernetes/enhancements/pull/5505) we now have the
ability to have the idea of the "age" of a resource and run operations with that
in mind.

### Goals

The goals of this KEP is to document the stale controller issue and propose
solutions to help detect and mitigate it. We will propose the ideas of enforcing
stricter cache semantics at certain points in the reconcile cycle of a
controller.

### Non-Goals

This is not intended to enforce consistency guarantees on all controllers. The
changes described here will be "opt-in" for every controller and will require
some degree of changes to the controller logic. It is not the goal to require
controllers to all view a consistent state of the world per reconcile.

## Proposal

We propose the ability for controllers to be able to view whether their own
writes have made it into the apiserver. This will allow for the controller to
skip reconciling an object until it knows whether or not the previous write has
been propogated to its cache. We will do this by adding plumbing to the
underlying cached informer and adding the ability for controllers to know the
resource version of objects they subscribe to. 

Once this is added, we will onboard certain controllers to be able to use the
newly exposed resource versions and skip reconciling and requeue until certain
objects they write to are updated in their cache.

### User Stories (Optional)

#### Story 1

I am a K8s cluster administrator. I want to be sure that my controllers do not
write too frequently. I enable this feature so that I know my controllers only
write when they are up to date.

#### Story 2

I am a controller author. I want certain objects to be ensured to be in my cache
after I write on a previous reconcile so I can be assured that my reads are up
to date. I use the newly provided  frameworks to ensure those objects are up to
date, mitigating the risks that I am operating on stale data. 

### Risks and Mitigations

There is the risk of skipping reconciles, if this is not correct and we don't
unpause properly then that would lead to scenarios where a controller may stop
reconciling an object entirely. We will feature gate this and add a set of
rigorous tests prior to Beta/GA to ensure that the feature is well tested and
not missing edge cases. Any time we try and optimize the reconcile loop there
are issues like this that may arise.

There are also some edge cases that need to be accounted for such as controller
restarts causing the cache to have to be resynced. We will need to account for
scenarios like this and ensure that the controller gets a consistent view of the
world after events like this. While these changes will not make existing
behavior worse, anyone implementing and depending on the ability to read their
writes will need to ensure that situations like that are consistent. Likely, we
will need to have some solutions to
https://github.com/kubernetes/kubernetes/issues/59848#issuecomment-2842495398
before controllers can fully rely on their watch cache on restart. The same
ideas discussed here can likely be applied to have a consistent cache.

## Design Details

Our proposal consists of two parts, one is the change that will enable the
controllers to be fully informed on the current resource versions of objects in
their cache. Second is the use of that new ability to actually ensure the read
after write guarantees on objects we care about in the controller itself.

### Informer and Cache Update

We will update the `ResourceEventHandlerFuncs` in
`staging/src/k8s.io/client-go/tools/cache/controller.go` to have one more function.

```
type ResourceEventHandlerFuncs struct {
	AddFunc      func(obj interface{})
	UpdateFunc   func(oldObj, newObj interface{})
	DeleteFunc   func(obj interface{})
	BookmarkFunc func(resourceVersion string) <--- NEW
}
```

The Add, Update and Delete functions are already nearly enough to be able to
track the lifecycle of objects but certain edge cases in tracking make it
necessary to add the Bookmark function. Controllers will use all the functions
to be able to track when they are able to reconcile after their prior writes.

This Bookmark functions only purpose is to inform any listeners that an update
has occurred. This is necessary to be able to fully track the lifecycle of
resources since otherwise a controller may not be able to tell whether an object
has not yet been added to the cache or whether the object has been added and
deleted without the cache tracking it.

### Staleness Mitigation in Controllers

With the changes in the informers controllers now need mechanisms to actively
prevent issues caused by stale data. The most critical failures often occur when
a controller makes a significant decision—like deleting a Pod or scaling a
resource—based on outdated information from its local cache. To address this, we
will introduce read after write guarantees at these critical decision points.

We will begin by implementing targeted fixes directly within the controllers
that are most sensitive to staleness. To solve this, we will modify the core
processing loop of key controllers. The new approach involves tracking the
resourceVersion of key resources after a successful write operation. This
last-written resourceVersion is stored in memory. We will effectively store the
mapping of the object that is reconciled on, to a tuple of resourceVersions of
any resources that we wish to track. Once the reads of the cache have processed
past the latest write for all the tracked resources we will allow the reconcile
to proceed for that object. This will only skip and requeue reconciles for the
specific objects that have written in a previous reconcile, and won't be a
global lock on all reconciles. By doing so, we ensure that we can progress as
much as possible until we need an updated cache.

We can provide an example with the Daemonset controller. Any time the daemonset
controller writes to pods, we will store the mapping from Daemonset -> Pod
Resource Version. At the same time, we will also add a tracker to the controller
which will update the latest seen RV of the pod cache. On any informer event, we
will update the seen RV if it is greater than what is currently stored. We will
provide some helper function `isReady(dsKey string)` that will query the tracker
and only return true if the resource version of the latest write for pods by
that daemonset is older than the latest read. We can also optimize other parts
in similar ways, such as writes to daemonset status, but they all take the same
pattern.

Lastly, inside of the controllers `processNextWorkItem` function, we will check
whether the daemonset is ready using the helper function and requeue without
running the reconciliation routine if the daemonset is not yet ready to be
worked on due to the cache still needing to catch up. In the case of the object
not being ready, we will requeue the object the same way as if an error
occurred. This will have the same exponential backoff semantics so after a few
reconciles of being unable to catch up the requeue will take longer and longer
until the cache has enough time to actually catch up to the writes.

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

All the new code will be unit tested and have reasonable levels of coverage
prior to merging.

We will be adding changes in the `pkg/controller` and package mostly. The
controller package will contain the bulk of the logic with additions to the pod
informer. We will add unit tests to ensure that if the cache is out of sync, we
skip reconciliation and requeue ourselves.

For the `Informer and Cache Updates` we will be making those changes all in
`staging/src/k8s.io/client-go/tools/cache` folder for the most part. We will add
tests to the newly exposed function and ensure that it works as we expect it to.

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

### Graduation Criteria

#### Alpha
- Addition of new bookmark function for informers
- Feature implemented behind a feature flag for 1 or more controllers
- Unit tests for controllers/bookmark function
- Addition of e2e tests for the controllers with feature gate enabled

#### Beta
- Analysis of onboarded controllers and addition of others that may have the same staleness issues
- Addition of additional E2E tests and stress tests to ensure edge cases are fully tested

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


#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved 

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

<!--
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

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

This is all internal behavior in the sync loop for controllers. Version skew
should not affect it.

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
  - Feature gate name: StaleControllerConsistency
  - Components depending on the feature gate: KCM Controllers that are determined to be high scale

###### Does enabling the feature change any default behavior?
There is no change to the default behavior of reconcilers, however the
reconcilers may skip reconciling for some time until their caches catch up. This
should be invisible but may look like the controllers are stuck when they really
are just waiting for stale caches to catch up.
<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?
Yes, the feature flag does not have any changes that are irreversible. The only
change is in how frequently the reconciliation of key controllers occurs but no
actual effect on any api objects should occur that a rollback would break.
<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?
It should start skipping reconciles for objects on enabled controllers until the
cache catches up again.

###### Are there any tests for feature enablement/disablement?
There are no APIs to enable, we will test for whether the feature gate properly
enables/disables the consistency guarantees.
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
Controller not updating objects as expected while it seems like informer metrics
are fine. We are planning on adding staleness metrics which can help inform on
this.
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
No
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
It will automatically go in use once the gate is enabled
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

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

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

The most significant drawback is the possibility of breaking the controllers by
causing them to not reconcile when they in fact should. To mitigate this as much
as possible, we will only add these semantics to controllers where we have
observed issues due to stale reads and only implement those, especially for the
beginning. If we see success on these controllers, we will look into creating a
framework around this so it is easier to take advantage, but it would already be
a success just for these controllers with documented staleness issues to be
fixed.

## Alternatives

There is not much of an alternative, we want to ensure that reconciles don't
occur on stale caches, so we have to skip reconciliation somehow. We can query
the cache directly and try and figure out the current version that way, but that
can lead to much more racy side effects than implementing a function in the
informer.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
