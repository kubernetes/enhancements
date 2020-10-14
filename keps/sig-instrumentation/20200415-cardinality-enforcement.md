---
title: Dynamic Cardinality Enforcement
authors:
  - "@logicalhan"
  - "@lilic"
owning-sig: sig-instrumentation
participating-sigs:
  - sig-instrumentation
reviewers:
  - "@dashpole"
  - "@brancz"
  - "@ehashman"
  - "@x13n"
approvers:
  - sig-instrumentation
creation-date: 2020-04-15
last-updated: 2020-05-19
stage: alpha
status: implementable
---

# Dynamic Cardinality Enforcement

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Open-Question](#open-question)
- [Graduation Criteria](#graduation-criteria)
- [Post-Beta tasks](#post-beta-tasks)
- [Implementation History](#implementation-history)
<!-- /toc -->

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

We will expose the machinery and tools to bind a metric's labels to a discrete set of values.

It is *not a goal* to implement and plumb this solution for each Kubernetes component (there are many SIGs and a number of verticals, which may have their own preferred way of doing things). As such it will be up to component owners to leverage this functionality that we provide, by feeding configuration data through whatever mechanism deemed appropriate (i.e. command line flags or reading from a file).

These flags are really only meant to be used as escape hatches, and should not be used to have extremely customized kubernetes setups where our existing dashboards and alerting rule definitions are just not going to apply generally anymore.

## Proposal

The simple solution to this problem would be for each metric added to keep the unbounded dimensions in mind and prevent it from happening. SIG instrumentation has already explicitly stated this in our instrumentation guidelines: which says that ["one should know a comprehensive list of all possible values for a label at instrumentation time."](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/instrumentation.md#dimensionality--cardinality). The problem is more complicated. First, SIG Instrumentation doesn't have a way to validate adherence to SIG instrumentation guidelines outside of reviewing PRs with instrumentation changes. Not only is this highly manual, and error-prone, we do not have a terrific existing procedure for ensuring SIG Instrumentation is tagged on relevant PRs. Even if we do have such a mechanism, it isn't a fully sufficient solution because:

1. metrics changes can be seemingly innocuous, even to the most diligent of code reviewers (i.e these issues are hard to catch)
2. metrics cardinality issues exist latently; in other words, they're all over the place in Kubernetes already (so even if we could prevent 100% of these occurrence from **now on**, that wouldn't guarantee Kubernetes is free of these classes of issues).

Instead, the proposed solution is we will be able to *dynamically configure* an allowlist of label values for a metric. By *dynamically configure*, we mean configure an allowlist *at runtime* rather than during the build/compile step. And by *at runtime*, we mean, more specifically, during the boot sequence for a Kubernetes component (and we mean daemons here, not CLI tools like kubectl).

Brief aside: a Kubernetes component (which is a daemon) is an executable, which can be launched from the command line manually if desired. Components take a number of start-up configuration flags, which are passed into the component to modify execution paths (if curious, check out the [large amount of flags we have on the kubelet](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/)). It is also possible to read configuration data from files (like yaml format) during the component boot sequence. This KEP does not have an opinion on the specific mechanism used to load config data into a Kubernetes binary during the boot sequence. What we *actually* care about, is just the fact that it is possible.

Our design is thus config-ingestion agnostic.

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

## Open-Question


## Graduation Criteria

todo


## Post-Beta tasks

todo

## Implementation History

todo

