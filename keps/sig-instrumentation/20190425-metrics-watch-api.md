---
title: Watch support for metrics APIs
authors:
  - "@x13n"
owning-sig: sig-instrumentation
participating-sigs:
  - sig-autoscaling
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-04-25
last-updated: 2019-04-29
status: provisional
see-also:
replaces:
superseded-by:
---

# Metrics API watch support

## Table of Contents

<!-- toc -->
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [User Stories](#user-stories)
      - [HPA](#hpa)
      - [Custom metrics provider](#custom-metrics-provider)
    - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
    - [Test Plan](#test-plan)
    - [Graduation Criteria](#graduation-criteria)
    - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Version Skew Strategy](#version-skew-strategy)
  - [Implementation History](#implementation-history)
  - [Drawbacks [optional]](#drawbacks-optional)
- [Related resources](#related-resources)
<!-- /toc -->

## Summary

Provide watch capability to all resource metrics APIs: `metrics.k8s.io`,
`custom.metrics.k8s.io` and `external.metrics.k8s.io`, [similarly to regular
Kubernetes APIs](https://kubernetes.io/docs/reference/using-api/api-concepts/#efficient-detection-of-changes).

## Motivation

There are three APIs for serving metrics. All of them support reading the
metrics in a request-response manner, forcing the clients (e.g. HPA) interested
in up-to-date values to poll in a loop. This introduces additional latency:
between the time when new metric values are known to the process serving the
API and the time when a client interested in reading them actually fetches the
data.

### Goals

- Allow resource metrics clients to subscribe to stream metric changes.

### Non-Goals

- Graduate the APIs to GA. This needs to be done eventually, but is out of scope
  for this work.

## Proposal

This proposal is essentially about implementing [Efficient detection of
changes](https://kubernetes.io/docs/reference/using-api/api-concepts/#efficient-detection-of-changes)
for metrics. GET requests will accept additional `watch` parameter, which would
cause the API to start streaming results. Since old metric values are never
modified, the only supported update type will be `ADDED`, when a new data point
appears. Metrics don't contain any resourceVersion associated with them, so it
won't be possible to retrieve old values by passing `resourceVersion` parameter.
Instead, this parameter will be ignored and all recent data points will be
returned instead. This means metrics APIs will never return `410 Gone` error
code.

### User Stories

There are two sides to that proposal: the API producers and consumers. Examples
below include one consumer (HPA) and one hypothetical producer.

#### HPA

As an autoscaling solution, I will be able to subscribe to updates on a certain
labelSelector and get new metrics as soon as they are known to the metric
backend.

#### Custom metrics provider

As a metrics provider, I will be able to provide a low-latency Metrics API
implementation by taking advantage of backend specific features (e.g. streaming
APIs or known best polling interval).

### Implementation Details/Notes/Constraints [optional]

TBD:
What are the caveats to the implementation?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they releate.

### Risks and Mitigations

TBD: preventing producers/consumers from breaking due to the API version change.

## Design Details

### Test Plan

TBD.

**Note:** *Section not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.
Anything that would count as tricky in the implementation and anything particularly challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage expectations).
Please adhere to the [Kubernetes testing guidelines][testing-guidelines] when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md

### Graduation Criteria

Since the APIs already exist in beta and the change is backwards-compatible,
this proposal will be applied to the beta API, updating it from v1beta$(n) to
v1beta$(n+1).

Stability of this feature proven by at least one backend implementation for
`metrics.k8s.io` and `custom.metrics.k8s.io` will be a blocker for graduating
these APIs to v1.

This stability will be measured by e2e tests that will fetch the data using
watch.

### Upgrade / Downgrade Strategy

TBD.

If applicable, how will the component be upgraded and downgraded? Make sure this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to make use of the enhancement?

### Version Skew Strategy

TBD.

If applicable, how will the component handle version skew with other components? What are the guarantees? Make sure
this is in the test plan.

Consider the following in developing a version skew strategy for this enhancement:
- Does this enhancement involve coordinating behavior in the control plane and in the kubelet? How does an n-2 kubelet without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI, CRI or CNI may require updating that component before the kubelet.

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded

## Drawbacks [optional]

No custom metrics backend today offers a streaming API that would allow a
straightforward implementation of the watch. However, the fact that Kubernetes
metrics APIs will support streaming with watch may encourage some backends to
add such support. Additionally, the polling frequency will be specific to
relevant adapters, rather than to the metrics client.

# Related resources

SIG instrumentation discussions:
- [Custom/External Metrics API watch](https://groups.google.com/forum/#!topic/kubernetes-sig-instrumentation/nJvDyIwDgu8)
- [Resource Metrics API watch](https://groups.google.com/d/msg/kubernetes-sig-instrumentation/_b6c0oyPLJA/Y4rMQTBDAgAJ)
