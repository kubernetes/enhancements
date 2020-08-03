# KEP-1904: Efficient watch resumption after kube-apiserver reboot

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
  - [Initialize watch cache from etcd history window](#initialize-watch-cache-from-etcd-history-window)
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
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
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

The kube-apiserver watch cache is initialized from etcd at the moment
when it starts with empty change history. As a consequence, clients
that want to resume a watch immediately after kube-apiserver reboots
almost always have a resource version that is out of the history window.
In this KEP we propose how to address this problem.

## Motivation

In order to support large Kubernetes clusters without huge number of etcd
and kube-apiserver replicas, kube-apiserver contains an in-memory cache
called `watch cache` from which all watch requests as well as non-quorum
reads (in opt-in fashion) are served. Watch cache is being propagated
directly from etcd using typical "list + watch" pattern and stores both
the current state as well as some recent 'transaction log'. There is a
separate watch cache for each resource type version.

Digging into more details, the way how watch cache is propagated is:
- on start, quorum list is send to etcd
- given the RV of the list represents the current "global etcd version",
  it doesn't have to reflect any change of objects of our type
- the watch cache is set to be synced to list resource version (which
  as mentioned above isn't necessary reflecting any change of objects
  of that type)
- from the on, watch to etcd is established and watch cache is being
  updated by incoming watch event stream; note that only objects of
  a given resource type are being watched and only those can update
  the resource version to which watch cache is synced from then on
- if watch cannot be reestablished, all watchers connected to it are
  disconnected, a new quorum list is send and watchcache is reset to
  that point in time

It's important to note that while only objects of a given resource type
are watched (thus resource version to which watch cache is synced can
only be updated to a value of a change (create/update/delete) of
some object of that type), the quorum ist, even though it is only
requesting objects of that resource type, returns "globally current
version for the whole etcd cluster" (objects of different types may
be stored in different etcd clusters, and only those that are stored in
the same etcd cluster matter).

While the inability to reestablish watch doesn't happen often in practice,
the kube-apiserver restarts obviously does (e.g. on upgrades). And this
is resulting in two main problems:

1. In case of rolling upgrade of kube-apiserver, when no object of a given
   resource type is changing (but objects of other types do), all watchers
   will eventually be forced to relist, causing significant performance
   and scalability issues for larger clusters. See the example below for
   clarification.

1. If no object of a given resource type is changing after kube-apiserver
  startup, watch cache may stuck being synced to resource version not being
	version of any object of a given type for extended period of time. See
  https://github.com/kubernetes/kubernetes/issues/91073 for more details.

To illustrate the problem better, imagine the following example:
- there is a single kube-apiserver that we are going to restart
- there are only two resource types: X and Y (for simplicity)
  [Y is needed to have RV changes by non-X objects]
- there is one object of type X: x with rv=100
- there is one object of type Y: y with rv=101
- there is a watch W for objects of type X already synced to rv=100
- watchcache for both X and Y is up-to-date

- T1: kube-apiserver being restarted
- T2: kube-apiserver is initialized (via quorum LIST from etcd) with
	 watchcache for X synced to rv=101 (most recent etcd version)
- T3: watch W is trying to reconnect, given rv=100 is outside of
   supported history window (we only support versions from 101 being
   the moment of watch cache initialization), relist is forced

Note that adding more kube-apiservers doesn't solve the problem. In
fact, it is introducing the second problem where watch cache across
kube-apiserver instances may be out of sync for extended period of time.

To illustrate this problem imagine the following example:
- the setup is as above, with the only difference of two kube-apiservers

- T1: kube-apiserver-1 being restarted
- T2: kube-apiserver-1 is initialized with watchcache for X synced to
   rv=101 (most recent etcd version) [as above]
- T3: object y is being updated to rv=102
- T4: kube-apiserver-2 is being restarted
- T5: kube-apiserver-2 is initialized with watchcache for X synced to
   rv=102 (most recent etcd version)

As long as no object of type X is being touched (created, updated or
deleted), watchcache for X will not be updated. So even though they
contain the same set of objects, one claims to be synced to rv=101
and the other claims to be synced to rv=102.
**This may result in resuming watches across api-servers to suffer from
"too old resource version" errors in a steady state.**

From the first glance it may look like having many kube-apiservers can
mitigate this problem. However, if no object of type X is being changed
for extended period of time, this doesn't help.
We've seen large production clusters with tens of thousands of pods,
where no pod was changing for extended periods of time (e.g. tens of
minutes).

The goal of this proposal is to avoid both of these problems.

### Goals

- avoid tons of relists during kube-apiservers rolling upgrades
- avoid different instances of kube-apiserver stuck with watchcache synced
  to different resource versions for extended period of time

### Non-Goals

