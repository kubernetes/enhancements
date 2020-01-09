---
title: Add Request-ID to each k8s component log
authors:
 - "@hase1128"
 - "@sshukun"
 - "@furukawa3"
 - "@vanou"
owning-sig: sig-auth
participating-sigs:
 - sig-auth
reviewers:
 - TBD
approvers:
 - TBD
editor: TBD
creation-date: 2019-11-01
last-updated: 2020-01-09
status: provisional
---

# Add Request-ID to each k8s component log

## Table of Contents

<!-- toc -->
 - [Summary](#summary)
 - [Motivation](#motivation)
   - [Target User](#target-user)
   - [Target User's objective](#target-users-objective)
   - [Case 1](#case-2)
   - [Case 2](#case-1)
   - [Goals](#goals)
   - [Non-Goals](#non-goals)
 - [Proposal](#proposal)

<!-- /toc -->

## Summary

This KEP proposes a new unique logging meta-data into all K8s logs. It makes us
more easy to identfy specfic logs related to a single API request (such as
`kubectl apply -f <my-pod.yaml>`). This feature is similar to
[Request-ID](https://docs.openstack.org/api-guide/compute/faults.html) for
OpenStack. It greatly reduces the investigation cost.

## Motivation

### Target User

Support team in k8s Service Provider

### Target User's objective

We'd like to resolve quickly for end users' problem.

Tracking logs among each k8s component related to specific an API request is
very tough work. It is necessary to match logs with timestamps as hints. If
multiple users throw many API requests at the same time, it is very difficult to
track logs across each k8s component log. It is difficult that target user can
not resolve end user's problem quickly in the above. Therefore, we'd like to add
a new identifier which is unique to each API request. This feature is useful for
the following use cases:

#### Case 1

In case of insecure or unauthorized operation happens, it is necessary to
identify what effect that operation caused. This proposed feature helps identify
what happened at each component or server by each insecure / unauthorized API
request. We can collect these logs as an evidence.

#### Case 2

If the container is terminated by OOM killer, there is a case to break down the
issue into parts(Pod or k8s) from the messages related OOM killer on host logs
and the API processing just before OOM killer. If the cause is that some unknown
pod creations, it is helpful to detect the root API request and who called this
request.

### Goals

Adding a Request-ID into all K8s component logs. The Request-ID is unique to an
operation.

### Non-Goals

To centrally manage the logs of each k8s component with `Request-ID` (This can
be realized with existing OSS such as Kibana, so no need to implement into K8s
components).

## Proposal

### Basic policy of Request-ID

 - Minimize the impact to existing features
 - Simple implementation
 - Collaborate with related KEPs. This means as follows.
   - Ensure consistent metadata(This may need to be considered and managed across the Kubernetes beyond the related KEPs).
   - Implementation which does not interfere related KEP's implementation

### Design Overview

Request-ID feature consists the following two features.
 - Export Request-ID to each kubernetes component log
 - Propagate Request-ID to related objects

#### Design overview of Export Request-ID

**Collaboration with related KEP**  
There is the existing KEP(structured logging feature) that related log exporting. 
Main concept of this feature is structuring the log format and replace existing klog with logr for structuring.
Structured format is attractive, however this replacement is very tough work and may be required troublesome migration steps.
In this situation, it is expect to take a long time to migration completely.
In the meantime, Request-ID feature takes more simple way to export ID information to log files.
(It is possibility that Request-ID feature may take structured logging feature in the future)

There is another related KEP(distributed tracing) that propagate Trace-ID(and etc.) to related kubernetes objects.
This feature adds these information(e.g. Trace-ID) to Annotations of objects.
Annotations is existing feature of kubernetes, and Request-ID feature adopts same method.

**Idea of design**  
As a result, exporting Request-ID is implementable by just reading Annotations from objects when using klog.
Note that this is no impact to existing klog feature.

##### Example of source code and log output

 - Original source code(scheduler.go)

```
func (sched *Scheduler) scheduleOne(ctx context.Context) {
	fwk := sched.Framework

	podInfo := sched.NextPod()
<snip>
	pod := podInfo.Pod
<snip>
	klog.V(3).Infof("Attempting to schedule pod: %v/%v", pod.Namespace, pod.Name)
```

 - Add Request-ID exportation into klog

```
func (sched *Scheduler) scheduleOne(ctx context.Context) {
	fwk := sched.Framework

	podInfo := sched.NextPod()
<snip>
	pod := podInfo.Pod
<snip>
	klog.V(3).Infof("Request-ID: %v Attempting to schedule pod: %v/%v", pod.<Annotations related Request-ID>, pod.Namespace, pod.Name)
```

 - Original log output
```
I1220 08:58:31.000196    6869 scheduler.go:564] Attempting to schedule pod: default/nginx

```

 - Request-ID log output
```
I1220 08:58:31.000196    6869 scheduler.go:564] Request-ID : d0ac7061-d9fc-43d0-957f-dbc7306d3ace Attempting to schedule pod: default/nginx

```

**Pros of this method**  
 - No impact to existing klog feature
 - As a result, no interfere with structured logging implementation

**Restricts**  
 - Only objects that can be obtained within the function scope calling klog can read annotations

#### Step to implementation of Export Request-ID

##### Step1

Target klogs: Only klogs that satisfy both of the following requirements.
 - klogs that are called during typical kubectl operations
   - e.g. kubectl operation: create, apply, delete, etc.
   - e.g. kubernetes object: deployment, pod, etc. 
 - klogs that can get object's annotation in the scope which calls the klog.

##### Step2

Expand the range of operations and resources from Step1.
   - e.g. kubectl operation: rollout, scale, drain, etc.
   - e.g. kubernetes object: service, secret, pv, etc. 

##### Step3(TBD.)

 - Considering a mechanism that can acquire annotations from any objects at any scopes, and then the target is klog that could not be done in Step1 and 2.

#### Design overview of Propagate Request-ID

**Collaboration with related KEP**  
There is an idea to use `distributed context` of the existing KEP(distributed tracing).
In this case, use of OpenTelemetry is prerequisite.
However, Request-ID feature does not need Exporter of OpenTelemetry because Request-ID is exported to only kubernetes log file by klog.
Request-ID collaborates with distributed tracing KEP in terms of adding context(e.g. Trace-ID such as Request-ID) to Annotations and propagation feature(distributed context of OpenTelemetry).
At first(in Alpha stage), OpenTelemetry is prerequisite of Request-ID feature.
Eventually, propagating feature is implemented by in-tree code change only(not use OpenTelemetry).This idea is TBD.

**Idea of design**  
TBD.


### Summary of collaboration idea of related KEPs

#### distributed tracing

 - Adding context information to kubernetes object's Annotations
 - Propagate Annotations by distributed context of OpenTelemetry(Eventually, propagating feature is implemented by in-tree code change only)
 - Consider consistent of metadata

#### structured logging

 - Request-ID implementation does not interfere with structured logging implementation
 - Consider consistent of metadata
