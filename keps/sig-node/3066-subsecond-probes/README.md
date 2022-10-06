# KEP-3066: Subsecond Probes

Probe timeouts are limited to seconds and that does NOT work well for clients looking for finer and coarser grained timeouts.
## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
    - [Knative](#knative)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Knative](#knative-1)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Overriding an existing field](#overriding-an-existing-field)
- [Design Details](#design-details)
  - [Existing Struct](#existing-struct)
  - [Existing Behavior](#existing-behavior)
    - [Defaulting logic](#defaulting-logic)
    - [Validation of fields](#validation-of-fields)
    - [Fields to Add](#fields-to-add)
    - [Logic for Added Fields](#logic-for-added-fields)
  - [Existing use of Probe struct fields.](#existing-use-of-probe-struct-fields)
    - [InitialDelaySeconds](#initialdelayseconds)
    - [TimeoutSeconds](#timeoutseconds)
    - [ProbeSeconds](#probeseconds)
  - [Summary](#summary-1)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
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
  - [<code>early*Offset</code>](#)
  - [<code>*Duration</code>](#-1)
  - [A new struct, <code>ProbeOffset</code>](#a-new-struct-)
  - [Setting the time units in a different field, <code>ReadSecondsAs</code>](#setting-the-time-units-in-a-different-field-)
  - [v2 api for probe.](#v2-api-for-probe)
  - [OffsetMilliseconds](#offsetmilliseconds)
  - [Reconcile seconds field to nearest whole second.](#reconcile-seconds-field-to-nearest-whole-second)
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The Probe struct contains fields that specify seconds for intervals.
Some users would like to have intervals less than or slightly greater
than one second.

## Motivation

Better performance in the form of faster startups, more timely readiness checks...

#### Knative

For example, Knative will create Pods (via Deployment) and wait for them to become `Ready`
when handling an HTTP request from an end-user. In this case, Pod readiness
latency has a direct impact on HTTP request latency. (In the steady state,
Knative will re-use an existing Pod, but this situation can happen on scale-up,
and is guaranteed to happen on scale-from-zero.)

### Goals

An ability to specify probe durations that have more granularity, more specificity, than just zero,
one, or n-seconds. Focus is on possibly faster readiness probes, for example every .5 seconds until succss
and more granular use beyond existing defaults, for example 1.5 second duration instead of picking
between 1 or 2 second duration.
Add additional tests cases to the timeout test cases.
Not breaking backwards compatibility with the V1 API.


### Non-Goals

V2 API for existing objects.
Converting fields from `int32` to `resource.Quantity`.


## Proposal

TLDR:
Add two new optional fields (of type `*int32`) to the Probe struct, which would be used to offset the second based time values in the Probe struct:

- `PeriodMilliseconds`       // How often (in milliseconds) to offset PeriodSeconds when performing the probe (*** to reduce un-necessary resource usage, when periodMilliseconds is used to reduce the period to less than a second the offset is only used until success is reached then no longer).
- `InitialDelayMilliseconds` // Length of time (in milliseconds) to offset IntialDelaySeconds before health checking is activated.

The seconds and milliseconds fields (as related) will be summed to get the resulting time duration. For example,

```yaml
...
#yaml
periodSeconds: 1
periodMilliseconds: 500
...
```

would be considered a period value of 1.5s or 1500ms. These values are used as `time.Duration` within the prober package ([ex1](https://github.com/kubernetes/kubernetes/blob/a750d8054a6cb3167f495829ce3e77ab0ccca48e/pkg/kubelet/prober/prober.go#L161) [ex2](https://github.com/kubernetes/kubernetes/blob/a750d8054a6cb3167f495829ce3e77ab0ccca48e/pkg/kubelet/prober/worker.go#L132) ), so there's no real difference between 1.5s and 1500ms.

A few additional examples:

`periodSeconds: 2` and `periodMilliseconds: -500` would be 1.5s / 1500ms.

`periodSeconds: 1` and `periodMilliseconds: -500` would be 0.5s / 500ms. (*** To reduce un-necessary resource usage, because periodMilliseconds is used to reduce the period to less than a second the offset is only used until success is reached then no longer becoming 1second after success.)

More generally, the effective period value would be = `periodSeconds` + `periodMilliseconds`.

There are a few corner cases around default (effective period) values:

`periodSeconds: 0` and `periodMilliseconds: 500` would be 10.5s / 10500ms (as 0 => 10 for `periodSeconds` via [defaults.go](https://github.com/kubernetes/kubernetes/blob/3b13e9445a3bf86c94781c898f224e6690399178/pkg/apis/core/v1/defaults.go#L213-L215))

Each optional millisecond field will be restricted to [-999,999], and the effective sum will never be allowed
to be less than 0 (if the effective sum is less than 0, it will fail at the validation stage and block deployments).


### User Stories (Optional)

#### Knative

See the discussion from the Sept 14, 2021 SIG-Node meeting
[recording](https://youtu.be/LMh7c9e7H-Q?list=PL69nYSiGNLP1wJPj5DYWXjiArF-MJ5fNG&t=1049)
[meeting notes](https://docs.google.com/document/d/1Ne57gvidMEWXR70OxxnRkYquAoMpt56o75oZtg-OeBg/edit#heading=h.by9uk7onna00)

### Notes/Constraints/Caveats (Optional)

Changing defaults is a strict no-go.

### Risks and Mitigations

Accidentally setting a timeout too low could DOS kubelet if many are used.
Thus will mitigate by preventing timeout values that are too small or to often:
exec(500ms minimum periodSeconds until success, then drop back to the original second range/value)
grpc(200ms minimum until success, then drop back to the original second range)
http(200ms minimum until success, then drop back to the original second range)

note for exec, grpc, http .. for startup and readiness we will support the millisecond range sum to be greater than zero..
in other words if they know 1.4 seconds is the right range and 1s just will not work they can ask for 1second plus 400mills
vs having to go pick one of 1second where the first is known to fail or 2 where they know it's 600mills slower than it should
be because 99% of runs will be less than 1.4s.

Keep initialDelayMilliseconds where initialDelay is supported because zero is the current minimum initialDelay in seconds..
thus no mitigation is necessary.

The periodMilliseconds option for periodSeconds is mitigated to only being valid until success and with appropriate minimums
specified above

timeoutMilliseconds will not be implemented in alpha.. may investigate for possible inclusion in a subsequent KEP.

Benchmarking currently WIP for alpha, will be implemented for beta to determine the ideal floor values. Several benchmarking tools are being considered (options include ones developed by Knative, Amazon, Red Hat). Benchmarking tooling will also work for similar areas (such as the PLEG work.)

As @aojea notes sig-scalability is already measuring pod latency on startup,
http://perf-dash.k8s.io/#/?jobname=gce-100Nodes-master&metriccategoryname=E2E&metricname=LoadHighThroughputPodStartup&Metric=pod_startup and adding a new metric
to https://github.com/kubernetes/perf-tests may suffice.

Question raised of how probes should be charged: charging is an issue for all exec/attach/port forward requests, and as usual the process in the container should be charged to the container/pod. This KEP isn't looking to make any major architectural changes in this regard.


#### Overriding an existing field

However, this change would be done on a probe by probe basis and would be opt-in. Meaning that the default behavior will not change unless a user specifically added the millisecond fields to their spec. Absent that specific opt-in by using the new fields, existing users not being impacted by this change (gate on/off) will be tested.

## Design Details

Draft PR:
https://github.com/kubernetes/kubernetes/pull/107958/commits

A quick highlighting of key points:

* Implementing a feature gate (`SubSecondProbes`) to gate the feature

* Adding three additional, optional fields to the Probe struct type:

```go
PeriodMilliseconds *int32 //note these are signed offset
InitialDelayMilliseconds *int32
TimeoutMilliseconds *int32
```

* Adding a utility function to get a single duration from second / millisecond values

```go
// GetProbeTimeDuration combines second and millisecond time increments into a single time.Duration
func GetProbeTimeDuration(seconds int32, milliseconds *int32) time.Duration {
	if milliseconds != nil {
		return time.Duration(seconds)*time.Second + time.Duration(*milliseconds)*time.Millisecond
	}
	return time.Duration(seconds) * time.Second
```

This would be substituted for use in the various places where times are used in `pkg/kubelet/prober`, for example:

```go
-	timeout := time.Duration(p.TimeoutSeconds) * time.Second                      // existing code, to be replaced
+	timeout := GetProbeTimeDuration(p.TimeoutSeconds, p.TimeoutMilliseconds)      // new usage, replaces line above
```

### Existing Struct
https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/types.go

https://github.com/kubernetes/kubernetes/blob/7e7bc6d53b021be6fe3d5a1125a990913b7a9028/pkg/apis/core/types.go#L2062-L2094

```go
// Probe describes a health check to be performed against a container to determine whether it is
// alive or ready to receive traffic.
type Probe struct {
	// The action taken to determine the health of a container
	ProbeHandler
	// Length of time before health checking is activated.  In seconds.
	// +optional
	InitialDelaySeconds int32
	// Length of time before health checking times out.  In seconds.
	// +optional
	TimeoutSeconds int32
	// How often (in seconds) to perform the probe.
	// +optional
	PeriodSeconds int32
	// Minimum consecutive successes for the probe to be considered successful after having failed.
	// Must be 1 for liveness and startup.
	// +optional
	SuccessThreshold int32
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	// +optional
	FailureThreshold int32
	// Optional duration in seconds the pod needs to terminate gracefully upon probe failure.
	// The grace period is the duration in seconds after the processes running in the pod are sent
	// a termination signal and the time when the processes are forcibly halted with a kill signal.
	// Set this value longer than the expected cleanup time for your process.
	// If this value is nil, the pod's terminationGracePeriodSeconds will be used. Otherwise, this
	// value overrides the value provided by the pod spec.
	// Value must be non-negative integer. The value zero indicates stop immediately via
	// the kill signal (no opportunity to shut down).
	// This is a beta field and requires enabling ProbeTerminationGracePeriod feature gate.
	// +optional
	TerminationGracePeriodSeconds *int64
}
```

### Existing Behavior

Taking the Seconds fields, in order of the struct.
InitialDelaySeconds
TimeoutSeconds
PeriodSeconds

#### Defaulting logic
 - InitialDelaySeconds, has no defaulting logic.
   Therefore it would get the golang defaulting logic, and default to 0.
 - TimeoutSeconds, defaults to 1 second if unset (which includes the explicitly set to 0 state). This is the last ending time before the timeout fails. Defaulting, thus, results in slow failures (when one considers a full second of time in cpu time).
https://github.com/kubernetes/kubernetes/blob/e19964183377d0ec2052d1f1fa930c4d7575bd50/pkg/apis/core/v1/defaults.go#L224-L226

```go
 	if obj.TimeoutSeconds == 0 {
		obj.TimeoutSeconds = 1
	}
```

 - Period Seconds is the biggest hurdle in charging and overall cost to the cluster, but also the most useful if set correctly.
PeriodSeconds defaults to 10 seconds if unset (which includes the explicitly set to 0 state). Fast failure detections are thus blocked with existing defaults.
https://github.com/kubernetes/kubernetes/blob/e19964183377d0ec2052d1f1fa930c4d7575bd50/pkg/apis/core/v1/defaults.go#L227-L229

```go
	if obj.PeriodSeconds == 0 {
		obj.PeriodSeconds = 10
	}
```

#### Validation of fields

Must be non-negative, Zero or greater.
```go
	allErrs = append(allErrs, ValidateNonnegativeField(int64(probe.InitialDelaySeconds), fldPath.Child("initialDelaySeconds"))...)
	allErrs = append(allErrs, ValidateNonnegativeField(int64(probe.TimeoutSeconds), fldPath.Child("timeoutSeconds"))...)
	allErrs = append(allErrs, ValidateNonnegativeField(int64(probe.PeriodSeconds), fldPath.Child("periodSeconds"))...)

```

#### Fields to Add
What fields may be necessary to add?

In pkg/apis/core/types.go

```go

type Probe struct {
    ...
    ...
    // How often (in milliseconds) to offset PeriodSeconds when performing the probe.
	// +optional
	PeriodMilliseconds *int32
	// Length of time (in milliseconds) to offset IntialDelaySeconds before health checking is activated.
	// +optional
	InitialDelayMilliseconds *int32
}

```
#### Logic for Added Fields
What is the least logic that could be used?

The combined value of `PeriodSeconds` and `PeriodMilliseconds` must be greater than 200ms and greater than 500ms for exec probe. And after success falls back to original seconds value.
The combined value of `IntialDelaySeconds` and `InitialDelayMilliseconds` must be greater than or equal to zero ms.


### Existing use of Probe struct fields.

#### InitialDelaySeconds
https://github.com/kubernetes/kubernetes/blob/7c2e6125694e1aadc78a5fed1cf696872af50a5e/pkg/kubelet/prober/worker.go#L246-L249
```go
	// Probe disabled for InitialDelaySeconds.
	if int32(time.Since(c.State.Running.StartedAt.Time).Seconds()) < w.spec.InitialDelaySeconds {
		return true
	}
```

#### TimeoutSeconds
https://github.com/kubernetes/kubernetes/blob/7c2e6125694e1aadc78a5fed1cf696872af50a5e/pkg/kubelet/prober/prober.go#L160-L213
```go
	timeout := time.Duration(p.TimeoutSeconds) * time.Second
```

#### ProbeSeconds
https://github.com/kubernetes/kubernetes/blob/7c2e6125694e1aadc78a5fed1cf696872af50a5e/pkg/kubelet/prober/worker.go#L131-L167
```go
func (w *worker) run() {
	probeTickerPeriod := time.Duration(w.spec.PeriodSeconds) * time.Second

	// If kubelet restarted the probes could be started in rapid succession.
	// Let the worker wait for a random portion of tickerPeriod before probing.
	// Do it only if the kubelet has started recently.
	if probeTickerPeriod > time.Since(w.probeManager.start) {
		time.Sleep(time.Duration(rand.Float64() * float64(probeTickerPeriod)))
	}

	probeTicker := time.NewTicker(probeTickerPeriod)

	defer func() {
		// Clean up.
		probeTicker.Stop()
		if !w.containerID.IsEmpty() {
			w.resultsManager.Remove(w.containerID)
		}

		w.probeManager.removeWorker(w.pod.UID, w.container.Name, w.probeType)
		ProberResults.Delete(w.proberResultsSuccessfulMetricLabels)
		ProberResults.Delete(w.proberResultsFailedMetricLabels)
		ProberResults.Delete(w.proberResultsUnknownMetricLabels)
	}()

probeLoop:
	for w.doProbe() {
		// Wait for next probe tick.
		select {
		case <-w.stopCh:
			break probeLoop
		case <-probeTicker.C:
		case <-w.manualTriggerCh:
			// continue
		}
	}
}
```

### Summary

The Kubernetes design for Probes includes a Probe struct containing
fields that specify seconds for intervals.
Some users would like to have intervals less than or slightly greater
than one second.
This KEP adds sub-second interval capablilities to Kubernetes Readiness, and Startup Probes with described mitigation limitations.
Liveness probes and more granular timeouts may be investigated in a followup KEP.


### Test Plan


Existing unit tests of prober `k8s.io/kubernetes/pkg/kubelet/prober/prober_manager_test.go`.

Existing node-e2e test `k8s.io/kubernetes/test/e2e/common/container_probe.go`

Enhanced with additional test cases, see the buckets added in [the draft implementation](https://github.com/kubernetes/kubernetes/pull/107958/commits)

### Graduation Criteria


### Upgrade / Downgrade Strategy

### Version Skew Strategy

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [ X ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: SubSecondProbes
    - Components depending on the feature gate:
		kube-apiserver and kubelet
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.

  No

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Also set `disable-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?).

  Yes, see the [drop test](https://github.com/psschwei/kubernetes/blob/1ca40771e26b96d6121c49b19c2cbe6466694c7a/pkg/api/pod/util_test.go#L1029)

* **What happens if we reenable the feature if it was previously rolled back?**

  No breaking changes

* **Are there any tests for feature enablement/disablement?**

  Yes, see the [drop test](https://github.com/psschwei/kubernetes/blob/1ca40771e26b96d6121c49b19c2cbe6466694c7a/pkg/api/pod/util_test.go#L1029)

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

This section will be completed when targeting beta graduation to a release.
### Dependencies

This section will be completed when targeting beta graduation to a release.
### Scalability

Enabling / using this feature will not result in any new API calls.
See earlier KEP text regarding limits and scaling (to zero the millisecond offsets)

No new scalability API types.

No new calls to the cloud provider will be provided.


Enabling this feature will not necessarily result in increasing size or count of
the existing API objects. (3 optional omit if nil *int32 fields for each probe)

Enabling / using this feature will result in changing the time taken (and thus charging) by certain
operations covered by [existing SLIs/SLOs]. Whether Negatively/Positively will be determined by
customers and testing in Beta. Thus, this KEP provides for mitigation in the form of larger
minimums and scaling of the millisecond offsets to zero.

Enabling / using this feature will result in changes to resource usage
(CPU, RAM, disk, IO, ...) in kubelet and runtime coponents. This KEP provides for
mitigation of the changes.

Reducing the probe frequency to subsecond intervals will result in probes polling slightly more
frequently until success, as mitigated for exec probes and restricting to startup and readyness.

In a follow up KEP further mitigations and allowances may be considered based on resource
pressure, use cases for liveness probes, and if exec probe costs can be reduced via
architecual changes.

### Troubleshooting

This section will be completed when targeting beta graduation to a release.

## Implementation History

## Drawbacks


## Alternatives

### `early*Offset`

(copying directly from [@sftim 's comment](https://github.com/kubernetes/enhancements/pull/3067#issuecomment-1039311016))

Add an extra field to specify “early failure”: an integer quantity of milliseconds (or microseconds - we should leave room for that). That early failure offset is something that legacy clients can ignore without a big problem and would fully support round-tripping to a future Pod v2 API that unifies these fields.

Using a negative offset makes the distinction between (1s - 995ms) more obvious to a legacy client that's unaware of the new fields. A positive offset (0s + 5ms, or maybe 0s + 5000μs) could get interpreted quite differently.

If we do that, we should absolutely disallow a negative offset ≥ 1.0s seconds. Otherwise we have a challenge around unambiguous round-tripping (eg from a future Pod v2 API back to Pod v1).

### `*Duration`

This would be very similar to the previous option, except that it would only require one field instead of multiple ones for each unit type. For example,

```yaml
  periodDuration: 1.5s
```

This could be used in one of two ways:

It could be added to the existing `*Seconds` value, as done in the previous example (i.e. `periodSeconds + periodDuration`). The same caveats listed in the previous option would also apply in this case (signed vs. unsigned, adding vs. subtracting). In this case, the value would be restricted to less than one second.

Alternatively, it could _replace_ the existing `*Seconds` value, using the normalizing pattern described [here](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#making-a-singular-field-plural).

As @sftim noted, the current API guidelines [require using integers](https://github.com/kubernetes/community/blob/489de956d7b601fd23c8f48a87b501e0de4a9c7f/contributors/devel/sig-architecture/api-conventions.md?plain=1#L878-L880) for durations. That said, the preceding line also mentions that the best approach is [still being determined](https://github.com/kubernetes/community/blob/489de956d7b601fd23c8f48a87b501e0de4a9c7f/contributors/devel/sig-architecture/api-conventions.md?plain=1#L873-L876), so there's some ambiguity in the docs on this topic.

### A new struct, `ProbeOffset`

This one kind of splits the difference between the two previous options, using a `struct` to allow for specifying a time unit and value:

```golang
type ProbeOffset struct {
  value uint32
  unit string
}
```

which would be used as follows

```yaml
  periodOffset:
    value: 500
    unit: milliseconds
```

This requires introducing a new type for duration, which in hindsight does not meet [either of the API conventions](https://github.com/kubernetes/community/blob/489de956d7b601fd23c8f48a87b501e0de4a9c7f/contributors/devel/sig-architecture/api-conventions.md#units) mentioned above.

Usage would be similar to the `*Duration` option, i.e. either replace or add to existing field (and if adding to the existing would be restricted to less than 1 second).

### Setting the time units in a different field, `ReadSecondsAs`

The last option would be specify a specific time unit value that would override, in a sense, the "seconds" in all the `*Seconds` fields. For example

```yaml
  periodSeconds: 100
  readSecondsAs: milliseconds
```

would result in an effective period of `500ms`.

Note that this would apply for all `*Seconds` values: if `readSecondsAs` is used, all probe times would need to be rendered in milliseconds. If field level granuality was required (i.e. one value in seconds, one in milliseconds), may as well create `*Milliseconds` fields.

For more examples of how this would work, see [this WIP PR](https://github.com/kubernetes/kubernetes/pull/107958).

Round-tripping issues are handled in a [drop function](https://github.com/kubernetes/kubernetes/blob/39a3c5c880d79fead1ae4dc80f462cac4a33878f/pkg/api/pod/util.go#L607-L636) (using similar logic as would be needed in the first two options).

While this would reduce the number of new fields one has to add, it is to some degree counterintuitive to have a `*Seconds` field in milliseconds.

### v2 api for probe.

Introduce a v2 API.
This seems too invasive.

All Seconds fields become resource.Quantity instead of int32. This supports subdivision in a single field.

Pros:
minimal changes to API (only adding one field)

Cons:
requires conversion back to seconds on legacy versions (does 1.5s round up or down?)

### OffsetMilliseconds

Use a negative offset, and combine with the existing field.

Example:
    Existing Field: int32 PeriodSeconds
    New Field:      int32 PeriodOffsetMilliseconds

    If I want to set to 0.5 seconds, 500 milliseconds,
    PeriodSeconds <= 10 & PeriodOffsetmilliseconds <= -9500
    OR
    PeriodSeconds <= 1 & PeriodOffsetmilliseconds <= -500
    OR
    PeriodSeconds unset, Default of 10, and PeriodOffsetmilliseconds <= -9500

Detail:
Same number of added fields.

Pros:
Uses the existing field, thus readers of only the existing field will still be acting on something that has been logically set. Without the compensation of a negative offset, the output doesn't make much sense as a decision is being made on half of the information, but there is a logical process to why behavior would occur, rather than allowing something to be set before throwing it out.

Cons:
Complicated logic.
Multiple ways to get same resulting time.
Changing behavior in the future involves touching more code than other solutions.

### Reconcile seconds field to nearest whole second.

Minimum of 1 second remains.

Extra logic in defaulter to use the milliseconds field and automatically set the seconds field, allowing those using the seconds field to get something close-enough. This doesn't make much sense for solving rapidity of probes, only for increasing the granularity, such as if I wanted to run

Pros:
Cons:


## Infrastructure Needed (Optional)

Usual infrastructure.