- allow consistent reads from kube-apiserver cache (this proposal makes
  it easier but it's not the goal to solve it)

## Proposal

We propose to utilize the 'progress notify' feature from etcd to solve the
problem.

Since version 3.0, etcd watch can be configured with `WithProgressNotify`
enabled. In that case, every N (hard-coded to 10 minutes in the code) etcd
checks if any event was send to the watcher within that interval and if not
sends a special progress notify containing the current etcd resource version
is send to the watcher.

We are going to utilize this feature to solve the problems described above.

1. We will work with etcd team to make the interval configurable. It should
   be fairly simple - [POC](https://github.com/etcd-io/etcd/pull/11463)

1. We modify all watches used to propagate watchcache to set the
   `WithProgressNotify` option. For watches being served from etcd in case
   of disabled watchcache, this should remain unchanged. We can also consider
   automatically translating to bookmarks if the overhead won't be too large.

1. We will modify the kubernetes, so that it can understand progress notify
   events and use them to update the so-far resource version. This is fairly
   simple given the already existing support for watch Bookmark events.
   The only change that will be exposed in the client-go libraries will be
   the change to reflector so that it can update underlying store resource
   version based on incoming Bookmark event. Basically instead of current:

```golang
	...
	case watch.Bookmark:
		// A `Bookmark` means watch has synced here, just update the resourceVersion
	default:
	...
```

   we will modify it to:

```golang
	...
	case watch.Bookmark:
		// A `Bookmark` means watch has synced here, just update the resourceVersion
+		if rvu, ok := r.store.(resourceVersionUpdater); ok {
+			rvu.UpdateResourceVersion(newResourceVersion)
+		}
	default:
	...
```

   where resourceVersionUpdate is a simple interface implementing just
   `UpdateResourceVersion(resourceVersion string)` function.

1. Change watch cache to utilize the resource version updates from Bookmark
   events.

1. On top of recent changes that send Kubernetes Bookmark events every minute,
   we will add a support to send them also on kube-apiserver shutdown.

1. We will set the progress notify period to reasonably small value. 
   The requirement is to ensure that in case of rolling upgrade of multiple
   kube-apiservers, the next-to-be-updated one will get either a real event
   or a progress notify one with a version at least as fresh as the version
   to which the just-upgraded one was initialized (based on that version it
   can then send bookmarks to its watchers on shutdown).
   Given we're going to send bookmarks on shutdown, we can wait for some
   short period until progress notify event come, so values of [1s, 10s]
   seem reasonable (we can also set 1s frequency and wait up to 10s for the
   delivery on shutdown to tolerate delays).
   In the past, it was successfully scale tested up until 250ms - see
   https://github.com/kubernetes/kubernetes/pull/86769#issuecomment-579171765
   so performance/scalability shouldn't be a problem.

Note that if due to some races/issues single ProgressNotify event won't be
delievered to a subset of kube-apiservers, this is not a problem, because:
- generally subsequent watch will be send to the same kube-apiserver due to
  http/http2 connection stickiness in Golang library (unless there is no
  disruption, which shouldn't be frequent situation)
- even if it wouldn't be the case, that can only cause issues on watchcache
  initialization; once all of them are initialized then (a) if watch is
  broken in newer kube-apiserver and reconnects to the older one it is fine
  because watch can be started with future resource version (b) if watch
  is broken on older kube-apiserver and reconnects to the newer one, we
  won't delete change history on ProgressNotify, so that older version
  should still be stored there (unless there is heavy churn of those objects
  which is the case that doesn't suffer from this problem)

The POC PR can be found in: https://github.com/kubernetes/kubernetes/pull/92472

### Risks and Mitigations

The biggest risk are bugs in the implementation. To mitigate this, the
implementation will be hidden behind `EfficientWatchResumption` feature
gate and necessary tests will be added and/or extended (details below).

## Design Details

### Test Plan

- unit tests for logic enhancing resource version tracking in reflector
- unit tests for newly added watch cache logic
- integration test for sending bookmark on kube-apiserver shutdown
- integration test for proving that resource version that
   kube-apiserver can serve from cache progresses eventually when objects of
   other types are being added/updated/deleted;
   this test should store events (or other type) in a separate etcd cluster
   (to test split-etcd backend mode) and ensure no RV leak across etcd clusters

### Graduation Criteria

Alpha should provide basic functionality covered with tests described above.

#### Alpha -> Beta Graduation

- Appropriate metrics are agreed on and implemented
- Ad-hoc manual rolling-upgrade of kube-apiservers in 5k-node cluster
   is not resulting in required re-listing for watched resources from
   node components

#### Beta -> GA Graduation

- Enabled in Beta for at least two releases without complaints
- Rolling-upgrade of kube-apiservers in 5k-node cluster test is
   automated and running periodically.

### Upgrade / Downgrade Strategy

Kubernetes can be safely updated/downgraded, as the implementation
is purely in memory:
- if etcd doesn't support frequent enough progress notify events,
   we won't get expected benefits (problems may not be addressed),
   but also no unexpected consequences
- enabling the feature may only result in additional watch bookmark
   events for clients, which they are explicitly opting-in anyway
- disabling the feature reverts the behavior of watchcache being
   synced to values of objects of different types; however given
   the initialization is happening at "now" anyway, the time won't
   go back

### Version Skew Strategy

n/a - watch bookmarks don't have any frequency guarantees

## Production Readiness Review Questionnaire

TODO: Fill in before making `Implementable`.

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: EfficientWatchResumption
    - Components depending on the feature gate: kube-apiserver

* **Does enabling the feature change any default behavior?**
  No.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes, watchcache (and watch bookmark events) will not be propagated with
  resource versions of objects of other types.

* **What happens if we reenable the feature if it was previously rolled back?**
  The expected behavior will be restored.

* **Are there any tests for feature enablement/disablement?**
  No.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

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


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
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

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

2020-06-30: KEP Proposed.
2020-08-04: KEP marked as implementable.

## Drawbacks

n/a

## Alternatives

### Initialize watch cache from etcd history window

The main alternative to the above solution would be to change the way how
watch cache is initialized to not only initialize the state but also a
transaction log (i.e. read the whole etcd history and initialize transaction
log based on it).

Pros:
- doesn't require any etcd changes
- no overhead when kube-apiserver is running (only initialization being more
   expensive)

Cons:
- given kube-apiserver is performing compaction (default every 5m), lack of
   changes of any object of type X within that period would result in inability
   to initialize transaction log anyway; so that is not universal solution
- minor: etcd API doesn't expose the last compaction revision that we should
   start syncing from
