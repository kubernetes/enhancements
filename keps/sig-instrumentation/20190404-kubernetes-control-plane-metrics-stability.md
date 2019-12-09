---
title: Kubernetes Control-Plane Metrics Stability
authors:
  - "@logicalhan"
owning-sig: sig-instrumentation
participating-sigs:
  - sig-instrumentation
  - sig-api-machinery
  - sig-node
reviewers:
  - "@brancz"
  - "@x13n"
  - "@DirectXMan12"
  - "@lavalamp"
  - "@dashpole"
  - "@ehashman"
  - "@mml"
approvers:
  - "@brancz"
  - "@x13n"
editor: "@brancz"
creation-date: 2019-04-04
last-updated: 2019-11-06
status: implementable
see-also:
  - 20181106-kubernetes-metrics-overhaul
---

# Kubernetes Control-Plane Metrics Stability

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Background](#background)
- [Proposal](#proposal)
  - [Metric Definition Phase](#metric-definition-phase)
  - [Metric Instantiation Phase](#metric-instantiation-phase)
  - [Metric Enrollment Phase](#metric-enrollment-phase)
- [Stability Classes](#stability-classes)
- [API Review](#api-review)
- [Deprecation Lifecycle](#deprecation-lifecycle)
  - [Show Hidden Metrics](#show-hidden-metrics)
  - [Why Not Bool Flag](#why-not-bool-flag)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Unresolved Questions](#unresolved-questions)
  - [Static Analysis for Validation](#static-analysis-for-validation)
  - [Beta Stability Level](#beta-stability-level)
  - [Prometheus Labels vs OpenCensus-type Tags](#prometheus-labels-vs-opencensus-type-tags)
  - [Dynamically Registered Metrics](#dynamically-registered-metrics)
- [Implementation History](#implementation-history)
- [References](#references)
    - [Metric Renaming](#metric-renaming)
<!-- /toc -->

## Summary

Currently metrics emitted in the Kubernetes control-plane do not offer any stability guarantees. This Kubernetes Enhancement Proposal (KEP) proposes a strategy and framework for programmatically expressing how stable a metric is, i.e. whether a metric's name, type and [labels](https://prometheus.io/docs/practices/naming/#labels) (i.e. dimensions) is liable to change. Since this document will likely evolve with ongoing discussion around metric stability, it will be updated accordingly.

## Motivation

Metrics stability has been an ongoing community concern. Oftentimes, cluster monitoring infrastructure assumes the stability of at least some control-plane metrics; thus, it would be prudent to offer some sort of guarantees around control-plane metrics, treating it more properly as an API.
Since the [metrics overhaul](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/20181106-kubernetes-metrics-overhaul.md) is nearing completion, there should be less reason to introduce breaking changes to metrics, making it an opportune time to introduce metric stability rules. Specifically, this KEP intends to address metric stability from an ingestion point of view.

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
* Defining the precise mechanism by which we will validate metric rule violations (i.e. static analysis) is also __*not*__ in scope.

## Background

Kubernetes control-plane binaries (i.e. scheduler, kubelet, controller-manager, apiserver) use a Prometheus client to export binary-specific metrics to a ‘/metrics’ endpoint in Prometheus format. Metrics are first defined and then instantiated; later they are registered to a metrics registry. The http handler for the metrics endpoint then delegates responses to the underlying registry.

For the remainder of this document, I will refer to the following terms by these definitions:

* __metric definition__ - this refers to defining a metric. In Kubernetes, we use the standard Prometheus pattern of using an options struct to define name, type, description of a metric.
* __metric instantiation__ - this refers to creating an instance of a metric. A metric definition is passed into a metric constructor which, in Kubernetes, is a Prometheus metric constructor (example).
* __metric enrollment__ - after being defined and created, individual metrics are officially enrolled to a metrics registry (currently a global one).
* __metric registration process__ - I use this to refer to the entire lifecycle of a metric from definition, to instantiation, then enrollment.

The fact that the metric registration process always involves these steps is significant because it allows for the possibility of injecting custom behavior in and around these steps.

## Proposal

This KEP proposes a programmatic mechanism to express the stability of a given control plane metric. Individual metrics would be quasi-versioned, i.e. they would have additional bits of metadata which would indicate whether that metric was alpha (not-stable), stable, or deprecated. Metric stability guarantees would depend on the values of those additional bits. This information would be represented to users in two primary ways: (1) metric stability information would be automatically to individual metrics help descriptions and (2) in the case of deprecated metrics, warning logs would be emitted on metric registration.

Specifically, this would involve injecting custom behavior during metric registration process by wrapping metric definition, instantiation and enrollment.

### Metric Definition Phase

Currently, the metric definition phase looks like this:

```go
var someMetricDefinition = prometheus.CounterOpts{
    Name: "some_metric",
    Help: "some description",
}
```

Since we are using the Prometheus provided struct, we are constrained to Prometheus provided fields. However, using a custom struct affords us the following:
```go
var deprecatedMetricDefinition = kubemetrics.CounterOpts{
    Name: "some_deprecated_metric",
    Help: "some description",
    StabilityLevel: kubemetrics.STABLE, // this is also a custom metadata field
    DeprecatedVersion: "1.15", // this is a custom metadata field
}

var alphaMetricDefinition = kubemetrics.CounterOpts{
    Name: "some_alpha_metric",
    Help: "some description",
    StabilityLevel: kubemetrics.ALPHA, // this is also a custom metadata field
	DeprecatedVersion: "1.15", // this can optionally be included on alpha metrics, although there is no change to contractual stability guarantees
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

Wrapping the Prometheus constructors would allow us to take, as inputs, the modified metric definitions defined above, returning a custom Kubernetes metric object which contains the metric which would have been instantiated as well as the custom metadata:

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

Currently, metric enrollment involves calls to a Prometheus function which enrolls the metric in a global registry, like so:
```go
prometheus.MustRegister(someCounterVecMetric)
```

Wrapping a prometheus registry with a Kubernetes specific one, would allow us to take our custom metrics from our instantiation phase and execute custom logic based on our custom metadata. Our custom registry would hold a reference to a prometheus registry and defer metric enrollment unless preconditions were met:

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

This proposal introduces two stability classes for metrics: (1) Alpha, (2) Stable. These classes are intended to make explicit the API contract between the control-plane and the consumer of control-plane metrics.

__Alpha__ metrics have __*no*__ stability guarantees; as such they can be modified or deleted at any time. At this time, all Kubernetes metrics implicitly fall into this category.

__Stable__ metrics can be guaranteed to *not change*, except that the metric may become marked deprecated for a future Kubernetes version. By *not change*, we mean three things:

1. the metric itself will not be deleted ([or renamed](#metric-renaming))
2. the type of metric will not be modified
3. no labels can be added or removed from this metric

From an ingestion point of view, it is backwards-compatible to add or remove possible __values__ for labels which already do exist (but __not__ labels themselves). Therefore, adding or removing __values__ from an existing label is permissible. Stable metrics can also be marked as __deprecated__ for a future Kubernetes version, since this is a metadata field and does not actually change the metric itself.

Removing or adding labels from stable metrics is not permissible. In order to add/remove a label to an existing stable metric, one would have to introduce a new metric and deprecate the stable one; otherwise this would violate compatibility agreements.

As an aside, all metrics should be able to be individually disabled by the cluster administrator, regardless of stability class. By default, all non-deprecated metrics will be automatically registered to the metrics endpoint unless explicitly blacklisted via a command line flag (i.e. '--disable-metrics=somebrokenmetric,anothermetric').

## API Review

Graduating a metric to a stable state is a contractual API agreement, as such, it would be desirable to require an api-review (to sig-instrumentation) for graduating or deprecating a metric (in line with current Kubernetes [api-review processes](https://github.com/kubernetes/community/blob/master/sig-architecture/api-review-process.md)). However, initiating or mandating such API review has historically been problematic for sig-instrumentation since, while a horizontal effort, is not automatically added as reviewers for metrics related changes.

One possible solution is through something similar to the existing Kubernetes conformance test gates (thanks @liggitt for pointing this one out). We will have a script which can generate a list of the current stable metrics (via static analysis). This list will be checked in. During the CI/CD flow, a verify script will run to generate a new list of stable metrics. If there is a diff present, then the verify script will fail, since the file should be updated and checked in. Thus, the file must be checked in and since the file will live in a directory owned by sig-instrumentation, sig-instrumentation approval on that PR will be required.

## Deprecation Lifecycle

This proposal introduces deprecation metadata for metrics, to be used to define a deprecation lifecycle. Metrics can be annotated with a Kubernetes version, from which point that metric will be considered deprecated. This allows us to indicate that a metric is slated for future removal and provides the consumer a reasonable window in which they can make changes to their monitoring infrastructure which depends on this metric.

While deprecation policies only actually change stability guarantees for __stable__ metrics (and not __alpha__ ones), deprecation information may however be optionally provided on alpha metrics to help component owners inform users of future intent, to help with transition plans (this change was made at the request of @dashpole, who helpfully pointed out that it would be nice to be able signal future intent even for alpha metrics).

When a stable metric undergoes the deprecation process, we are signaling that the metric will eventually be deleted. The lifecyle looks roughly like this (each stage represents a Kubernetes release):

__Stable metric__ -> __Deprecated metric__ -> __Hidden metric__ -> __Deletion__

__Deprecated__ metrics have the same stability guarantees of their counterparts. If a stable metric is deprecated, then a deprecated stable metric is guaranteed to *not change*. When deprecating a stable metric, a future Kubernetes release is specified as the point from which the metric will be considered deprecated.

```go
var someCounter = kubemetrics.CounterOpts{
    Name: "some_counter",
    Help: "this counts things",
    StabilityLevel: kubemetrics.STABLE,
    DeprecatedVersion: "1.15", // this metric is deprecated when the Kubernetes version == 1.15
}
````

__Deprecated__ metrics will have their description text prefixed with a deprecation notice string '(Deprecated from x.y)' and a warning log will be emitted during metric registration (in the spirit of the official [Kubernetes deprecation policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-a-flag-or-cli)).

Before deprecation:

```text
# HELP some_counter this counts things
# TYPE some_counter counter
some_counter 0
```

During deprecation:
```text
# HELP some_counter (Deprecated from 1.15) this counts things
# TYPE some_counter counter
some_counter 0
```
Like their stable metric counterparts, deprecated metrics will be automatically registered to the metrics endpoint.

On a subsequent release (when the metric's deprecatedVersion is equal to current_kubernetes_version - 1)), a deprecated metric will become a __hidden metric__. _Unlike_ their deprecated counterparts, hidden metrics will __*no longer be automatically registered*__ to the metrics endpoint (hence hidden). However, they can be explicitly enabled through a command line flag on the binary (i.e. '--show-hidden-metrics-for-version=<previous minor release>'). This is to provide cluster admins an escape hatch to properly migrate off of a deprecated metric, if they were not able to react to the earlier deprecation warnings. Hidden metrics should be deleted after one release.

### Show Hidden Metrics
As described above, admins can enable hidden metrics through a command-line flag on a specific binary. 
This intends to be used as an escape hatch for admins if they missed the migration of the metrics deprecated in the last release. 

The flag `show-hidden-metrics-for-version` takes a version for which you want to show metrics deprecated in that release.
The version is expressed as __x.y__, where __x__ is the major version, __y__ is the minor version.
The patch version is not needed even though a metrics can be deprecated in a patch release, the reason for that is
the metrics deprecation policy runs against the minor release.

The flag can only take the previous minor version as it's value. 
All metrics hidden in previous will be emitted if admins set the previous version to `show-hidden-metrics-for-version`.
The too old version is not allowed because this violates the metrics deprecated policy.

Take metric `A` as an example, here assumed that `A` is deprecated in `1.n`.
According to metrics deprecated policy, we can reach the following conclusion:
- In release `1.n`, the metric is deprecated, and it can be emitted by default.
- In release `1.n+1`, the metric is hidden by default and it can be emitted by command line `show-hidden-metrics-for-version=1.n`.
- In release `1.n+2`, the metric should be removed from the codebase. No escape hatch anymore.

So, if admins want to enable metric `A` in release `1.n+1`, they should set `1.n` to the command line flag. 
That is `show-hidden-metrics=1.n`. 

### Why Not Bool Flag
Alternatively, another solution which was previously suggested(refer to the discussion on [PR](https://github.com/kubernetes/kubernetes/pull/84292)) 
was provide a bool flag-like `show-hidden-metrics`. That works like:
- `show-hidden-metrics=true`: enable all hidden metrics deprecated in a previous minor version.
- `show-hidden-metrics=false`: the default value, do nothing.

This proposal has a side effect(thanks for @lavalamp pointed it out) in the scenario:
1. in version X, turn this on to get back deprecated metric M
2. in version Y, not turn this off, and therefore fail to notice metric N is being deprecated
3. in version Z, metric N is removed with (effectively) no warning

## Design Details

### Test Plan

Internal wrappers can be unit tested. There has been some discussion around providing APIs around metric definitions, hopefully with the end goal of being able to augment our test strategy with current and historical metric definition data.

### Graduation Criteria

This feature enhancement will not require graduation criteria. We intend to isolate these changes into a sig-instrumentation repo, build out the framework and the static analysis validation piece (which will require it's own KEP) and then have another KEP which details migration strategy across the kubernetes codebase. Since this can be built in place without affecting any surrounding code, it is safe.

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

### Static Analysis for Validation

Having a set of wrappers in place which allows us to define custom metadata on metrics is quite powerful, since it enables a number of _theorectically possible_ features, like static analysis for verifying metric guarantees during a precommit phase. How we would actually go about doing this is TBD. It is possible to use the metrics registry to output metric descriptions in a separate endpoint; using static analysis we could validate metrics descriptions with our stability rules.

### Beta Stability Level

Currently I am inclined to omit the beta stage from metric versioning if only to reduce initial complexity. It may however become more desirable to include this state in a later design/implementation phase.

### Prometheus Labels vs OpenCensus-type Tags

Having these series of wrappers in place allows us to potentially provide a custom wrapper struct around Prometheus labels. This is particularly desirable because labels are shared across metrics and we may want to define uniform behavior for a given label ([constraining values for labels](https://github.com/kubernetes/kubernetes/issues/75839#issuecomment-478654080), [whitelisting values for a label](https://github.com/kubernetes/kubernetes/issues/76302)). Prometheus labels are pretty primitive (i.e. lists of strings) but potentially we may want an abstraction which more closely resembles [open-census tags](https://opencensus.io/tag/).

### Dynamically Registered Metrics

Metrics which are added dynamically after application boot - Metrics which are dynamically added depending on things which occur during runtime should probably not be allowed to be considered stable metrics, since we can't rely on them to exist reliably.

## Implementation History

- Wrappers added around prometheus registry and counter/counterVec metric type [PR-77037](https://github.com/kubernetes/kubernetes/pull/77037)
- Wrappers added around prometheus gauge, histogram & summary metric types [PR-77618](https://github.com/kubernetes/kubernetes/pull/77618)
- Make sig-instrumentation owners for component-base metrics directory [PR-77621](https://github.com/kubernetes/kubernetes/pull/77621)

## References

#### Metric Renaming

Metric renaming is be tantamount to deleting a metric and introducing a new one. Accordingly, metric renaming will also be disallowed for stable metrics.
