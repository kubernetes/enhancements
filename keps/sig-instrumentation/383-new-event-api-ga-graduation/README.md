# New Event API GA Graduation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [Usability Issues With the Current API](#usability-issues-with-the-current-api)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
    - [Few Examples](#few-examples)
    - [Comparison Between Old and New APIs](#comparison-between-old-and-new-apis)
  - [Performance Improvements](#performance-improvements)
    - [Changes in EventRecorder](#changes-in-eventrecorder)
      - [Short Examples](#short-examples)
    - [Client Side Changes](#client-side-changes)
    - [Restarts](#restarts)
  - [Defence in Depth](#defence-in-depth)
    - [Aggressive Backoff](#aggressive-backoff)
  - [Other Related Changes](#other-related-changes)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Deprecated Fields](#deprecated-fields)
    - [Beta to GA Graduation](#beta-to-ga-graduation)
    - [Load Test](#load-test)
    - [Components Migration](#components-migration)
- [Considerations](#considerations)
  - [Performance Impact](#performance-impact)
  - [Backward Compatibility](#backward-compatibility)
  - [Sample Queries With &quot;New&quot; Events](#sample-queries-with-new-events)
    - [Get All NodeController Events](#get-all-nodecontroller-events)
    - [Get All Events From Lifetime of a Given Pod](#get-all-events-from-lifetime-of-a-given-pod)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Leaving Current Dedup Mechanism but Improve Backoff Behavior](#leaving-current-dedup-mechanism-but-improve-backoff-behavior)
  - [Timestamp List as a Dedup Mechanism](#timestamp-list-as-a-dedup-mechanism)
  - [Events as an Aggregated Object](#events-as-an-aggregated-object)
  - [Using New API Group for Storing Data](#using-new-api-group-for-storing-data)
  - [Pivoting Towards More Machine Readable Events by Introducing Stricter Structure](#pivoting-towards-more-machine-readable-events-by-introducing-stricter-structure)
  - [Pivoting Towards Making Events More Helpful for Cluster Operator During Debugging](#pivoting-towards-making-events-more-helpful-for-cluster-operator-during-debugging)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The KEP aims at fixing few issues in the current way Events are structured and implemented. This effort has two main goals - reduce the performance impact that Events have on the rest of the cluster and add more structure to the Event object which is the first and necessary step to make it possible to automate Event analysis.

Most sections and design details are copied from the original design doc: [Make Kubernetes Events Useful and Safe](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/instrumentation/events-redesign.md).

The new Event API has been promoted from v1beta1 to v1 in 1.19 and currently only scheduler was migrated to use it. This KEP proposes to graduate the new Event API by migrating all the remaining Kubernetes components to use the new Event API.

## Motivation

There's a relatively wide agreement that current implementation of Events in Kubernetes is problematic. Events are supposed to give app developer insight into what's happening with his/her app. Important requirement for Event library is that it shouldn't cause/worsen performance problems in the cluster.

The problem is that neither of those requirements are actually met. Currently Events are extremely spammy (e.g. Event is emitted when Pod is unable to schedule every few seconds) with unclear semantics (e.g. Reason was understood by developers as "reason for taking action" or "reason for emitting event"). Also there are well known performance problems caused by Events (e.g. [#47366](https://github.com/kubernetes/kubernetes/issues/47366), [#47899](https://github.com/kubernetes/kubernetes/issues/47899)) - Events can overload API server if there's something wrong with the cluster (e.g. some correlated crashloop on many Nodes, user created way more Pods than fit on the cluster which fail to schedule repeatedly). This was raised by the community on number of occasions.

### Goals

* Update Event semantics such that they'll be considered useful by app developers.
* Reduce impact that Events have on the system's performance and stability.

### Non-Goals

* Persist Events outside of etcd or for a longer time.

### Usability Issues With the Current API

Users would like to be able to use Events also for debugging and trace analysis of Kubernetes clusters. Current implementation makes it hard for the following reasons:
* 1s granularity of timestamps (system reacts much quicker than that, making it more or less unusable),
* deduplication, that leaves only count and, first and last timestamps (e.g. when Controller is creating a number of Pods information about it is deduplicated),
* `InvolvedObject`, `Message`, `Reason` and `Source` semantics are far from obvious. If we treat `Event` as a sentence object of this sentence is stored either in `Message` (if the subject is a Kubernetes object (e.g. Controller)), or in `InvolvedObject`, if the subject is some kind of a controller (e.g. Kubelet).
* hard to query for interesting series using standard tools (e.g. all Events mentioning given Pod is pretty much impossible because of deduplication logic),
* as semantic information is passed in the message, which in turn is ignored by the deduplication logic it is not clear that this mechanism will not cause deduplication of Events that are completely different.

## Proposal

The new Event API makes all semantic information about events first-class fields, allowing better deduplication and querying. The API changes aim to:
* make it easy to list all interesting Events in common scenarios using kubectl:
  * Listing Events mentioning given Pod,
  * Listing Events emitted by a given component (e.g. Kubelet on a given machine, NodeController),
* make timestamps precise enough to allow better events correlation,
* update the field names to better indicate their function.

In addition, we will improve 'Event series' detection and send only 'series start' and 'series finish' Events, and add more aggressive backoff policy for Events.

After the migration is done, users and administrators:
* will be able to better track interesting changes in the state of objects they're interested in,
* will be convinced that Events do not destabilize.

## Design Details

Note that the design details talk about all aspects of the whole Event redesign effort, including API changes where the implementation has been done in API group v1beta1.

### API Changes

We'd like to propose following structure in Events object in the new events API group:

```golang
// Event is a report of an event somewhere in the cluster. It generally denotes some state change in the system.
type Event struct {
  metav1.TypeMeta `json:",inline"`
  // +optional
  metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

  // Required. Time when this Event was first observed.
  EventTime metav1.MicroTime `json:"eventTime" protobuf:"bytes,2,opt,name=eventTime"`

  // Data about the Event series this event represents or nil if it's a singleton Event.
  // +optional
  Series *EventSeries `json:"series,omitempty" protobuf:"bytes,3,opt,name=series"`

  // Name of the controller that emitted this Event, e.g. `kubernetes.io/kubelet`.
  // +optional
  ReportingController string `json:"reportingController,omitempty" protobuf:"bytes,4,opt,name=reportingController"`

  // ID of the controller instance, e.g. `kubelet-xyzf`.
  // +optional
  ReportingInstance string `json:"reportingInstance,omitempty" protobuf:"bytes,5,opt,name=reportingInstance"`

  // What action was taken/failed regarding to the regarding object.
  // +optional
  Action string `json:"action,omitempty" protobuf:"bytes,6,name=action"`

  // Why the action was taken.
  Reason string `json:"reason,omitempty" protobuf:"bytes,7,name=reason"`

  // The object this Event is about. In most cases it's an Object reporting controller implements.
  // E.g. ReplicaSetController implements ReplicaSets and this event is emitted because
  // it acts on some changes in a ReplicaSet object.
  // +optional
  Regarding corev1.ObjectReference `json:"regarding,omitempty" protobuf:"bytes,8,opt,name=regarding"`

  // Optional secondary object for more complex actions. E.g. when regarding object triggers
  // a creation or deletion of related object.
  // +optional
  Related *corev1.ObjectReference `json:"related,omitempty" protobuf:"bytes,9,opt,name=related"`

  // Optional. A human-readable description of the status of this operation.
  // Maximal length of the note is 1kB, but libraries should be prepared to
  // handle values up to 64kB.
  // +optional
  Note string `json:"note,omitempty" protobuf:"bytes,10,opt,name=note"`

  // Type of this event (Normal, Warning), new types could be added in the
  // future.
  // +optional
  Type string `json:"type,omitempty" protobuf:"bytes,11,opt,name=type"`

  // Deprecated field assuring backward compatibility with core.v1 Event type
  // +optional
  DeprecatedSource corev1.EventSource `json:"deprecatedSource,omitempty" protobuf:"bytes,12,opt,name=deprecatedSource"`
  // Deprecated field assuring backward compatibility with core.v1 Event type
  // +optional
  DeprecatedFirstTimestamp metav1.Time `json:"deprecatedFirstTimestamp,omitempty" protobuf:"bytes,13,opt,name=deprecatedFirstTimestamp"`
  // Deprecated field assuring backward compatibility with core.v1 Event type
  // +optional
  DeprecatedLastTimestamp metav1.Time `json:"deprecatedLastTimestamp,omitempty" protobuf:"bytes,14,opt,name=deprecatedLastTimestamp"`
  // Deprecated field assuring backward compatibility with core.v1 Event type
  // +optional
  DeprecatedCount int32 `json:"deprecatedCount,omitempty" protobuf:"varint,15,opt,name=deprecatedCount"`
}

// EventSeries contain information on series of events, i.e. thing that was/is happening
// continuously for some time.
type EventSeries struct {
  // Number of occurrences in this series up to the last heartbeat time
  Count int32 `json:"count" protobuf:"varint,1,opt,name=count"`
  // Time when last Event from the series was seen before last heartbeat.
  LastObservedTime metav1.MicroTime `json:"lastObservedTime" protobuf:"bytes,2,opt,name=lastObservedTime"`
  // Information whether this series is ongoing or finished.
  // Deprecated. Planned removal for 1.18
  State EventSeriesState `json:"state" protobuf:"bytes,3,opt,name=state"`
}

type EventSeriesState string

const (
  EventSeriesStateOngoing  EventSeriesState = "Ongoing"
  EventSeriesStateFinished EventSeriesState = "Finished"
  EventSeriesStateUnknown  EventSeriesState = "Unknown"
)
```

#### Few Examples

| Regarding | Action | Reason | ReportingController | Related | 
| ----------| -------| -------| --------------------|---------| 
| Node X | BecameUnreachable | HeartbeatTooOld | kubernetes.io/node-ctrl | <nil> |
| PVC X | FailedToAttachVolume | Unknown | kubernetes.io/pv-attach-ctrl | Node Y |
| ReplicaSet X | FailedToInstantiatePod | QuotaExceeded | kubernetes.io/replica-set-ctrl | <nil> |
| ReplicaSet X | InstantiatedPod | | kubernetes.io/replica-set-ctrl | Pod Y |
| Ingress X | CreatedLoadBalancer | | kubernetes.io/ingress-ctrl | <nil> |
| Pod X | ScheduledOn | | kubernetes.io/scheduler | Node Y |
| Pod X | FailedToSchedule | FitResourcesPredicateFailed | kubernetes.io/scheduler | <nil> |

#### Comparison Between Old and New APIs

| Old         | New         |
|-------------|-------------|
| Old Event { | New Event { |
| TypeMeta | TypeMeta |
| ObjectMeta | ObjectMeta |
| InvolvedObject ObjectReference | Regarding ObjectReference |
|  | Related *ObjectReference |
|  | Action string |
| Reason string | Reason string |
| Message string | Note string |
| Source EventSource |  |
|  | ReportingController string |
|  | ReportingInstance string |
| FirstTimestamp metav1.Time | |
| LastTimestamp metav1.Time | |
|  | EventTime metav1.MicroTime |
| Count int32 | |
|  | Series EventSeries |
| Type string | Type string |
| } | } |

Namespace in which Event will live will be equal to
- Namespace of Regarding object, if it's namespaced,
- NamespaceSystem, if it's not.

Note that this means that if Event has both Regarding and Related objects, and only one of them is namespaced, it should be used as Regarding object.

The biggest change is the semantics of the Event object in case of loops. If Series is nil it means that Event is a singleton, i.e. it happened only once and the semantics is exactly the same as currently in Events with `count = 1`. If Series is not nil it means that the Event is either beginning or the end of an Event series - equivalence of current Events with `count > 1`. Events for ongoing series have Series.State set to EventSeriesStateOngoing, while endings have Series.State set to EventSeriesStateFinished (Series.State field has been deprecated and will be removed before GA graduation).

This change is better described in the section below.


### Performance Improvements

We want to replace current behavior, where EventRecorder patches Event object every time when deduplicated Event occurs with an approach where being in the loop is treated as a state, hence Events only should be updated only when system enters or exits loop state (or is a singleton Event).

Because Event object TTL in etcd we can't have above implemented cleanly, as we need to update Event objects periodically to prevent etcd garbage collection from removing ongoing series. We can use this need to update users with new data about number of occurrences.

The assumption we make for deduplication logic after API changes is that Events with the same <Regarding, Action, Reason, ReportingController, ReportingInstance, Related> tuples are considered isomorphic. This allows us to define notion of "event series", which is series of isomorphic events happening not farther away from each other than some defined threshold. E.g. Events happening every second are considered a series, but Events happening every hour are not.

The main goal of this change is to limit number of API requests sent to the API server to the minimum. This is important as overloading the API server can severely impact usability of the system.

In the absence of errors in the system (all Pods are happily running/starting, Nodes are healthy, etc.) the number of Events is easily manageable by the system. This means that it's enough to concentrate on erroneous states and limit number of Events published when something's wrong with the cluster.

There are two cases to consider: Event series, which result in ~1 API call per ~30 minutes, so won't cause a problem until there's a huge number of them; and huge number of non-series Events. To improve the latter we require that no high-cardinality data are put into any of Regarding, Action, Reason, ReportingController, ReportingInstance, Related fields. Which bound the number of Events to O(number of objects in the system^2). For the present we don't have any automatic way to ensure this, so we will rely on manual inspection and review.

#### Changes in EventRecorder

EventRecorder is our client library for Events that are used in components to emit Events. The main function in this library is `Eventf`, which takes the data and passes it to the EventRecorder backend, which does deduplication and forwards it to the API server.

We need to write completely new deduplication logic for new Events, preserving the old one to avoid necessity to rewrite all places when Events are used together with this change. Additionally we need to add a new `Eventf`-equivalent function to the interface that will handle creation of new kind of events.

New deduplication logic will work in the following way:
- When event is emitted for the first time it's written to the API server without series field set.
- When isomorphic event is emitted within the threshold from the original one EventRecorder detects the start of the series, updates the Event object, with the Series field set carrying count. In the EventRecorder it also creates an entry in `activeSeries` map with the timestamp of last observed Event in the series.
- All subsequent isomorphic Events don't result in any API calls, they only update last observed timestamp value and count in the EventRecorder.
- For all active series every 30 minutes EventRecorder will create a "heartbeat" call. Goal of this update is to periodically update user on number of occurrences and prevent garbage collection in etcd. The heartbeat will be an Event update that updates the count and last observed time fields in the series field.
- For all active series every 6 minutes (longer than the longest backoff period) EventRecorder will check if it noticed any attempts to emit isomorphic Event. If there were, it'll check again after aforementioned period (6 minutes). If there weren't it assumes that series is finished and emits closing Event call. This updates the Event by updating the count and last observed time fields in the series field.

##### Short Examples

After first occurrence, Event looks like:
```
{
  regarding: A,
  action: B,
  reportingController: C,
  ...,
}
```
After second occurrence, Event looks like:
```
{
  regarding: A,
  action: B,
  reportingController: C,
  ...,
  series: {count: 2},
}
```
After half an hour of crashlooping, Event looks like:
```
{
  regarding: A,
  action: B,
  reportingController: C,
  ...,
  series: {count: 4242},
}
```
Minute after crashloop stopped, Event looks like:
```
{
  regarding: A,
  action: B,
  reportingController: C,
  ...,
  series: {count: 424242},
}
```

#### Client Side Changes

All clients will need to eventually migrate to use new Events, but no other actions are required from them. Deduplication logic change will be completely transparent after the move to the new API.

#### Restarts

We don't take specific actions for this case, since:
- if an Event already ended, it will hang for another hour and will be GC-ed because of TTL;
- if it didn't end, we will update it after restart anyway

### Defence in Depth

Because Events proved problematic we want to add multiple levels of protection in the client library to reduce chances that Events will be overloading API servers in the future. We propose to do the following thing.

#### Aggressive Backoff

We need to make sure that kubernetes client used by EventRecorder uses properly configured and backoff pretty aggressively. Events should not destabilize the cluster, so if EventRecorder receives 429 response it should exponentially back off for non-negligible amount of time, to let API server recover.

### Other Related Changes

To allow easier querying we need to make following fields selectable for Events:
- event.reportingController
- event.reportingInstance
- event.action
- event.reason
- event.regarding...
- event.related...
- event.type

Kubectl will need to be updated to use new Events if present.

### Test Plan

Correctness:

- Update all unit tests that are using Event lib to use the new API and make sure they all pass.
- Manually run tests with both healthy and crash-looping clusters that keep generating Events to ensure they are produced in an expected way.

Scalability and Performance:

- Run scale tests with the pause pods to be "sleep 5; exit 1" pods.
- Record the memory usage in both healthy and crash-looping clusters of various size (e.g., 50, 500, 5k nodes).
- Ensure Event increase can be handled by etcd and there's no big performance reduction.

### Graduation Criteria

The new Event API is in v1 right now and scheduler has been migrated to use new API. The plan is to do a load test and migrate all the remaining components.

#### Deprecated Fields

This section lists the deprecated Event API fields that should be removed before graduating to GA.

- State field of EventSeries (planned removal for 1.18)
  - This field should also be removed from `corev1.Event` API: [#75987](https://github.com/kubernetes/kubernetes/pull/75987)

#### Beta to GA Graduation

- Remove deprecated fields listed above
- Gather data from performance and scalability tests
- Remove all in-tree use of the core/v1.Event API in favour of events.k8s.io/v1

#### Load Test

The idea is to use [ClusterLoader](https://github.com/kubernetes/perf-tests/tree/master/clusterloader2) testing framework to do a load test to make sure events generated can be handled by etcd and there's no big performance reduction.

#### Components Migration

The list of components that need to be migrated is shown as follows:

- kubelet
- cloud-controller-manager
- kube-controller-manager
- leader election
- node problem detector
- gce ingress controller
- event exporter

(Note: there are more out-of-tree components that need to be migrated)


## Considerations

### Performance Impact

We're not changing how Events are stored in the etcd (except adding new fields to the storage type). We'll keep current TTL for all Event objects.

Proposed changes alone will have possibly three effects on performance: we will emit more Events for Pod creation (disable deduplication for "Create Pod" Event emitted by controllers), we will emit fewer Events for hotloops (3 API calls + 1 call/30min per hotloop series, instead of 1/iteration), and Events will be bigger. This means that Event memory footprint will grow slightly, but in the unhealthy looping state number of API calls will be reduced significantly.

We looked at the amount of memory used in our performance tests in cluster of various size. The results are following:

| | 5 nodes | 100 nodes | 500 nodes | 5000 nodes |
|-|---------|-----------|-----------|------------|
| event-etcd | 28MB | 65MB | 161MB | n/a |
| All master component | 530MB | 1,2GB | 3,9GB | n/a |
| Excess resources in default config | 3,22GB | 13,8GB | 56,1GB | n/a |

The difference in size of the Event object comes from new Action and Related fields. We can safely estimate the increase to be smaller than 30%. We'll also emit additional Event per Pod creation, as currently Events for that are being deduplicated. There are currently at least 6 Events emitted when Pod is started, so impact of this change can be bounded by 20%. This means that in the worst case the increase in Event size can be bounded by 56%. As can be seen in the table above we can easily afford such increase.

### Backward Compatibility

Kubernetes API machinery moves towards moving all resources for which it make sense to separate API groups e.g. to allow defining separate storage for it. For this reason we're going to create a new `events` API group in which Event resources will live.

In the same time we can't stop emitting v1.Events from the Core group as this is considered breaking API change. For this reason we decided to create a new API group for events but map it to the same internal type as core Events.

As objects are stored in the versioned format we need to add new fields to the Core group, as we're going to use Core group as storage format for new Events.

After the change we'll have three types of Event objects. Internal representation (denoted internal), "old" core API group type (denoted core) and "new" events API group (denoted events). They will look in the following way - green color denotes added fields:

| internal.Event | core.Event | events.Event |
|----------------|------------|--------------|
| TypeMeta | TypeMeta | TypeMeta |
| ObjectMeta | ObjectMeta | ObjectMeta |
| InvolvedObject ObjectReference | InvolvedObject ObjectReference | Regarding ObjectReference |
| Related \*ObjectReference | Related \*ObjectReference | Related \*ObjectReference |
| Action string | Action string | Action string |
| Reason string | Reason string | Reason string |
| Message string | Message string | Note string |
| Source.Component string | Source.Component string | ReportingController string |
| Source.Host string | Source.Host string | DeprecatedHost string |
| ReportingInstance string | ReportingInstance string | ReportingInstance string |
| FirstTimestamp metav1.Time | FirstTimestamp metav1.Time | DeprecatedFirstTimestamp metav1.Time |
| LastTimestamp metav1.Time | LastTimestamp metav1.Time | DeprecatedLastTimestamp metav1.Time |
| EventTime metav1.MicroTime | EventTime metav1.MicroTime | EventTime metav1.MicroTime |
| Count int32 | Count int32 | DeprecatedCount int32 |
| Series.Count int32 | Series.Count int32 | Series.Count int32 |
| Series.LastObservedTime | Series.LastObservedTime | Series.LastObservedTime |
| Series.State string | Series.State string | Series.State string |
| Type string | Type string | Type string |

Considered alternative was to create a separate type that will hold all additional fields in core.Event type. It was dropped, as it's not clear it would help with the clarity of the API.

There will be conversion functions that'll allow reading/writing Events as both core.Event and events.Event types. As we don't want to officially extend core.Event type, new fields will be set only if Event would be written through events.Event endpoint (e.g. if Event will be created by core.Event endpoint EventTime won't be set).

This solution gives us clean(-ish) events.Event API and possibility to implement separate storage for Events in the future. The cost is adding more fields to core.Event type. We think that this is not a big price to pay, as the general direction would be to use separate API groups more and core group less in the future.

`Events` API group will be added directly as beta API, as otherwise kubernetes component's wouldn't be allowed to use it.

### Sample Queries With "New" Events

#### Get All NodeController Events

List Events from the NamespaceSystem with field selector `reportingController = "kubernetes.io/node-controller"`

#### Get All Events From Lifetime of a Given Pod

List all Event with field selector `regarding.name = podName, regarding.namespace = podNamespace`, and `related.name = podName, related.namespace = podNamespace`. You need to join results outside of the kubernetes API.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
  - [x] Other
    - Describe the mechanism:

      (1) The API itself can be enabled / disabled at kube-apiserver level
      by using `--runtime-config` flag;

      (2) For the use of API, we have a fallback mechanism instead of using
      a feature gate. That is, we simply fallback to the old Event libraries
      if the API is diabled.

      Currently this fallback is implemented purely in scheduler but we're
      planning to move it into the library itself.

    - Will enabling / disabling the feature require downtime of the control
      plane?

      Yes, enabling / disabling API requires to restart apiserver as well as
      the components using that.

    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

      No.

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.

  While the graduation of the API itself doesn't change default behavior,
  migration of individual components does, as the events will be reported
  differently.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  Also set `rollback-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g. if this is runtime
  feature, can it break the existing applications?).

  Yes. If the new Event API is disabled, it will fallback to the original one 
  (The new events are roundtrippable with the old `corev1.Events`).

  If individual components don't implement it, rollback of client-library use
  may not be possible (i.e. they only fallback to the old API if the new API
  is disabled, if there is bug in the client-library, there is no way to
  fallback as of now).

* **What happens if we reenable the feature if it was previously rolled back?**

  New types of Events will be generated instead of the old one.

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling and disabling feature
  gates. However, unit tests in each component dealing with managing data created
  with and without the feature are necessary. At the very least, think about
  conversion tests if API types are being modified.

  Manual tests will be performed to ensure things work when either enabling
  or disabling the new Event API.

  More information in [Test Plan](#test-plan) section.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g. what if some components will restart
  in the middle of rollout?

  The rollout of API won't affect already running workloads.
  The rollout of the API migration of individual components may cause components
  fail to initialize in case of bugs. It will not affect running workloads though.

* **What specific metrics should inform a rollback?**

  [apiserver_request_total](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/endpoints/metrics/metrics.go#L66):
  If the value is much higher or lower than expected, it might be due to bugs
  in the library.

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and do that now.

  Not yet. It could be done by enabling / disabling new Event API.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

  State field of EventSeries will be removed from corev1.Event API.

### Monitoring requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metrics. Operations against Kubernetes API (e.g.
  checking if there are objects with field X set) may be last resort. Avoid
  logs or events for this purpose.

  The API, as a feature that workloads may in theory use,
  can be determined by looking into the apiserver_requests_total metric.

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  - [x] Metrics
    - Metric name: apiserver_requests_total
    - Components exposing the metric: kube-apiserver
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At the high-level this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide a comprehensive guidance, but at the very
  high level (they needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

  Events have always been "best-effort".
  We're sticking to that with the new API too, so no SLO will be introduced.

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**
  Describe the metrics themselves and the reason they weren't added (e.g. cost,
  implementation difficulties, etc.).

  No.

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  Think about both cluster-level services (e.g. metrics-server) as well
  as node-level agents (e.g. specific version of CRI). Focus on external or
  optional services that are needed. For example, if this feature depends on
  a cloud provider API, or upon an external software-defined storage or network
  control plane.

  For each of the fill in the following, thinking both about running user workloads
  and creating new ones, as well as about cluster-level services (e.g. DNS):
  
  N/A


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirms the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:

  In the new EventRecorder, every 30 minutes a "heartbeat" call will be performed
  to update Event status and prevent garbage collection in etcd. This heartbeat
  is happening for events that are happening periodically (If an event didn't
  happen for 6 minutes, it will be GC-ed).

* **Will enabling / using this feature result in introducing new API types?**

  Yes, a new API type "eventsv1.Event" is being introduced.
  The number of Event objects depends on cluster state and its churn. Event
  deduplication and reasonable cardinality of the fields should keep their
  number within reasonable boundaries (obviously dependent on cluster size).

* **Will enabling / using this feature result in any new calls to cloud
  provider?**

  No.

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**
  Describe them providing:
  
  The difference in size of the Event object comes from new Action and Related
  fields. We can safely estimate the increase to be smaller than 30%. However,
  more events may be emitted. For example, new Events will be emitted for Pod
  creation done by standard controllers (e.g. ReplicaSet), as they are currently
  deduplicated across all 'owner' objects. However, given that that are at least
  5 other events being emitted during pod startup, the impact for it can be
  bounded by 20%. In total, we estimated that increase in total size of all
  Events can be conservatively bounded by around 50%, but practical boundary
  should be much smaller.

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  
  No

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  
  The potential increase of Event size might cause non-negligible increase of
  storage in Etcd, network bandwidth to send them, and CPU to process them.

### Troubleshooting

Troubleshooting section serves the `Playbook` role as of now. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now we leave it here though.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

  The Events will be dropped if API server or etcd is unavailable.

* **What are other known failure modes?**
  
  N/A

* **What steps should be taken if SLOs are not being met to determine the problem?**

  N/A

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2017-10-7 design proposal merged under [kubernetes/community](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/instrumentation/events-redesign.md)
- 2017-11-23 Event API group is [merged](https://github.com/kubernetes/kubernetes/pull/49112)
- New Event API [staging/src/k8s.io/client-go/kubernetes/typed/events/v1beta1](https://github.com/kubernetes/kubernetes/tree/master/staging/src/k8s.io/client-go/kubernetes/typed/events/v1beta1)
- Scheduler migration: [#78447](https://github.com/kubernetes/kubernetes/pull/78447/files), [#83692](https://github.com/kubernetes/kubernetes/pull/83692)

New Events API has been graduated to GA in 1.19:
- Promoted new Events API from v1beta1 to v1: [#91645](https://github.com/kubernetes/kubernetes/pull/91645), [#92662](https://github.com/kubernetes/kubernetes/pull/92662), [#92874](https://github.com/kubernetes/kubernetes/pull/92874)
- Implemented a common fallback library for Events API: [#91798](https://github.com/kubernetes/kubernetes/pull/91798), [92082](https://github.com/kubernetes/kubernetes/pull/92082)
- Implemented CRUD tests for new Events API: [#92607](https://github.com/kubernetes/kubernetes/pull/92607), [#92724](https://github.com/kubernetes/kubernetes/pull/92724), [#92755](https://github.com/kubernetes/kubernetes/pull/92755), [#93296](https://github.com/kubernetes/kubernetes/pull/93296)

## Alternatives

### Leaving Current Dedup Mechanism but Improve Backoff Behavior

As we're going to move all semantic information to fields, instead of passing some of them in message, we could just call it a day, and leave the deduplication logic as is. When doing that we'd need to depend on the client-recorder library on protecting API server, by using number of techniques, like batching, aggressive backing off and allowing admin to reduce number of Events emitted by the system. This solution wouldn't drastically reduce number of API requests and we'd need to hope that small incremental changes would be enough.

### Timestamp List as a Dedup Mechanism

Another considered solution was to store timestamps of Events explicitly instead of only count. This gives users more information, as people complain that current dedup logic is too strong and it's hard to "decompress" Event if needed. This change has clearly worse performance characteristic, but fixes the problem of "decompressing" Events and generally making deduplication lossless. We believe that individual repeated events are not interesting per se, what's interesting is when given series started and when it finished, which is how we ended with the current proposal.

### Events as an Aggregated Object

We considered adding nested information about occurrences into the Event. In other words we'd have single Event object per Subject and instead of having only `Count`, we could have stored slice of `timestamp-object` pairs, as a slightly heavier deduplication information. This would have non-negligible impact on size of the event-etcd, and additional price for it would be much harder query logic (querying nested slices is currently not implemented in kubernetes API), e.g. "Give me all Events that refer Pod X" would be hard.

### Using New API Group for Storing Data

Instead of adding "new" fields to the "old" versioned type, we could have change the version in which we store Events to the new group and use annotations to store "deprecated" fields. This would allow us to avoid having "hybrid" type, as `v1.Events` became, but the change would have a much higher risk (we would have been moving battle-tested and simple `v1.Event` store to new `events.Event` store with some of the data present only in annotations). Additionally performance would degrade, as we'd need to parse JSONs from annotations to get values for "old" fields.
Adding panic button that would stop creation/update of Events
If all other prevention mechanism fail we’d like a way for cluster admin to disable Events in the cluster, to stop them overloading the server. However, we dropped this idea, as it's currently possible to achieve the similar result by changing RBAC rules.

### Pivoting Towards More Machine Readable Events by Introducing Stricter Structure

We considered making easier for automated systems to use Events by enforcing "active voice" for Event objects. This would allow us to assure which field in the Event points to the active component, and which to direct and indirect objects. We dropped this idea because Events are supposed to be consumed only by humans.

### Pivoting Towards Making Events More Helpful for Cluster Operator During Debugging

We considered exposing more data that cluster operator would need to use Events for debugging, e.g. making ReportingController more central to the semantics of Event and adding some way to easily grep though the logs of appropriate component when looking for context of a given Event. This idea was dropped because Events are supposed to give application developer who's running his application on the cluster a rough understanding what was happening with his app.