---
title: Kubernetes Control-Plane Metrics Stability
authors:
  - "@logicalhan"
owning-sig:
  - sig-instrumentation
participating-sigs:
  - sig-instrumentation
  - sig-api-machinery
  - sig-node
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-04-04
see-also:
  - 0031-kubernetes-metrics-overhaul
status: provisional
---

# Kubernetes Control-Plane Metrics Stability

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Background](#background)
* [Proposal](#proposal)
    * [Metric Definition Phase](#metric-definition-phase)
    * [Metric Instantiation Phase](#metric-instantiation-phase)
    * [Metric Enrollment Phase](#metric-enrollment-phase)
* [Design Details](#design-details)
  * [Test Plan](#test-plan)
  * [Graduation Criteria](#graduation-criteria)
* [Drawbacks](#drawbacks)
* [Alternative](#alternatives)
* [Unresolved Questions](#unresolved-questions)
* [Implementation History](#implementation-history)
* [Reference](#reference)

## Summary

Currently metrics emitted in the kubernetes control-plane do not offer any stability guarantees. This Kubernetes Enhancement Proposal (KEP) proposes a strategy and framework for programmatically expressing how stable a metric is. This KEP also defines the specific guarantees made for each enumerated level of stability. Since this document will likely evolve with ongoing discussion around metric stability, it will be updated accordingly.

## Motivation

Metrics stability has been an ongoing community concern. Oftentimes, cluster monitoring infrastructure assumes the stability of at least some control-plane metrics; thus, it would be prudent to offer some sort of guarantees around control-plane metrics, treating it more properly as an API. Since the [metrics overhaul](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/0031-kubernetes-metrics-overhaul.md) is nearing completion, there should be less reason to introduce breaking changes to metrics, making it an opportune time to introduce metric stability rules. Specifically, this KEP intends to address metric stability from an ingestion point of view.

Guarantees around metrics have been [proposed previously](https://docs.google.com/document/d/1_CdNWIjPBqVDMvu82aJICQsSCbh2BR-y9a8uXjQm4TI/edit#) and there are [ongoing community discussions](https://groups.google.com/forum/#!topic/kubernetes-sig-instrumentation/XbElxDtww0Y) around this issue. Some suggested solutions include:

1. Having a ‘stable’ metrics endpoint, i.e. ‘/metrics/v1’
2. Leaving metrics as is and documenting the ones which have a stability guarantee

This KEP suggests another alternative but is very much in line with the spirit of the other proposed solutions.

### Goals

 * Describe the various stability guarantees for the consumption of control-plane metrics.
 * Define a uniform mechanism for expressing metric stability.

### Non-Goals

* We are __*not*__ defining which specific control-plane metrics are actually stable.
* We are __*not*__ providing guarantees around specific values in metrics; as such, breakages in alerting based of off assumptions on specific values in metrics are out-of-scope.

## Background

Kubernetes control-plane binaries (i.e. scheduler, kubelet, controller-manager, apiserver) use a prometheus client to export binary-specific metrics to a ‘/metrics’ endpoint in prometheus format. Metrics are first defined and then instantiated; later they are registered to a metrics registry. The http handler for the metrics endpoint then delegates responses to the underlying registry.

For the remainder of this document, I will refer to the following terms by these definitions:

* __metric definition__ - this refers to defining a metric. In kubernetes, we use the standard prometheus pattern of using an options struct to define name, type, description of a metric.
* __metric instantiation__ - this refers to creating an instance of a metric. A metric definition is passed into a metric constructor which, in kubernetes, is a prometheus metric constructor (example).
* __metric enrollment__ - after being defined and created, individual metrics are officially enrolled to a metrics registry (currently a global one).
* __metric registration process__ - I use this to refer to the entire lifecycle of a metric from definition, to instantiation, then enrollment.

The fact that the metric registration process always involves these steps is significant because it allows for the possibility of injecting custom behavior in and around these steps.

## Proposal

This KEP proposes a programmatic mechanism to express the stability of a given control plane metric. Individual metrics would be quasi-versioned, i.e. they would have additional bits of metadata which would indicate whether that metric was alpha (not-stable), stable, or deprecated. Metric stability guarantees would depend on the values of those additional bits.

Specifically, this would involve injecting custom behavior during metric registration process by wrapping metric definition, instantiation and enrollment.

### Metric Definition Phase

Currently, the metric definition phase looks like this:

```go
var someMetricDefinition = prometheus.CounterOpts{
    Name: "some_metric",
    Help: "some description",
}
```

Since we are using the prometheus provided struct, we are constrained to prometheus provided fields. However, using a custom struct affords us the following:
```go
var deprecatedMetricDefinition = kubemetrics.CounterOpts{
    Name: "some_deprecated_metric",
    Help: "some description",
    DeprecatedVersion: "1.15", // this is a custom metadata field
}

var alphaMetricDefinition = kubemetrics.CounterOpts{
    Name: "some_alpha_metric",
    Help: "some description",
    StabilityLevel: kubemetrics.ALPHA, // this is also a custom metadata field
}
```

### Metric Instantiation Phase

Currently, the metric instantiation phase looks like this:

```go
var someCounterVecMetric = prometheus.NewCounterVec(
    someMetricDefinition,
    []string{"some-label", "other-label"},
}
```

Wrapping the prometheus constructors would allow us to take, as inputs, the modified metric definitions defined above, returning a custom kubernetes metric object which contains the metric which would have been instantiated as well as the custom metadata:

```go
var deprecatedMetric = kubemetrics.NewCounterVec( // this is a wrapped initializer, which takes in our custom metric definitions
    deprecatedMetricDefinition, // this is our custom wrapped metric definition from above
    []string{"some-label", "other-label"},
}
var alphaMetric = kubemetrics.NewCounterVec{
    alphaMetricDefinition, // this is also our custom wrapped metric definition from above
    []string{"some-label", "other-label"},
}
```

### Metric Enrollment Phase

Currently, metric enrollment involves calls to a prometheus function which enrolls the metric in a global registry, like so:
```go
prometheus.MustRegister(someCounterVecMetric)
```

Wrapping a prometheus registry with a kubernetes specific one, would allow us to take our custom metrics from our instantiation phase and execute custom logic based on our custom metadata. Our custom registry would hold a reference to a prometheus registry and defer metric enrollment unless preconditions were met:

```go
import version "k8s.io/apimachinery/pkg/version"

type Registry struct {
    promregistry *prometheus.Registry
    KubeVersion version.Info
}

// inject custom registration behavior into our registry wrapper
func (r *Registry) MustRegister(metric kubemetrics.Metric) {
    // pretend we have a version comparison utility library
    if metricutils.compare(metric.DeprecatedVersion).isLessThan(r.KubeVersion) {
        // check if binary has deprecated metrics enabled otherwise
        // no-op registration
        return
    } else if metricutils.compare(metric.DeprecatedVersion).isEqual(r.KubeVersion) {
        // append deprecated text to description
        // emit warning in logs
        // continue to actual registration
    }
    // append alpha text to metric description if metric.isAlpha
    // fallback to original prometheus behavior
    r.promregistry.MustRegister(metric.realMetric)
}

```

Which we would invoke, like so:
```go
kubemetrics.MustRegister(deprecatedMetric)
kubemetrics.MustRegister(alphaMetric)
```

## Stability Classes

This proposal introduces three stability classes for metrics: (1) Alpha, (2) Stable, (3) Deprecated. These classes are intended to make explicit the API contract between the control-plane and the ingester of control-plane metrics.

__Alpha__ metrics have __*no*__ stability guarantees; as such they can be modified or deleted at any time. At this time, all kubernetes metrics implicitly fall into this category.

__Stable__ metrics can be guaranteed to *not change*, except that the metric may become marked deprecated for a future kubernetes version. By *not change*, we mean three things:

1. the metric itself will not be deleted
3. the type of metric will not be modified
4. no labels can be added or removed from this metric

As an aside, in this document, we consider metric renaming to be tantamount to deleting a metric and introducing a new one. Accordingly, metric renaming will also be disallowed for stable metrics. 

From an ingestion point of view, it is backwards-compatible to add or remove possible __values__ for labels which already do exist (but __not__ labels themselves). Therefore, adding or removing __values__ from an existing label is permissible. Stable metrics can also be marked as __deprecated__ for a future kubernetes version, since this is a metadata field and does not actually change the metric itself.

__Deprecated__ metrics are also guaranteed to *not change*. The purpose of deprecated metrics is to provide a reasonable window for consumers of control-plane metrics to make changes to their monitoring infrastructure (and also so that we do not handcuff control-plane developers to a set of stable metrics which they will have to then support for all time). Deprecated metrics can fall under one of two categories:

 1. _soft deprecated_ - this occurs when a stable metric has been recently deprecated; 'recently deprecated' is defined as when the metric's deprecated version is equal to the current kubernetes version
 2. _hard deprecated_ - this occurs when the metric has been deprecated for > 1 release.

Soft deprecated metrics will have their description text prefixed with a deprecation notice string '(Deprecated from x.y)' and a warning log will be emitted during metric registration (in the spirit of the official [kubernetes deprecation policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-a-flag-or-cli)). Like their stable metric counterparts, soft deprecated metrics will be automatically registered to the metrics endpoint.

Metrics are _hard deprecated_ once they have been deprecated for more than one release. _Unlike_ their soft deprecated counterparts, hard deprecated metrics will __*no longer be automatically registered*__ to the metrics endpoint. By default, hard deprecated metrics will not be registered unless explicitly instructed to do so through command line flag on the binary (i.e. '--enable-hard-deprecated-metrics=all-metrics'). It is expected that deprecated metrics will be removed once they have been deprecated for more than one release. As such, metrics which have been deprecated for more than one release will have __no__ stability guarantees.

## Design Details

### Test Plan

Internal wrappers can be unit tested. There has been some discussion around providing APIs around metric definitions, hopefully with the end goal of being able to augment our test strategy with current and historical metric definition data.

### Graduation Criteria

TBD

## Drawbacks

More generally, this proposal has the drawbacks which any proposal suggesting a more rigorous enforcement of an API is going to have. There is always a tradeoff between the ease at which a developer can make breaking changes to an API, with consumers' ability to reliably use that API.

Relative to a more hands-off approach, like one where we just document the metrics which the community has agreed to 'certify' as stable, this approach is definitely more heavyweight. This approach involves more code and more code is more maintenance. However, most of the code will be centralized and the internal logic is easily unit-testable. We also do not have to worry much about changing internal API semantics, since our wrappers will be used internally only, which means it should be easy to modify for new usecases in the future. This sort of approach also enables static analysis tooling around metrics which we could run in precommit.

Also, we should note that this approach can be manufactured in-place; this framework could be rolled out without actually introducing any backwards-incompatible changes (unlike moving stable metrics to a '/metrics/v1' endpoint).

There is also some inflexibility in responding to the situation where code is re-architected in such a way that it's no longer feasible to provide a metric (e.g. there's no longer anything to measure). Generally, we would want to try to avoid this situation by not making a metric stable if there's any way for it to get refactored away. Currently, in this sort of case, the metrics stability proposal would only dictate that we continue to register the metric and undergo the normal metric deprecation policy, as it would be necessary for avoiding ingestion pipeline breakages (thanks @DirectXMan12 for pointing this out).

## Alternatives

Using a more traditional versioned endpoint was one of the first suggested ideas. However, metrics basically form a single API group so making a change to a single (previously considered stable) metric would necessitate a version bump for all metrics. In the worst case, version bumps for metrics could occur with each release, which is undesirable from a consumption point of view.

It would also be possible to group metrics into distinct endpoints, in order to avoid global version bumps. However, this breaks the more common metrics ingestion patterns, i.e. as a consumer of metrics for a component, you would no longer be able to assume all of your relevant metrics come from one location. This is also a potentially confusing pattern for consumer of metrics, since you would have to manage a series of metrics for a given component and also be cognizant of the version for each of these components. It would be easy to get wrong.

Alternatively, one lightweight solution which was previously suggested was documenting the metrics which have stability guarantees. However, this is prone to documentation rot and adds manual (and error-prone) overhead to the metrics process.

## Unresolved Questions

Static analysis for validation - Having a set of wrappers in place which allows us to define custom metadata on metrics is quite powerful, since it enables a number of _theorectically possible_ features, like static analysis for verifying metric guarantees during a precommit phase. How we would actually go about doing this is TBD. It is possible to use the metrics registry to output metric descriptions in a separate endpoint; using static analysis we could validate metrics descriptions with our stability rules.

Initial alpha phase - Ideally, I would like to have alpha metrics disabled by default, but toggleable if explicit command-line flags are passed to the binary (i.e. '--enable-alpha-metrics=all-metrics'). This would be consistent with the traditional kubernetes definition of alpha features. However, since all control-plane metrics are currently in an alpha state (i.e. have no stability guarantees), disabling alpha metrics by default would entail that shipping this feature would mean no metrics would be enabled by default, which is obviously undesirable.

Beta metrics - Currently I am inclined to omit the beta stage from metric versioning if only to reduce initial complexity. It may however become more desirable to include this state in a later design/implementation phase.

Prometheus labels - Having these series of wrappers in place allows us to potentially provide a custom wrapper struct around prometheus labels. This is particularly desirable because labels are shared across metrics and we may want to define uniform behavior for a given label ([constraining values for labels](https://github.com/kubernetes/kubernetes/issues/75839#issuecomment-478654080), [whitelisting values for a label](https://github.com/kubernetes/kubernetes/issues/76302)). Prometheus labels are pretty primitive (i.e. lists of strings) but potentially we may want an abstraction which more closely resembles [open-census tags](https://opencensus.io/tag/).

Metrics which are added dynamically after application boot - Metrics which are dynamically added depending on things which occur during runtime should probably not be allowed to be considered stable metrics, since we can't rely on them to exist reliably.

## Implementation History

TBD

## References

1. [original proposal](https://docs.google.com/document/d/1CcbfC-M8CHDfq1rMAOtW0-LKHvermyUiV6BMXXYiqoM/edit#)
