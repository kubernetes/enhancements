# Log tracking for K8s component log

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
  - [New three unique logging metadata](#new-three-unique-logging-metadata)
  - [Opt-in for Components](#opt-in-for-components)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Background](#background)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Summary of Cases](#summary-of-cases)
- [Design Details](#design-details)
  - [Prerequisite](#prerequisite)
  - [Design of ID propagation (controller)](#design-of-id-propagation-controller)
  - [Design of Mutating webhook(Out of tree)](#design-of-mutating-webhookout-of-tree)
  - [Behaviors with and without Mutating webhook](#behaviors-with-and-without-mutating-webhook)
    - [with Mutating webhook](#with-mutating-webhook)
    - [without Mutating webhook](#without-mutating-webhook)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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

## Summary

This KEP proposes a method for adding new three unique logging metadata into annotation of objects and propagating them across components. Then the K8S components have ability to *track/filter* logs of all relevant object sets in need that makes us more easy to identify specific logs related to an user request (such as `kubectl apply`) and object (such as Pod, Deployment).
It is expected to reduce investigation cost greatly when troubleshooting.

### New three unique logging metadata

We use three metadata. These metadata have different features and are used for troubleshooting from different perspectives.

| metadata name | feature |
| ------ | ------ |
| traceid | spans an user request. unique for user's request |
| spanid | spans a controller action. unique for controller actions |
| objsetid | spans the entire object lifecycle. unique for a related object set |

### Opt-in for Components

Updating/changing the logs is outside the scope of this KEP.  To actually add metadata to K8s component logs, the following procedures are needed in addition.
- Open issues for each component, and discuss them with the SIGs that own that component.
- After get agreement, utilize this KEP's feature to change the source code that outputs log to add metadata into these logs.

## Motivation

Tracking logs among each Kubernetes component related to specific an user operation and objects is very tough work.
It is necessary to match logs by basically using timestamps and object's name as hints.
If multiple users throw many API requests at the same time, it is very difficult to track logs across each Kubernetes component log.

### Goals

 - Implement method which propagates new logging metadata among each K8s components
 - Store and update logging metadata into object annotation through mutating admission controller(aka Webhook)

### Non-Goals

 - Add new logging metadata into actual K8s component logs
   - This task will be done by opening  issues after completing this KEP
 - To centrally manage the logs of each Kubernetes component with objsetid(This can be realized with existing OSS such as Kibana, so no need to implement into Kubernetes components).
 - This proposal does not add additional telemetry to any components, just context-based metadata to existing logs. This KEP doesn't require running any additional OpenTelemetry components (such as the OpenTelemetry collector, which the  [API Server Tracing](https://github.com/kubernetes/enhancements/issues/647) KEP uses)

## Background
It's Known that all requests to K8s will reach API Server first, in order to propagate these metadata to mutating admission controller, we require API Server to support propogating these metadata as well. Fortunately there is already a KEP [API Server Tracing](https://github.com/kubernetes/enhancements/issues/647)  to do this.

```
Goals
The API Server generates and exports spans for incoming and outgoing requests.
The API Server propagates context from incoming requests to outgoing requests.
```
So this KEP relies on KEP [API Server Tracing](https://github.com/kubernetes/enhancements/issues/647)

## Proposal

### User Stories (Optional)

#### Story 1

Suspicious user operation(e.g. unknown pod operations) or cluster processing(e.g. unexpected pod migration to another node) is detected.
Users want to get their mind around the whole picture and root cause.
As part of the investigation, it may be necessary to scrutinize the relevant logs of each component in order to figure out the series of cluster processing.
It takes long time to scrutinize the relevant logs without this log tracking feature, because component logs are independent of each other, and it is difficult to find related logs and link them.

This is similar to the [Auditing](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/), except for the following points.

 - Audit only collects information about http request sending and receiving in kube-apiserver, so it can't track internal work of each component.
 - Audit logs can't be associated to logs related to user operation (kubectl operation), because auditID is different for each http request.

#### Story 2

Failed to attach PV to pod
Prerequisite: It has been confirmed that the PV has been created successfully.
In this case, the volume generation on the storage side is OK, and there is a possibility that the mount process to the container in the pod is NG.
In order to identify the cause, it is necessary to look for the problem area while checking the component (kubelet) log as well as the system side syslog and mount related settings.

This log tracking feature is useful to identify the logs related to specific user operation  and cluster processing, and can reduce investigation cost in such cases.
### Summary of Cases

 - Given a component log(such as error log), find the API request that caused this (error) log.
 - Given an API Request(such as suspicious API request), find the resulting component logs.


## Design Details

### Prerequisite

We need to consider three cases:
- Case1: Requests from kubectl that creating an object
- Case2: Requests from kubectl other than creating (e.g. updating, deleting) an object
- Case3: Requests from controllers

The design below is based on the above three cases

The following picture show our design![design](./overview.png)

we don't have any modifications in kubectl in this design.

We use three logging metadata, and propagate them across each K8s component by using [SpanContext](https://pkg.go.dev/go.opentelemetry.io/otel@v0.15.0/trace#SpanContext) and [Baggage](https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/baggage/api.md)  of OpenTelemetry.


### Design of ID propagation (controller)
In controller side, we don't update the object annotation any more, instead, we just focus on how to propagate the new logging metadata across objects. And with the help of KEP [API Server Tracing](https://github.com/kubernetes/enhancements/issues/647) , the new logging metadata can be propagated across K8s components.

**1. New context propagation library function**

A new API `SpanContextWithObject` is introduced to return a `context.Context` which includes a Span and a Baggage according to the passed object.

```
func SpanContextWithObject(ctx context.Context, meta metav1.Object) context.Context
```
it does something like below pseudo code:
```
// SpanContextWithObject returns a `context.Context` which includes a Span and a Baggage
func SpanContextWithObject(ctx context.Context, meta metav1.Object) context.Context{
	// 1. extract baggage from annotation and chain it to ctx
	// 2. extract SpanContext from annotation
	// 3. generate Span from SpanContext
	// 4. chain Span to ctx which is required by propagators.Inject
}
```
**2. Propagate Span and Baggage with context.Context across objects**

When controllers create/update/delete an object A based on another B, we propagate context from B to A. Below is an example to show how deployment propagate Span and Baggage to replicaset.

```diff
 func (dc *DeploymentController) getNewReplicaSet(d *apps.Deployment, rsList, oldRSs []*apps.ReplicaSet, createIfNotExisted bool) (*apps.ReplicaSet, error) {
+       ctx := httptrace.SpanContextWithObject(context.Background(), d)
        existingNewRS := deploymentutil.FindNewReplicaSet(d, rsList)
@@ -220,7 +227,8 @@ func (dc *DeploymentController) getNewReplicaSet(d *apps.Deployment, rsList, old
        // hash collisions. If there is any other error, we need to report it in the status of
        // the Deployment.
        alreadyExists := false
-       createdRS, err := dc.client.AppsV1().ReplicaSets(d.Namespace).Create(context.TODO(), &newRS, metav1.CreateOptions{})
+       createdRS, err := dc.client.AppsV1().ReplicaSets(d.Namespace).Create(ctx, &newRS, metav1.CreateOptions{})
```
In order to propagate context across all objects we concerned , we also need to change some APIs by adding `ctx context.Context` to parameter list whose parameters doesn't contain context.Context yet. Below APIs will be impacted so far.

| APIs                          | file name                                                    |
| ----------------------------- | ------------------------------------------------------------ |
| createPods()                  | pkg/controller/controller_utils.go                           |
| CreatePodsWithControllerRef() | pkg/controller/controller_utils.go<br />pkg/controller/replication/conversion.go<br />pkg/controller/daemon/daemon_controller.go<br />pkg/controller/replication/conversion.go |

**3. How to log the metadata from a context.Context**

Please note that changing the log is *not* the scope of this KEP. In the future, we can get the metadata and change the log like below accordingly.
- Example
```go
// package httptrace
func WithObjectV2(ctx context.Context, meta metav1.Object) context.Context {
    ctx = SpanContextWithObject(ctx, meta)

    log := kontext.FromContext(ctx)
    objsetID, traceID, spanID := ObjSetIDsFrom(ctx)
    log.WithValues("objectsetid", objsetID, "traceid", traceID, "spanid", spanID)
    return kontext.IntoContext(ctx, log)
}

// package deployment
func cleanupDeployment(oldRSs []*apps.ReplicaSet, deployment *apps.Deployment) {
    ctx := httptrace.WithObjectV2(context.TODO(), deployment)
    [...]
    logger := kontext.FromContext(ctx)
    logger.Infof("Looking to cleanup old replica sets for deployment %q", deployment.Name)
    cleanupFIXME(ctx, ...)
 }

func cleanupFIXME(ctx context.Context, ...){
    logger := kontext.FromContext(ctx)
    logger.Info("Looking to cleanup FIXME")
}
```
In this example, it's expected that the log looks like:

```shell
2019/12/01 14:49:12 [INFO] objectsetid="04bc5995-0147-4db8-9f06-fdbcb2e1a087" traceid="ca057eae1a26b66314fe3e361eedc5ca" spanid="3696483da6bfdcea" Looking to cleanup old replica sets for deployment hello-world
2019/12/01 14:49:12 [INFO] objectsetid="04bc5995-0147-4db8-9f06-fdbcb2e1a087" traceid="ca057eae1a26b66314fe3e361eedc5ca" spanid="3696483da6bfdcea" Looking to cleanup FIXME
```
the logger and specific logging metadata will be propagated across the call stack.

### Design of Mutating webhook(Out of tree)
We use mutating admission controller(aka webhook)  to change/update the object annotation. It takes advantages of:

- Ease of use. Using client-go with a context.Context is easier than adding an annotation. The webhook takes care of writing the annotation.
- Object to object context propagation. Without the mutating admission controller, we can only associate actions from a single object. With the mutating admission controller, the logging metadata would be added for objects modified by controllers of the initial object (e.g. metadata added to a deployment annotation would appear in pod logs).

**Logging metadata**

below table show how these 3 logging metadata map to the SpanContext and Baggage of OpenTelemetry.
| metadata name | feature |
| ------ | ------ |
| traceid | We use SpanContext.TraceID as traceid<br>traceid spans an user request.<br>traceid is unique for user's request |
| spanid | We use SpanContext.SpanID as spanid<br>spanid spans a controller action.<br>spanid is unique for controller action |
| objsetid | We implement new id(objsetid) to SpanContext<br>We use [Baggage](https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/baggage/api.md) to propagate objsetid<br/>We use UID of root object as objsetid<br>objsetid spans the entire object lifecycle. <br>objsetid is unique for related object set. |

All of three id's inception is from object creation and it dies with object deletion


These 3 logging metadata are stored  with 2 pairs of key/value in annotation.

| key                          | example value                                           | description                                                  |
| ---------------------------- | ------------------------------------------------------- | ------------------------------------------------------------ |
| trace.kubernetes.io/span     | 00-ca057eae1a26b66314fe3e361eedc5ca-3696483da6bfdcea-00 | it consists of `<version>-<traceid>-<spanid>-<flag>`. <br />A corresponding http header field is like `traceparent: 00-ca057eae1a26b66314fe3e361eedc5ca-3696483da6bfdcea-00` which is a w3c specific [trace-context](https://w3c.github.io/trace-context/#traceparent-header ). <br />2 metadata(`traceid` and `spanid`) are stored here. |
| trace.kubernetes.io/objsetid | 04bc5995-0147-4db8-9f06-fdbcb2e1a087                    | A corresponding http header field is like `baggage: objectSetID=04bc5995-0147-4db8-9f06-fdbcb2e1a087` which is a w3c specific [Baggage](https://w3c.github.io/baggage/#examples-of-http-headers).<br />It stores metadata `objsetid`. |

Roughly this webhook do something like below:

**1.Extract SpanContext and objsetid from request's header**

**2.Update SpanContext and objsetid to object annotation**

- For traceid and spanid and traceflags(sampled/not sampled)

    Always update `trace.kubernetes.io/span` annotation based on the incoming request.

- For objsetid

    if objsetid is nil,  Use the UID of object as objsetid, else, leave objsetid as it is. Update it to `trace.kubernetes.io/objsetid` in need.

### Behaviors with and without Mutating webhook
Since the mutating webhook is optional for users, we will explain the different behaviors between with and without mutating webhook.

#### with Mutating webhook

**kubectl request**:

- APIServer uses otel to start a new Span
- APIServer uses otel to propagate SpanContext to the other end(webhook)
- Webhook generates an objsetid
- Webhook persists SpanContext and objsetid to object

**controllers request:**

- Controller uses otel to start a related Span, which connected to the SpanContext in object
- Controller uses otel to propagate Baggage(objsetid)  stored in object and SpanContext  to the other end(APIServer)
- APIServer uses otel  to start a related Span, which connected to the SpancContext from the  incoming request
- APIServer uses otel to propagate SpanContext and Baggage(objsetid) to the other end(webhook)
- Webhook persists SpanContext and objsetid to object

#### without Mutating webhook

**kubectl request:**

- APIServer uses otel to start a new Span
- APIServer uses otel to propagate SpanContext to the other end
- ~~Webhook generates an objsetid~~
- ~~Webhook persists SpanContext and objsetid to object~~

**controllers request:**

- Controller start uses otel to start a new Span
- Controller uses otel to propagate SpanContext  to the other end(APIServer)
- APIServer uses otel  to start related Span, which connected to the SpancContext from the  incoming request
- APIServer uses otel to propagate SpanContext to the other end
- ~~Webhook persists SpanContext and objsetid to object~~

In short, the webhook decides whether to add logging metadata to the object.

### Risks and Mitigations

TBD

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

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: ObjectTraceIDs
    - Components depending on the feature gate: kube-controller-manager
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**

  In apiserver, new request handlers added by this feature will generate
  or update the trace information (ids), then the ids will be added to the
  object's annotation by the webhook provided by this feature.

  In controller-manager, when sending request to apiserver, it will get the
  ids from the referenced object's annotation and inject the ids into the
  outgoing request header with the W3C format.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes.

* **What happens if we reenable the feature if it was previously rolled back?**
  Objects created during the rollback will have no objsetid until they
  are recreated.

* **Are there any tests for feature enablement/disablement?**
  Unit test can ensure that the feature enablement/disablement is valid.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  The proposed ids will not be added or updated. It will not have impact
  on running workloads.

* **What specific metrics should inform a rollback?**
  N/A

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  N/A

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  N/A

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  It can be determined by checking objects annotation with proposed ids.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  N/A

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  N/A

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  N/A

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  N/A

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

## Implementation History

* 2020-09-01: KEP proposed
* 2020-09-28: PRR questionnaire updated
* [Mutating admission webhook which injects trace context for demo](https://github.com/Hellcatlk/mutating-trace-admission-controller/tree/trace-ot)
* [Instrumentation of Kubernetes components for demo](https://github.com/Hellcatlk/kubernetes/pull/1)
* [Instrumentation of Kubernetes components for demo based on KEP647](https://github.com/Hellcatlk/kubernetes/pull/3)
