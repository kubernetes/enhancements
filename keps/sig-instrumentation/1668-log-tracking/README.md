# Log tracking for K8s component log

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
  - [New three unique logging meta-data](#new-three-unique-logging-meta-data)
  - [Note](#note)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Summary of Cases](#summary-of-cases)
  - [Logging metadata](#logging-metadata)
- [Design Details](#design-details)
  - [Prerequisite](#prerequisite)
  - [Design of ID propagation in apiserver](#design-of-id-propagation-in-apiserver)
  - [Design of ID propagation (controller)](#design-of-id-propagation-controller)
  - [Design of ID propagation in <a href="https://github.com/kubernetes/client-go">client-go</a>](#design-of-id-propagation-in-client-go)
  - [Design of Mutating webhook(Out of tree)](#design-of-mutating-webhookout-of-tree)
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

This KEP proposes a method for adding new three unique logging meta-data into K8s component logs.
It makes us more easy to identify specific logs related to an user request (such as `kubectl apply`) and object (such as Pod, Deployment).
It is expected to reduce investigation cost greatly when trouble shoothing.

### New three unique logging meta-data

We use three meta-data. These meta-data have different features and are used for troubleshooting from different perspectives.

| meta-data name | feature |
| ------ | ------ |
| traceid | spans an user request. unique for user's request |
| spanid | spans a controller action. unique for controller action |
| initialtraceid | spans the entire object lifecycle. unique for related objects |

### Note

This KEP is **how** a component could add meta-data to logs. To actually add meta-data to K8s component logs, the following procedure is necessary in addition.
- Open issues for each component, and discuss them with the SIGs that own that component.
- After get agreement, utilize this KEP's feature to change the source code that outputs log to add meta-data into these logs.
Please note that this KEP alone does not change the log format(does not add meta-data to logs).

## Motivation

Tracking logs among each Kubernetes component related to specific an user operation and objects is very tough work.
It is necessary to match logs by basically using timestamps and object's name as hints.
If multiple users throw many API requests at the same time, it is very difficult to track logs across each Kubernetes component log.

### Goals

 - Implement method which propagates new logging meta-data among each K8s component
 - Design and implement so as not to interfere with [Tracing KEP](https://github.com/kubernetes/enhancements/pull/1458)
   - e.g. implement of initialtraceid, adding traceid to object annotation executed in mutating webhook, etc.

### Non-Goals

 - Add new logging metadata into actual K8s component logs
   - This task will be done by opening  issues after completing this KEP
 - To centrally manage the logs of each Kubernetes component with Request-ID (This can be realized with existing OSS such as Kibana, so no need to implement into Kubernetes components).

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

### Logging metadata

We use three logging meta-data, and propagate them each K8s component by using OpenTelemetry.
OpenTelemetry has SpanContext and [Baggage](https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/baggage/api.md) which is used for propagation of K8s component.

| meta-data name | feature |
| ------ | ------ |
| traceid | We use SpanContext.TraceID as traceid<br>traceid spans an user request.<br>traceid is unique for user's request |
| spanid | We use SpanContext.SpanID as spanid<br>spanid spans a controller action.<br>spanid is unique for controller action |
| initialtraceid | We implement new id(InitialTraceID) to SpanContext<br>We use [Baggage](https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/baggage/api.md) to propagate initialtraceid<br/>We use UID of root object as initialtraceid<br>initialtraceid spans the entire object lifecycle. <br>initialtraceid is unique for related objects |

All of three id's inception is from object creation and it dies with object deletion

## Design Details

### Prerequisite

We need to consider three cases:
- Case1: Requests from kubectl that creating an object
- Case2: Requests from kubectl other than creating (e.g. updating, deleting) an object
- Case3: Requests from controllers

The design below is based on the above three cases

The following picture show our design![design](./overview.png)



we don't have any modifications in kubectl in this design.

### Design of ID propagation in apiserver

**1. Do othttp's original [Extract](https://pkg.go.dev/go.opentelemetry.io/otel/api/propagators#TraceContext.Extract)() to get [SpanContext](https://pkg.go.dev/go.opentelemetry.io/otel/api/trace#SpanContext)**

- For request from kubectl, SpanContext is null,  do [Start](https://github.com/open-telemetry/opentelemetry-go/blob/3a9f5fe15f50a35ad8da5c5396a9ed3bbb82360c/sdk/trace/tracer.go#L38)() to start new SpanContext (new traceid and spanid)
- For request from controller we can get a valid SpanContext(including traceid and spanid), do [Start](https://github.com/open-telemetry/opentelemetry-go/blob/3a9f5fe15f50a35ad8da5c5396a9ed3bbb82360c/sdk/trace/tracer.go#L38)() to update the SpanContext (new spanid)

**2. Chain SpanContext and initialtraceid to "r.ctx"**

- we use r.ctx to propagate those IDs to next handler

**3. Make up a new outgoing [request](#design-of-id-progagation-in-client-go)**

### Design of ID propagation (controller)

**1. Extract SpanContext and initialtraceid from annotation of object to golang ctx**

**2. Propagate golang ctx from objects to API Calls**

When controllers create/update/delete an object A based on another B, we propagate context from B to A. E.g.:
```
    ctx = httptrace.SpanContextFromAnnotations(context.Background(), objB.GetAnnotations())
    err = r.KubeClient.CoreV1().Create(ctx, objA...)
```
We do propagation across objects without adding traces to that components.

**3. Make up a new outgoing [request](#design-of-id-progagation-in-client-go)**

### Design of ID propagation in [client-go](https://github.com/kubernetes/client-go)
client-go  helps to inject [TraceContext](https://pkg.go.dev/go.opentelemetry.io/otel/propagators#example-TraceContext) and [Baggage](https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/baggage/api.md) to the outgoing http request header, something changes are like below:
```diff
@@ -868,6 +871,7 @@ func (r *Request) request(ctx context.Context, fn func(*http.Request, *http.Resp
                req = req.WithContext(ctx)
                req.Header = r.headers

+               props := otelhttptrace.WithPropagators(otel.NewCompositeTextMapPropagator(propagators.TraceContext{}, propagators.Baggage{}))
+               otelhttptrace.Inject(req.Context(), req, props)

                r.backoff.Sleep(r.backoff.CalculateBackoff(r.URL()))
                if retries > 0 {
```
apiserver and controller use the API to make up a new outgoing request.

### Design of Mutating webhook(Out of tree)

**1.Extract SpanContext and initialtraceid from request's header **

**2.Update SpanContext and initialtraceid to object**

- For traceid and spanid and traceflags(sampled/not sampled)

Always set the trace context annotation based on the incoming request.

- For initialtraceid

if initialtraceid is nil,  Use the UID of object as initialtraceid, else, leave initialtraceid as it is.


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
    - Components depending on the feature gate: kube-apiserver, kube-controller-manager
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
  Objects created during the rollback will have no initialtrace-id until they
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
