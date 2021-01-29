# KEP-1668: Trace Context Propagation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Definitions](#definitions)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Architecture](#architecture)
  - [Trace context propagation](#trace-context-propagation)
  - [Mutating admission webhook](#mutating-admission-webhook)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [In-tree changes](#in-tree-changes)
    - [Trace Utility Package](#trace-utility-package)
    - [Add Go context to parameter list](#add-go-context-to-parameter-list)
  - [Out-of-tree changes](#out-of-tree-changes)
    - [Mutating webhook](#mutating-webhook)
  - [Behaviors with and without Mutating webhook](#behaviors-with-and-without-mutating-webhook)
    - [with Mutating webhook](#with-mutating-webhook)
    - [without Mutating webhook](#without-mutating-webhook)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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
- [ ] (R) Production readiness review approved
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

This KEP proposes to propagate trace context across components and across a series of related objects originating from an user request. It lays the foundation for enhancing relevant but scattered logs with the trace ID as common an identifier.

## Motivation

Current logging for a series of related messages lacks common identifiers that can be commonly found in other distributed systems. (such as [Global Request IDs](https://specs.openstack.org/openstack/oslo-specs/specs/pike/global-req-id.html)  in openstack) This makes debugging, auditing, reproducing problems, analyzing root cause via logs across components hard, administrators and developers have to match logs by basically using timestamps and object's name as hints which may takes a huge cost especially in a scenario with a large number of requests occur in a short period of time.

### Definitions

**Span**: The smallest unit of a trace.  It has a start and end time, and is attached to a single trace.

**Trace**: A collection of Spans which represents a single process.

**Trace Context**: A reference to a Trace that is designed to be propagated across component boundaries.  Sometimes referred to as the "Span Context".  It is can be thought of as a pointer to a parent span that child spans can be attached to.

### Goals

- Trace context received by the API Server as part of [API Server Tracing](https://github.com/kubernetes/enhancements/issues/647) can be propagated to kubernetes components
- A series of related objects originating from an user request can be associated by trace ID

### Non-Goals

- Generate new trace context(Span)
- Replace/change existing logging, metrics, or the events API
- Add additional telemetry to any components which is already done by [API Server Tracing](https://github.com/kubernetes/enhancements/issues/647). 
- Run any additional OpenTelemetry components (such as the OpenTelemetry collector, which the  [API Server Tracing](https://github.com/kubernetes/enhancements/issues/647) KEP uses)

## Proposal

### Architecture

### Trace context propagation

To link work done across components as belonging to the same action(user request), we must pass trace context across process boundaries. In traditional distributed systems, this context can be passed down through RPC metadata or HTTP headers. Kubernetes, however, due to its watch-based nature, requires us to attach trace context directly to the target object.

In this proposal, we choose to propagate this trace context as object annotations called `trace.kubernetes.io/context`

###  Mutating admission webhook

For trace context to be correlated as part of the same action, we must extract the trace context from the incomming request and embed it in target objects. To accomplish this, we have introduced an [out-of-tree mutating admission webhook](https://github.com/Hellcatlk/mutating-trace-admission-controller/tree/trace-ot).

The proposed in-tree changes will utilize the span context annotation injected into objects with this webhook.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

This design is inspired by the earlier KEP [Leveraging Distributed Tracing to Understand Kubernetes Object Lifecycles](https://github.com/kubernetes/enhancements/pull/650)

### In-tree changes

#### Trace Utility Package

This package will be able to retrieved span from the span context embedded in the `trace.kubernetes.io/context` object annotation. This package will facilitate propagating traces through kubernetes objects. The exported functions include:

```go
// WithObject returns a context attached with a Span retrieved from object annotation, it doesn't start a new span
func WithObject(ctx context.Context, obj meta.Object) (context.Context, error)
```

When controllers create/update/delete an object A based on another B, we propagate context from B to A. Below is an example to show how deployment propagate trace context  to replicaset.

```diff
 func (dc *DeploymentController) getNewReplicaSet(d *apps.Deployment, rsList, oldRSs []*apps.ReplicaSet, createIfNotExisted bool) (*apps.ReplicaSet, error) {
+       ctx := httptrace.WithObject(context.Background(), d)
        existingNewRS := deploymentutil.FindNewReplicaSet(d, rsList)
@@ -220,7 +227,8 @@ func (dc *DeploymentController) getNewReplicaSet(d *apps.Deployment, rsList, old
        // hash collisions. If there is any other error, we need to report it in the status of
        // the Deployment.
        alreadyExists := false
-       createdRS, err := dc.client.AppsV1().ReplicaSets(d.Namespace).Create(context.TODO(), &newRS, metav1.CreateOptions{})
+       createdRS, err := dc.client.AppsV1().ReplicaSets(d.Namespace).Create(ctx, &newRS, metav1.CreateOptions{})
```

#### Add Go context to parameter list
In OpenTelemetry's Go implementation,  span context is passed down through Go context. This will necessitate the threading of context across more of the Kubernetes codebase, which is a [desired outcome regardless](https://github.com/kubernetes/kubernetes/issues/815). In alpha stage,  we need to change some APIs by adding `ctx context.Context` to parameter list whose parameters doesn't contain context.Context yet. Below APIs will be impacted so far.

| APIs                          | file name                                                    |
| ----------------------------- | ------------------------------------------------------------ |
| createPods()                  | pkg/controller/controller_utils.go                           |
| CreatePodsWithControllerRef() | pkg/controller/controller_utils.go<br />pkg/controller/replication/conversion.go<br />pkg/controller/daemon/daemon_controller.go<br />pkg/controller/replication/conversion.go |


### Out-of-tree changes

#### Mutating webhook
We use mutating admission controller(aka webhook)  to change/update the object annotation. It takes advantages of:

- Ease of use. Using client-go with a context.Context is easier than adding an annotation. The webhook takes care of writing the annotation.
- Object to object context propagation. Without the mutating admission controller, we can only associate actions from a single object. With the mutating admission controller, the logging metadata would be added for objects modified by controllers of the initial object (e.g. metadata added to a deployment annotation would appear in pod logs).

This mutating admission webhook extracts  a `span context` from incoming request, and then stores it into object annotation`trace.kubernetes.io/context` with base64 encoded version of [this wire format](https://github.com/census-instrumentation/opencensus-specs/blob/master/encodings/BinaryEncoding.md#trace-context). The webhook can be configured to inject context into only target object types.

below is a key/value pair example in object annotation :

| key               | value(encoded)                       | origin value(decoded)                                   | description                                                  |
| ---------------------------- | ------------------------------------------------------- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| trace.kubernetes.io/context | 0/3kO3imLJzu3N54RTuOEUDbc2C0poIMAA== | 00-ca057eae1a26b66314fe3e361eedc5ca-3696483da6bfdcea-00 |it consists of `<version>-<traceid>-<spanid>-<flag>`. <br />A corresponding http header field is like `traceparent: 00-ca057eae1a26b66314fe3e361eedc5ca-3696483da6bfdcea-00` which is a w3c specific [trace-context](https://w3c.github.io/trace-context/#traceparent-header ). <br />|

### Behaviors with and without Mutating webhook
Since the mutating webhook is optional for users, we will explain the different behaviors between with and without mutating webhook.

#### with Mutating webhook

**kubectl request**:

- APIServer uses otel to start a new Span
- APIServer uses otel to propagate `span context` to the other end(webhook)
- Webhook persists `span context` to object

**controllers request:**

- Controller uses otel to start a related Span, which connected to the `span context` in object
- Controller uses otel to propagate `span context`  to the other end(APIServer)
- APIServer uses otel  to start a related Span, which connected to the `span context` from the  incoming request
- APIServer uses otel to propagate `span context`  to the other end(webhook)
- Webhook persists `span context` to object

#### without Mutating webhook

**kubectl request:**

- APIServer uses otel to start a new Span
- APIServer uses otel to propagate `span context` to the other end
- ~~Webhook persists `span context` to object~~

**controllers request:**

- Controller start uses otel to start a new Span
- Controller uses otel to propagate `span context`  to the other end(APIServer)
- APIServer uses otel  to start related Span, which connected to the `span context` from the  incoming request
- APIServer uses otel to propagate `span context` to the other end
- ~~Webhook persists `span context`  to object~~

In short, the webhook decides whether to add `span context` to the object.

### Test Plan

All added code will be covered by unit tests.

### Graduation Criteria

#### Alpha

- Feature covers 3 important workload objects: Deployment, Statefulset, Daemonset
- Related unit tests described in this KEP are completed

#### Beta

- Feature covers other objects which not limited to ownerRef relationship
- All necessary tests are completed

#### GA

- Feedback about this feature is collected and addressed
- Enabled in Beta for at least two releases without complaints

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.

-->

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: PropagateContextTrace
    - Components depending on the feature gate: kube-controller-manager
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**

  In apiserver, new request handlers added by this feature will generate
  or update the trace context , then the trace context will be added to the
  object's annotation by the webhook provided by this feature.

  In controller-manager, when sending request to apiserver, it will get the
  trace context from the referenced object's annotation and inject the trace context into the
  outgoing request header with the W3C format.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
Yes

* **What happens if we reenable the feature if it was previously rolled back?**
  Objects created during the rollback will have no trace context until they
  are recreated.

* **Are there any tests for feature enablement/disablement?**
  Unit test can ensure that the feature enablement/disablement is valid

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
   It will not have impact  on running workloads.

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
  N/A

* **Will enabling / using this feature result in introducing new API types?**
  N/A

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
  N/A

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  N/A

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  N/A

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  TBD

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
  The feature will be unavailable.

* **What are other known failure modes?**
  TBD

* **What steps should be taken if SLOs are not being met to determine the problem?**
  N/A

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History
* 2020-09-01: KEP proposed
* 2020-09-28: PRR questionnaire updated
* [Mutating admission webhook which injects trace context for demo](https://github.com/Hellcatlk/mutating-trace-admission-controller/tree/trace-ot)
* [Instrumentation of Kubernetes components for demo](https://github.com/Hellcatlk/kubernetes/pull/1)
* [Instrumentation of Kubernetes components for demo based on KEP647](https://github.com/Hellcatlk/kubernetes/pull/3)
* refactor [Log tracking](https://github.com/kubernetes/enhancements/pull/1961) KEP to Trace Context Propagation
<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
