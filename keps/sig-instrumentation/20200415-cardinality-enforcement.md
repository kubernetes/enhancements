---
title: Dynamic Cardinality Enforcement
authors:
  - "@logicalhan"
owning-sig: sig-instrumentation
participating-sigs:
  - sig-instrumentation
reviewers:
  - todo
approvers:
  - todo
editor: todo
creation-date: 2020-04-15
last-updated: 2020-04-15
status: provisional
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

TLDR; metrics with unbounded dimensions can cause memory issues in the components they instrument.

The simple solution to this problem is to say "don't do that". SIG instrumentation has already explicitly stated this in our instrumentation guidelines: which says that ["one should know a comprehensive list of all possible values for a label at instrumentation time."](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/instrumentation.md#dimensionality--cardinality).

The problem is more complicated. First, SIG Instrumentation doesn't have a way to validate adherence to SIG instrumentation guidelines outside of reviewing PRs with instrumentation changes. Not only is this highly manual, and error-prone, we do not have a terrific existing procedure for ensuring SIG Instrumentation is tagged on relevant PRs. Even if we do have such a mechanism, it isn't a fully sufficient solution because:

1. metrics changes can be seemingly innocous, even to the most diligent of code reviewers (i.e these issues are hard to catch)
2. metrics cardinality issues exist latently; in other words, they're all over the place in Kubernetes already (so even if we could prevent 100% of these occurrence from **now on**, that wouldn't guarantee Kubernetes is free of these classes of issues).


## Motivation

TLDR; Having metrics turn into memory leaks sucks and it sucks even more when we can't fix these issues without re-releasing the entire Kubernetes binary.

**Q:** *How have we approached these issues historically?*

**A:** __Unfortunately, not consistently.__

