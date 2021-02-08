# Dynamic Cardinality Enforcement

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
<!-- /toc -->
## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
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

Background for this KEP is that metrics with unbounded dimensions can cause memory issues in the components they instrument. Our proposed solution for this is we want to be able to *dynamically configure* an allowlist of label values for a metric *at runtime*.

## Motivation

Having metrics turn into memory leaks is a problem, but what is even a bigger problem is  when we can't fix these issues without re-releasing the entire Kubernetes binary.

Historically we have approached these issues in various ways and were not consistent. A few approaches:

- Sometimes, a [metric label dimension is intended to be bound to some known sets of values but coding mistakes cause IDs to be thrown in as a label value](https://github.com/kubernetes/kubernetes/issues/53485).

- In another case, [we opt to delete the entire metric](https://github.com/kubernetes/kubernetes/pull/74636) because it basically can't be used in a meaningfully correct way.

- Recently we've opted to both (1) [wholesale delete a metric label](https://github.com/kubernetes/kubernetes/pull/87669) and (2) [retroactively introduce a set of acceptable values for a metric label](https://github.com/kubernetes/kubernetes/pull/87913).

Fixing these issues is currently a manual process, both laborious and time-consuming. We don't have a standard prescription for resolving this class of issue.

### Goals

This KEP proposes a possible solution for this, with the primary goal being to enable binding a metric dimension to a known set of values in such a way that is not coupled to code releases of Kubernetes.

### Non-Goals

We will expose the machinery and tools to bind a metric's labels to a discrete set of values. The allowlist will be ingested via a new-added component metric flag.

It is *not a goal* to define the allowlist for each Kubernetes component metrics.

## Proposal

The simple solution to this problem would be for each metric added to keep the unbounded dimensions in mind and prevent it from happening. SIG instrumentation has already explicitly stated this in our instrumentation guidelines: which says that ["one should know a comprehensive list of all possible values for a label at instrumentation time."](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/instrumentation.md#dimensionality--cardinality). The problem is more complicated. First, SIG Instrumentation doesn't have a way to validate adherence to SIG instrumentation guidelines outside of reviewing PRs with instrumentation changes. Not only is this highly manual, and error-prone, we do not have a terrific existing procedure for ensuring SIG Instrumentation is tagged on relevant PRs. Even if we do have such a mechanism, it isn't a fully sufficient solution because:

1. metrics changes can be seemingly innocuous, even to the most diligent of code reviewers (i.e these issues are hard to catch)
2. metrics cardinality issues exist latently; in other words, they're all over the place in Kubernetes already (so even if we could prevent 100% of these occurrence from **now on**, that wouldn't guarantee Kubernetes is free of these classes of issues).

Instead, the proposed solution is we will be able to *dynamically configure* an allowlist of label values for a metric. By *dynamically configure*, we mean configure an allowlist *at runtime* rather than during the build/compile step. And by *at runtime*, we mean, more specifically, during the boot sequence for a Kubernetes component (and we mean daemons here, not CLI tools like kubectl).

Brief aside: a Kubernetes component (which is a daemon) is an executable, which can be launched from the command line manually if desired. Components take a number of start-up configuration flags, which are passed into the component to modify execution paths (if curious, check out the [large amount of flags we have on the kubelet](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/)). Our design is going to add a flag that applies to all components to ingest the allowlist.

Our design is also based on the premise that metrics can be uniquely identified (i.e. by their metric descriptor). Fortunately for us, this is actually a built in constraint of prometheus clients (which is how we instrument the Kubernetes components). This metric ID is resolvable to a unique string (this is pretty evident when looking at a raw prometheus client endpoint).

We want to provide a data structure which basically maps a unique metric ID and one of it's labels to a bounded set of values.

Purely for illustration, take the following metric:

```json
[
	{
		"metricFamily": "some_metric"
	}
]

```

We want to be able to express this metric and target a label:


```json
[
	{
		"metricFamily": "some_metric",
		"label": "label_too_many_values",
	}
]

```

And we want to express a set of expected values:


```json
[
	{
		"metricFamily": "some_metric",
		"label": "label_too_many_values",
		"labelValueAllowlist": [
			"1",
			"2",
			"3"
		]
	}
]

```

Since we already have an interception layer built into our Kubernetes monitoring stack (from the metrics stability effort), we can leverage existing wrappers to provide a global entrypoint for intercepting and enforcing rules for individual metrics.

As a result metric data will not be invalid, just data will be more coarse when the metric type is a counter. In the case of a gauge type metric the data will be invalid but the number of series would be bound. One exception to our treatment of labels will be histograms (specifically the label which denotes buckets). If we explicitly declare a whitelist of acceptable values for histogram buckets, then we will simply omit buckets which are not in that list, since we can preserve data fidelity for histogram buckets by bucket omission (for everything except the catch-all bucket).

Following is an example used to demonstrate the proposed solution.

Let's say we have some client code somewhere which instantiates our metric from above:

```golang
	SomeMetric = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Name:           "some_metric",
			Help:           "alalalala",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"label_too_many_values"},
	)
```

And some other code which increments our metric:

```golang
	var label_value string
	label_value = "1"
	SomeMetric.WithLabelValues(label_value).Inc()
	label_value = "2"
	SomeMetric.WithLabelValues(label_value).Inc()
	label_value = "3"
	SomeMetric.WithLabelValues(label_value).Inc()
```

In this case we would expect our raw prometheus metric output to look like this:

```prometheus
# HELP some_metric alalalala
# TYPE some_metric counter
some_metric{label_too_many_values="1"} 1
some_metric{label_too_many_values="2"} 1
some_metric{label_too_many_values="3"} 1
```
This would not change. What would change is if we encounter label values outside our explicit allowlist of values. So if where to encounter this malicious piece of code:


```golang
	var label_value string
	for i=0;i<1000000;i++ {
		label_value = strconv.Itoa(i+4) // bring us out of our expected range
		SomeMetric.WithLabelValues(label_value).Inc()
	}
```

Then in existing Kubernetes components, we would have a terrible memory leak, since we would effectively create a million metrics (one per unique label value). If we curled our metrics endpoint it would thus look something like this:

```prometheus
# HELP some_metric alalalala
# TYPE some_metric counter
some_metric{label_too_many_values="1"} 1
some_metric{label_too_many_values="2"} 1
some_metric{label_too_many_values="3"} 1
some_metric{label_too_many_values="4"} 1
... //
... //
... // zillion metrics here
some_metric{label_too_many_values="1000003"} 1
```

In total we would have more than a million of series for one metric, which is known to cause a memory problem.

With our cardinality enforcer in place, we would expect the output to look like this:

```prometheus
# HELP some_metric alalalala
# TYPE some_metric counter
some_metric{label_too_many_values="1"} 1
some_metric{label_too_many_values="2"} 1
some_metric{label_too_many_values="3"} 1
some_metric{label_too_many_values="unexpected"} 1000000
```

We can effect this change by registering a allowlist to our prometheus registries during the boot-up sequence. Since we intercept metric registration events and have a wrapper around each of the primitive prometheus metric types, we can effectively add some code that looks like this (disclaimer: the below is pseudocode for demonstration):


```golang
const (
	unexpectedLabel = "unexpected"
)

func (v *CounterVec) WithLabelValues(lvs ...string) CounterMetric {
	if !v.IsCreated() {
		return noop // return no-op counter
	}
	// INJECTED CODE HERE
	newLabels = make([]string, len(lvs))
	for i, l := range lvs {

		// do we have an allowlist on any of the labels for this metric?
		if metricLabelAllowlist, ok := MetricsLabelAllowlist[BuildFQName(v.CounterOpts.Namespace, v.CounterOpts.Subsystem, v.CounterOpts.Name)]; ok {
			// do we have an allowlist for this label on this metric?
			if allowlist, ok := metricLabelAllowlist[v.originalLabels[i]]; ok {
				if allowllist.Has(l) {
					newLabels[i] = l
					continue
				}
			}
		}
		newLabels[i] = unexpectedLabel
	}

	return v.CounterVec.WithLabelValues(newLabels...)
	// END INJECTED CODE
}
```

This design allows us to optionally adopt the idea about simplifying the interface for component owners, who can then opt to just specify a metric and label pair *without* having to specify an allowlist. The good part of this idea is it simplifies how a component owner can implement our cardinality enforcing helpers without having to necessary plumb through complicated maps. This would make it considerably easier to feed this data in through the command line since it can be done like this:

```bash
$ kube-apiserver --accepted-metric-labels "some_metric=label_too_many_values"
```

This would then be interpreted by our machinery as this:


```json
[
	{
		"metric-id": "some_metric",
		"label": "label_too_many_values",
		"labelValueAllowlist": []
	}
]

```

## Design Details
### Test Plan
For `Alpha`, unit test to verify that the metric label will be set to "unexpected" if the metric encounters label values outside our explicit allowlist of values.
### Graduation Criteria
For `Alpha`, the allowlist of metrics can be configured via the exposed flag and the unit test is passed.
### Upgrade / Downgrade strategy
N/A
### Version Skew Strategy
N/A

## Production Readiness Review Questionnaire
### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
  - [x] Other
    - Describe the mechanism: 
      New flag will be used to config the allowlist of label values for a metric.
      This flag will become standard flag for all k8s components and will be added to
      `k8s.io/component-base`.
    - Will enabling / disabling the feature require downtime of the control
      plane? Yes, the components need to restart with flag enabled.
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled). 
      Yes, the components need to restart with flag enabled.

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.
  Using this feature requires restarting the component with the flag enabled. Once enabled, the metric label will be set to "unexpected" if the metric encounters label values outside our explicit allowlist of values. 

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Also set `disable-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?).
  Yes, restarting the component without the allowlist flag will basically disable this feature.
  
* **What happens if we reenable the feature if it was previously rolled back?**
  The enable-disable-enable process will not cause problem. But it may be problematic during the rolled back period with the unbounded metrics value.
  
* **Are there any tests for feature enablement/disablement?**
  No.  
### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?
  Using this feature requires restarting the component with the flag enabled.
* **What specific metrics should inform a rollback?**
  None.
* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.
  No.
* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  A component metric flag for ingesting allowlist to be added.
### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  The out-of-bound data will be recorded with label "unexpected" rather than the specific value.
* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  None.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  None.

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  None.

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  No.


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  No.

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  No.

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
  No.
* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  No.

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  Checking metrics label against allowlist may increase the metric recording time.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  No.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
  No additional impact comparing to what already exists.
* **What are other known failure modes?**
  None.

* **What steps should be taken if SLOs are not being met to determine the problem?**
  None.

## Implementation History

2020-04-15: KEP opened

2020-05-19: KEP marked implementable
