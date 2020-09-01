---
title: Log tracking for K8s component log

authors:
 - "@hase1128"
 - "@KobayashiD27"
 - "@fenggw-fnst"
 - "@zhijianli88"
 - "@Hellcatlk"
owning-sig: sig-instrumentation
participating-sigs:
reviewers:
 - @dashpole
 - @serathius
approvers:
 - @dashpole
editor: TBD
creation-date: 2020-09-01
last-updated: 2020-09-01
status: provisional
---

# Log tracking for K8s component log

## Table of Contents

<!-- toc -->
 - [Summary](#summary)
 - [Motivation](#motivation)
   - [Use Case 1](#use-case-1)
   - [Use Case 2](#use-case-2)
   - [Goals](#goals)
   - [Non-Goals](#non-goals)
 - [Proposal](#proposal)
   - [Logging metadata](#logging-metadata)
   - [Prerequisite](#prerequisite)
   - [Design of ID propagation (incoming request to webhook)](#design-of-id-propagation-incoming-request-to-webhook)
   - [Design of Mutating webhook](#design-of-mutating-webhook)
   - [Design of ID propagation (controller)](#design-of-id-propagation-controller)
 - [Test Plan](#test-plan)
 - [Migration / Graduation Criteria](#migration--graduation-criteria)
   - [Alpha](#alpha)
   - [Beta](#beta)
   - [GA](#ga)

<!-- /toc -->

## Summary

This KEP proposes a method for adding new three unique logging meta-data into K8s component logs.
It makes us more easy to identify specific logs related to an user request (such as `kubectl apply`) and object (such as Pod, Deployment).
It is expected to reduce investigation cost greatly when trouble shoothing.

### New three unique logging meta-data

We use three meta-data. These meta-data have different features and are used for troubleshooting from different perspectives.

| meta-data name | feature |
| ------ | ------ |
| trace-id | spans an user request. unique for user's request |
| span-id | spans a controller action. unique for controller action |
| initial-trace-id | spans the entire object lifecycle. unique for related objects |

### Note

This KEP is **how** a component could add meta-data to logs. To actually add meta-data to K8s component logs, the following procedure is necessary in addition.
- Open issues for each component, and discuss them with the SIGs that own that component.
- After get agreement, utilize this KEP's feature to change the source code that outputs log to add meta-data into these logs.
Please note that this KEP alone does not change the log format(does not add meta-data to logs).

## Motivation

Tracking logs among each Kubernetes component related to specific an user operation and objects is very tough work.
It is necessary to match logs by basically using timestamps and object's name as hints.
If multiple users throw many API requests at the same time, it is very difficult to track logs across each Kubernetes component log.

### Use Case 1

Suspicious user operation(e.g. unknown pod operations) or cluster processing(e.g. unexpected pod migration to another node) is detected.
Users want to get their mind around the whole picture and root cause.
As part of the investigation, it may be necessary to scrutinize the relevant logs of each component in order to figure out the series of cluster processing.
It takes long time to scrutinize the relevant logs without this log tracking feature, because component logs are independent of each other, and it is difficult to find related logs and link them.

This is similar to the [Auditing](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/), except for the following points.

 - Audit only collects information about http request sending and receiving in kube-apiserver, so it can't track internal work of each component.
 - Audit logs can't be associated to logs related to user operation (kubectl operation), because auditID is different for each http request.

### Use Case 2

Failed to attach PV to pod
Prerequisite: It has been confirmed that the PV has been created successfully.
In this case, the volume generation on the storage side is OK, and there is a possibility that the mount process to the container in the pod is NG.
In order to identify the cause, it is necessary to look for the problem area while checking the component (kubelet) log as well as the system side syslog and mount related settings.

This log tracking feature is useful to identify the logs related to specific user operation  and cluster processing, and can reduce investigation cost in such cases.

### Summary of Cases

 - Given a component log(such as error log), find the API request that caused this (error) log.
 - Given an API Request(such as suspicious API request), find the resulting component logs.

### Goals

 - Implement method which propagates new logging meta-data among each K8s component
 - Design and implement so as not to interfere with [Tracing KEP](https://github.com/kubernetes/enhancements/pull/1458)
   - e.g. implement of initial-trace-id, adding trace-id to object annotation executed in mutating webhook, etc.

### Non-Goals

 - Add new logging metadata into actual K8s component logs
   - This task will be done by opening  issues after completing this KEP
 - To centrally manage the logs of each Kubernetes component with Request-ID (This can be realized with existing OSS such as Kibana, so no need to implement into Kubernetes components).

## Proposal

### Logging metadata

We use three logging meta-data, and propagate them each K8s component by using OpenTelemetry.
OpenTelemetry has SpanContext which is used for propagation of K8s component.

| meta-data name | feature |
| ------ | ------ |
| trace-id | We use SpanContext.TraceID as trace-id<br>trace-id spans an user request.<br>trace-id is unique for user's request |
| span-id | We use SpanContext.SpanID as span-id<br>span-id spans a controller action.<br>span-id is unique for controller action |
| initial-trace-id | We implement new id(InitialTraceID) to SpanContext<br>We use SpanContext.InitialTraceID as initial-trace-id<br>initial-trace-id spans the entire object lifecycle. <br> initial-trace-id is unique for related objects |

All of three id's inception is from object creation and it dies with object deletion

### Prerequisite
We need to consider three cases:
- Case1: Requests from kubectl that creating an object
- Case2: Requests from kubectl other than creating (e.g. updating, deleting) an object
- Case3: Requests from controllers

The design below is based on the above three cases

### Design of ID propagation (incoming request to webhook)

**1. Incoming request to apiserver from kubectl or controller**
- For request from kubectl, request's header does not have trace-id, span-id or initial-trace-id
- For request from controller, request's header has trace-id, span-id and initial-trace-id

**2. Preprocessing handler (othttp handler)**  
2.1 Do othttp's original Extract(), and get SpanContext
- For request from kubectl, result is null (no trace-id, span-id, initial-trace-id)
- For request from controller we can get trace-id, span-id and initial-trace-id  
2.2 Create/Update SpanContext
- For request from kubectl
  - Since we don't get any SpanContext, do StartSpan() to start new trace (new trace-id and span-id)
  - the new SpanContext will be saved in the request's context "r.ctx"
- For request from controller
  - Since we get SpanContext, do StartSpanWithRemoteParent() to update the SpanContext (new span-id)
  - the updated SpanContext will be saved in the request's context "r.ctx"

**3. Creation handler**  
3.1 do our new Extract() to get initial-trace-id from request header to a golang ctx
- For request from kubectl we can't get initial-trace-id
- For request from controller we can get initial-trace-id  
3.2 get SpanContext from r.ctx to golang ctx

Notice that in this creation handler, the request will be consumed, so we need golang ctx to carry our information for propagation in apiserver.

**4. Make new request for sending to webhook**  
4.1 call othttp's original Inject() to inject the trace-id and span-id from golang ctx to header
4.2 call our new Inject() to inject the initial-trace-id from golang ctx to header  
- For request from kubectl we don't have initial-trace-id, so do nothing
- For request from controller we can do this

the order above(4.1 and 4.2) does not matter

### Design of Mutating webhook
check the request's header
- if there is initial-trace-id, add trace-id, span-id and initial-trace-id to annotation (This is the case for requests from controller.)
- if there is no initial-trace-id, check the request's operation
  - if operation is create, copy the trace-id as initial-trace-id, and add trace-id, span-id and initial-trace-id to annotation (This is the case for requests from kubectl create.)
  - if operation is not create, add trace-id, span-id to annotation (This is the case for requests from kubectl other than create.)

### Design of ID propagation (controller)
When controllers create/update/delete an object A based on another B, we propagate context from B to A. E.g.:
```
    ctx = traceutil.WithObject(ctx, objB)
    err = r.KubeClient.CoreV1().Create(ctx, objA...)
```
We do propagation across objects without adding traces to that components.

### Test Plan
TBD

### Migration / Graduation Criteria

#### Alpha
TBD

#### Beta
TBD

#### GA
TBD