Sometimes, a [metric label dimension is intended to be bound to some known sets of values but coding mistakes cause IDs to be thrown in as a label value](https://github.com/kubernetes/kubernetes/issues/53485).

In anther case, [we opt to delete the entire metric](https://github.com/kubernetes/kubernetes/pull/74636) because it basically can't be used in a meaningfully correct way.

Recently we've opted to both (1) [wholesale delete a metric label](https://github.com/kubernetes/kubernetes/pull/87669) and (2) [retroactively introduce a set of acceptable values for a metric label](https://github.com/kubernetes/kubernetes/pull/87913).

Fixing these issues is a currently a manual process, both laborious and time-consuming.

We don't have a standard prescription for resolving this class of issue. This is especially bad when you consider that this class of issue is so totally predictable (in general).

### Goals

This KEP proposes a possible solution for this, with the primary goal being to enable binding a metric dimension to a known set of values in such a way that is not coupled to code releases of Kubernetes.

### Non-Goals

We will expose the machinery and tools to bind a metric's labels to a discrete set of values.

It is *not a goal* to implement and plumb this solution for each Kubernetes component (there are many SIGs and a number of verticals, which may have their own preferred way of doing things). As such it will be up to component owners to leverage this functionality that we provide, by feeding configuration data through whatever mechanism deemed appropriate (i.e. command line flags or reading from a file).

## Proposal

TLDR; we want to be able to *dynamically configure* a whitelist of label values for a metric.

By *dynamically configure*, we mean configure a whitelist *at runtime* rather than during build/compile step (this is so ugh).

And by *at runtime*, we mean, more specifically, during the boot sequence for a Kubernetes component (and we mean daemons here, not cli tools like kubectl).

Brief aside: a Kubernetes component (which is a daemon) is an executable, which you can launch from the command line manually if you so desired. Components take a number of start-up configuration flags, which are passed into the component to modify execution paths (if curious, you can check out the [zillion flags we have on the kubelet](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/)). It is also possible to read configuration data from files (like yamls) during the component boot sequence. This KEP does not have an opinion on the specific mechanism used to load config data into a Kubernetes binary during the boot sequence. What we *actually* care about, is just the fact that it is possible.

Our design, is thus config-ingestion agnostic.

Our design is also based on the premise that metrics can be uniquely identified (i.e. by their metric descriptor). Fortunately for us, this is actually a built in constraint of prometheus clients (which is how we instrument the Kubernetes components). This metric ID is resolvable to a unique string (this is pretty evident when you look at a raw prometheus client endpoint).

We want to provide a data structure which basically maps a unique metric ID and one of it's labels to a bounded set of values.

Purely for illustration, take the following metric:

```json
[
	{
		"metric-id": "some_metric"
	}
]

```

We want to be able to express this metric and target a label:


```json
[
	{
		"metric-id": "some_metric",
		"label": "label_too_many_values",
	}
]

```

And we want to express a set of expected values:


```json
[
	{
		"metric-id": "some_metric",
		"label": "label_too_many_values",
		"labelValueWhitelist": [
			"1",
			"2",
			"3"
		]
	}
]

```

Since we already have an interception layer built into our Kubernetes monitoring stack (from the metrics stability effort), we can leverage existing wrappers to provide a global entrypoint for intercepting and enforcing rules for individual metrics.

**Q:** *But won't this invalidate our metric data?*

**A:** __It shouldn't invalidate metric data. But metric data may be more coarse.__

It's easier to demonstrate our strategy than explain stuff out in written word, so we're going to just do that. Let's say we have some client code somewhere which instantiates our metric from above:

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
This would not change. What would change is if we encounter label values outside our explicit whitelist of values. So if where to encounter this malicious piece of code:


```golang
	var label_value string
	for i=0;i<1000000;i++ {
		label_value = strconv.Itoa(i+4) // bring us out of our expected range
		SomeMetric.WithLabelValues(label_value).Inc()
	}
```

Then in existing Kubernetes components, we would have a terrible memory leak, since we would effectively create a million metrics (one per unique label value). If we curled our prometheus endpoint it would thus look something like this:

```prometheus
# HELP some_metric alalalala
# TYPE some_metric counter
some_metric{label_too_many_values="1"} 1
some_metric{label_too_many_values="2"} 1
some_metric{label_too_many_values="3"} 1
some_metric{label_too_many_values="4"} 1
... //
... // zillion metrics here
some_metric{label_too_many_values="1000003"} 1
```

That would suck. With our cardinality enforcer in place, we would expect the output to look like this:

```prometheus
# HELP some_metric alalalala
# TYPE some_metric counter
some_metric{label_too_many_values="1"} 1
some_metric{label_too_many_values="2"} 1
some_metric{label_too_many_values="3"} 1
some_metric{label_too_many_values="unexpected"} 1000000
```

We can effect this change by registering a whitelist to our prom registries during the boot-up sequence. Since we intercept metric registration events and have a wrapper around each of the primitive prometheus metric types, we can effectively add some code that looks like this (disclaimer: the below is pseudocode for demonstration):

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

		// do we have a whitelist on any of the labels for this metric?
		if metricLabelWhitelist, ok := MetricsLabelWhitelist[BuildFQName(v.CounterOpts.Namespace, v.CounterOpts.Subsystem, v.CounterOpts.Name)]; ok {
			// do we have a whitelist for this label on this metric?
			if whitelist, ok := metricLabelWhitelist[v.originalLabels[i]]; ok {
				if whitelist.Has(l) {
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

This design allows us to optionally adopt @lilic's excellent idea about simplifying the interface for component owners, who can then opt to just specify a metric and label pair *without* having to specify a whitelist. Personally, I like that idea since it simplifies how a component owner can implement our cardinality enforcing helpers without having to necessary plumb through complicated maps. This would make it considerably easier to feed this data in through the command line since you could do something like this:

```bash
$ kube-apiserver --accepted-metric-labels "some_metric=label_too_many_values"
```

..which would then be interpreted by our machinery as this:


```json
[
	{
		"metric-id": "some_metric",
		"label": "label_too_many_values",
		"labelValueWhitelist": []
	}
]

```

## Open-Question
_(Discussion Points which need to be resolved prior to merge)_

- @dashpole

> Should have labels with a specific set of values, should we start enforcing that all metrics have a whitelist at compile-time?

- @x13n

> ... instead of getting label_too_many_values right away, [enforcing the cardinality limit directly] would still work until certain label cardinality limit is reached. Whitelisting would guarantee a value will not be dropped, but other values wouldn't be dropped either unless there is too many of them. Cluster admin can configure the per metric and per label limits once and can get alerted on "some metric labels are dropped" instead of "your metrics storage is getting blown up".

- @brancz/@lilic

> Potentially we would want to treat buckets completely separately (as in a separate flag just for bucket configuration of histograms). @bboreham opened the original PR for apiserver request duration bucket reduction, maybe he has some input as well.
>
> My biggest concern I think with all of this is, it's going to be super easy to have extremely customized kubernetes setups where our existing dashboards and alerting rule definitions are just not going to apply generally anymore. I'd like to make sure we emphasize that these flags are really only meant to be used as escape hatches, and we must always strive to truly fix the root of the issue.


## Graduation Criteria

todo


## Post-Beta tasks

todo

## Implementation History

todo