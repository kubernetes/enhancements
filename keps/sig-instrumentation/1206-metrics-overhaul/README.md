# Kubernetes Metrics Overhaul

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [cAdvisor instrumentation changes](#cadvisor-instrumentation-changes)
    - [Consistent labeling](#consistent-labeling)
  - [Changing API latency histogram buckets](#changing-api-latency-histogram-buckets)
  - [Kubelet metric changes](#kubelet-metric-changes)
    - [Make metrics aggregatable](#make-metrics-aggregatable)
    - [Prober metrics](#prober-metrics)
  - [Kube-scheduler metric changes](#kube-scheduler-metric-changes)
  - [Kube-proxy metric changes](#kube-proxy-metric-changes)
    - [Change proxy metrics to conform metrics guidelines](#change-proxy-metrics-to-conform-metrics-guidelines)
    - [Clean the deprecated metrics which introduced in v1.14](#clean-the-deprecated-metrics-which-introduced-in-v114)
  - [Kube-apiserver metric changes](#kube-apiserver-metric-changes)
    - [Apiserver and etcd metrics](#apiserver-and-etcd-metrics)
    - [Fix admission metrics in true units](#fix-admission-metrics-in-true-units)
    - [Remove the deprecated admission metrics](#remove-the-deprecated-admission-metrics)
  - [Client-go metric changes](#client-go-metric-changes)
    - [Workqueue metrics](#workqueue-metrics)
  - [Convert latency/latencies in metrics name to duration](#convert-latencylatencies-in-metrics-name-to-duration)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Test Plan](#test-plan)
- [Deprecation Plan](#deprecation-plan)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
  - [1.14](#114)
  - [1.15](#115)
  - [1.16](#116)
  - [1.17](#117)
  - [Not attached to a release milestone](#not-attached-to-a-release-milestone)
<!-- /toc -->

## Summary

This Kubernetes Enhancement Proposal (KEP) outlines the changes planned in the scope of an overhaul of all metrics instrumented in the main kubernetes/kubernetes repository. This is a living document and as existing metrics, that are planned to change are added to the scope, they will be added to this document. As this initiative is going to affect all current users of Kubernetes metrics, this document will also be a source for migration documentation coming out of this effort.

This KEP is targeted to land in Kubernetes 1.14. The aim is to get all changes into one Kubernetes minor release, to have only a migration be necessary. We are preparing a number of changes, but intend to only start merging them once the 1.14 development window opens.

## Motivation

A number of metrics that Kubernetes is instrumented with do not follow the [official Kubernetes instrumentation guidelines](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/instrumentation.md). This is for a number of reasons, such as the metrics having been created before the instrumentation guidelines were put in place (around two years ago), and just missing it in code reviews. Beyond the Kubernetes instrumentation guidelines, there are several violations of the [Prometheus instrumentation best practices](https://prometheus.io/docs/practices/instrumentation/). In order to have consistently named and high quality metrics, this effort aims to make working with metrics exposed by Kubernetes consistent with the rest of the ecosystem. In fact even metrics exposed by Kubernetes are inconsistent in themselves, making joining of metrics difficult.

Kubernetes also makes extensive use of a global metrics registry to register metrics to be exposed. Aside from general shortcomings of global variables, Kubernetes is seeing actual effects of this, causing a number of components to use `sync.Once` or other mechanisms to ensure to not panic, when registering metrics. Instead a metrics registry should be passed to each component in order to explicitly register metrics instead of through `init` methods or other global, non-obvious executions. Within the scope of this KEP, we want to explore other ways, however, it is not blocking for its success, as the primary goal is to make the metrics exposed themselves more consistent and stable.

While uncertain at this point, once cleaned up, this effort may put us a step closer to having stability guarantees for Kubernetes around metrics. Currently metrics are excluded from any kind of stability requirements.

### Goals

* Provide consistently named and high quality metrics in line with the rest of the Prometheus ecosystem.
* Consistent labeling in order to allow straightforward joins of metrics.

### Non-Goals

* Add/remove metrics. The scope of this effort just concerns the existing metrics. As long as the same or higher value is presented, adding/removing may be in scope (this is handled on a case by case basis).
* This effort does not concern logging or tracing instrumentation.

## Proposal

### cAdvisor instrumentation changes

#### Consistent labeling

Change the container metrics exposed through cAdvisor (which is compiled into the Kubelet) to [use consistent labeling according to the instrumentation guidelines](https://github.com/kubernetes/kubernetes/pull/69099). Concretely what that means is changing all the occurrences of the labels:
`pod_name` to `pod`
`container_name` to `container`

As Kubernetes currently rewrites meta labels of containers to “well-known” `pod_name`, and `container_name` labels, this code is [located in the Kubernetes source](https://github.com/kubernetes/kubernetes/blob/097f300a4d8dd8a16a993ef9cdab94c1ef1d36b7/pkg/kubelet/cadvisor/cadvisor_linux.go#L96-L98), so it does not concern the cAdvisor code base.

### Changing API latency histogram buckets

API server histogram latency buckets run from 125ms to 8s. This range does not accurately model most API server request latencies, which could run as low as 1ms for GETs or as high as 60s before hitting the API server global timeout.

https://github.com/kubernetes/kubernetes/pull/73638

### Kubelet metric changes

#### Make metrics aggregatable

Currently, all Kubelet metrics are exposed as summary data types. This means that it is impossible to calculate certain metrics in aggregate across a cluster, as summaries cannot be aggregated meaningfully. For example, currently one cannot calculate the [pod start latency in a given percentile on a cluster](https://github.com/kubernetes/kubernetes/issues/66791).

Hence, where possible, we should change summaries to histograms, or provide histograms in addition to summaries like with the API server metrics.

https://github.com/kubernetes/kubernetes/pull/72323

https://github.com/kubernetes/kubernetes/pull/72470

https://github.com/kubernetes/kubernetes/pull/73820

#### Prober metrics

Make prober metrics introduced in https://github.com/kubernetes/kubernetes/pull/61369 conform to the [Kubernetes instrumentation guidelines](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/instrumentation.md).

https://github.com/kubernetes/kubernetes/pull/76074

### Kube-scheduler metric changes

https://github.com/kubernetes/kubernetes/pull/72332

### Kube-proxy metric changes

#### Change proxy metrics to conform metrics guidelines

https://github.com/kubernetes/kubernetes/pull/72334

#### Clean the deprecated metrics which introduced in v1.14

https://github.com/kubernetes/kubernetes/pull/75023

### Kube-apiserver metric changes

#### Apiserver and etcd metrics

https://github.com/kubernetes/kubernetes/pull/72336

#### Fix admission metrics in true units

https://github.com/kubernetes/kubernetes/pull/72343

#### Remove the deprecated admission metrics

https://github.com/kubernetes/kubernetes/pull/75279

### Client-go metric changes

#### Workqueue metrics

Workqueue metrics need follow prometheus best practices and naming conventions.

* Instead of naming metrics based on the name of the workqueue, create metrics for workqueues that use name as a label.
* Use the recommended base units.
* Change summaries to histograms.

https://github.com/kubernetes/kubernetes/pull/71300

### Convert latency/latencies in metrics name to duration

https://github.com/kubernetes/kubernetes/pull/74418

### Risks and Mitigations

Risks include users upgrading Kubernetes, but not updating their usage of Kubernetes exposed metrics in alerting and dashboarding potentially causing incidents to go unnoticed.

To prevent this, we will implement recording rules for Prometheus that allow best effort backward compatibility as well as update uses of breaking metric usages in the [Kubernetes monitoring mixin](https://github.com/kubernetes-monitoring/kubernetes-mixin), a widely used collection of Prometheus alerts and Grafana dashboards for Kubernetes.

## Test Plan

Each individual change for this KEP must be accompanied with appropriate unit tests. As the scope of changes are provided are on the level of individual metrics, integration testing is not required.

## Deprecation Plan

In our efforts to change existing old metrics, we flag them `(Deprecated)` in the front of metrics help text.

These old metrics will be deprecated in v1.14 and coexist with the new replacement metrics. Users can use this release to change related monitoring rules and dashboards.

The release target of removing the deprecated metrics is v1.15.

Prior to removing deprecated metrics, we will attend appropriate community meetings (i.e. SIG Node) to provide sufficient notice.

## Graduation Criteria

All metrics exposed by components from kubernetes/kubernetes follow Prometheus best practices and (nice to have) tooling is built and enabled in CI to prevent simple violations of said best practices.

## Implementation History

As of release 1.17, this KEP is considered fully implemented.

### 1.14

- Use prometheus conventions for workqueue metrics [#71300](https://github.com/kubernetes/kubernetes/pull/71300)
  [@danielqsj](https://github.com/danielqsj) 2018-12-31
- Change scheduler metrics to conform metrics guidelines [#72332](https://github.com/kubernetes/kubernetes/pull/72332)
  [@danielqsj](https://github.com/danielqsj) 2019-01-14
- Change apiserver metrics to conform metrics guidelines [#72336](https://github.com/kubernetes/kubernetes/pull/72336)
  [@danielqsj](https://github.com/danielqsj) 2019-01-17
- Change proxy metrics to conform metrics guidelines [#72334](https://github.com/kubernetes/kubernetes/pull/72334)
  [@danielqsj](https://github.com/danielqsj) 2019-01-25
- Fix admission metrics in true units [#72343](https://github.com/kubernetes/kubernetes/pull/72343)
  [@danielqsj](https://github.com/danielqsj) 2019-01-28
- Adjust buckets in apiserver request latency metrics [#73638](https://github.com/kubernetes/kubernetes/pull/73638)
  [@wojtek-t](https://github.com/wojtek-t) 2019-02-04
- Change docker metrics to conform metrics guidelines [#72323](https://github.com/kubernetes/kubernetes/pull/72323)
  [@danielqsj](https://github.com/danielqsj) 2019-02-06
- Change kubelet metrics to conform metrics guidelines [#72470](https://github.com/kubernetes/kubernetes/pull/72470)
  [@danielqsj](https://github.com/danielqsj) 2019-02-18
- Rename cadvisor metric labels to match instrumentation guidelines [#69099](https://github.com/kubernetes/kubernetes/pull/69099)
  [@ehashman](https://github.com/ehashman) 2019-02-22
- Fit RuntimeClass metrics to prometheus conventions [#73820](https://github.com/kubernetes/kubernetes/pull/73820)
  [@haiyanmeng](https://github.com/haiyanmeng) 2019-02-22
- Convert latency/latencies in metrics name to duration [#74418](https://github.com/kubernetes/kubernetes/pull/74418)
  [@danielqsj](https://github.com/danielqsj) 2019-03-01
- Clean the deprecated metrics which introduced recently [#75023](https://github.com/kubernetes/kubernetes/pull/75023)
  [@danielqsj](https://github.com/danielqsj) 2019-03-07

### 1.15

- Remove the deprecated admission metrics [#75279](https://github.com/kubernetes/kubernetes/pull/75279)
  [@danielqsj](https://github.com/danielqsj) 2019-03-20
- Change kubelet probe metrics to counter [#76074](https://github.com/kubernetes/kubernetes/pull/76074)
  [@danielqsj](https://github.com/danielqsj) 2019-04-12

### 1.16

- Drop deprecated cadvisor metric labels [#80376](https://github.com/kubernetes/kubernetes/pull/80376)
  [@ehashman](https://github.com/ehashman) 2019-08-14

### 1.17

- Turn off apiserver deprecated metrics [#83837](https://github.com/kubernetes/kubernetes/pull/83837)
  [@RainbowMango](https://github.com/RainbowMango) 2019-11-16
- Turn off kubelet deprecated metrics [#83841](https://github.com/kubernetes/kubernetes/pull/83841)
  [@RainbowMango](https://github.com/RainbowMango) 2019-12-09

### Not attached to a release milestone

- Introduce promlint to guarantee metrics follow Prometheus best practices [#86477](https://github.com/kubernetes/kubernetes/pull/86477)
  [@RainbowMango](https://github.com/RainbowMango) 2020-05-25
