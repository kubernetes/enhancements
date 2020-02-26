---
title: Add Request-ID to each k8s component log
authors:
 - "@hase1128"
 - "@sshukun"
 - "@furukawa3"
 - "@vanou"
owning-sig: sig-instrumentation
participating-sigs:
reviewers:
 - TBD
approvers:
 - TBD
editor: TBD
creation-date: 2019-11-01
last-updated: 2020-02-26
status: provisional
---

# Add Request-ID to each Kubernetes component log

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
   - [Basic policy of Request-ID](#basic-policy-of-request-id)
   - [Design Overview](#design-overview)
     - [Design overview of Propagate Request-ID](#design-overview-of-propagate-request-id)
     - [Detail design of Propagate Request-ID](#detail-design-of-propagate-request-id)
     - [Design overview of Export Request-ID](#design-overview-of-export-request-id)
     - [Detail design of Export Request-ID](#detail-design-of-export-request-id)
     - [Design overview of Control Request-ID](#design-overview-of-control-request-id)
     - [Detail design of Control Request-ID](#detail-design-of-control-request-id)
 - [Test Plan](#test-plan)
 - [Migration / Graduation Criteria](#migration--graduation-criteria)
   - [Alpha](#alpha)
   - [Beta](#beta)
   - [GA](#ga)

<!-- /toc -->

## Summary

This KEP proposes a new unique logging meta-data into all Kubernetes logs. It makes us
more easy to identify specific logs related to a single user operation (such as
`kubectl apply -f <my-pod.yaml>`). This feature is similar to
[Request-ID](https://docs.openstack.org/api-guide/compute/faults.html) for
OpenStack. It greatly reduces the investigation cost.

## Motivation

### Target User

Support team in k8s Service Provider

### Target User's objective

We'd like to resolve quickly for end users' problem.

Tracking logs among each Kubernetes component related to specific an user operation is very tough work. It is necessary to match logs with timestamps as hints. If multiple users throw many API requests at the same time, it is very difficult to track logs across each Kubernetes component log. 

It is difficult that support team in k8s Service Provider resolve end user's problem quickly in the above. Therefore, we'd like to add a new identifier which is unique to each user operation. This feature is useful for the following use cases:

#### Case 1

In case of insecure or unauthorized operation happens, it is necessary to
identify what effect that operation caused. This proposed feature helps identify
what happened at each component or server by each insecure / unauthorized API
request. We can collect these logs as an evidence.

#### Case 2

If the container is terminated by OOM killer, there is a case to break down the
issue into parts(Pod or Kubernetes) from the messages related OOM killer on host logs
and the API processing just before OOM killer. If the cause is that some unknown
pod creations, it is helpful to detect the root API request and who called this
request.

### Goals

 - Adding a Request-ID into each K8s component log.
 - The Request-ID is unique to an operation.
 - Control enabled/disabled Request-ID feature(Request-ID feature is disabled on default to avoid an impact for existing user).

### Non-Goals

 - To centrally manage the logs of each Kubernetes component with Request-ID (This can
be realized with existing OSS such as Kibana, so no need to implement into Kubernetes
components).
 - We don't associate Request-ID to all of operations(Our target is important operations such as `kubectl create/delete/etc.`).

## Proposal

### Basic policy of Request-ID

 - Minimize the impact to existing users who are retrieving logs and analyzing with existing log format.
   - So we disabled Request-ID feature on default.
 - Collaborate with related KEPs to avoid unnecessary conflict to them regarding implementation and feature.
   - Use existing KEP's feature as much as possible.
   - Therefore, we will merge Request-ID feature after related KEP features are merged.

### Design Overview

Request-ID feature consists the three features.
 - Propagate Request-ID to related objects.
 - Export Request-ID to each kubernetes component log.
 - Control enabled/disabled the above two features.

#### Design overview of Propagate Request-ID

There is an idea to use `distributed context` of the existing KEP([Distributed Tracing](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/0034-distributed-tracing-kep.md)). We use `Distributed Tracing` feature for propagation.

 - We use Trace-ID as Request-ID
 - Trace-ID is contained in Annotation of Kubernetes objects. (This is a part of `Distributed Tracing` feature)
 - Trace-ID is removed when tracing is finished. (This is a part of `Distributed Tracing` feature)

We have two notes should be considered when using `Distributed Tracing` feature for Request-ID feature.

**NOTE1:**  
`Distributed Tracing` feature is only enabled when `--trace` option is added to `kubectl` command. However, in order for the Kubernetes service provider's support team to troubleshoot from customer's Kubernetes log files, we would like to ensure that Request-ID is always added to the log file regardless of `--trace` option. So we need to implement additional parameter which called `--request-id`. This parameter can control(enabled/disabled) Request-ID feature(See [Design overview of Control Request-ID](#design-overview-of-control-request-id)). As a result, the customers who want to use Request-ID can always use this feature, and does not affect other users who does not want to use this feature. Below is a table showing the relationship between the `--trace option` and the `--request-id` parameter.

| | --trace: OFF | --trace: ON |
| ------ | ------ | ------ |
| --request-id = 0 | No Tracing / No export Request-ID | Tracing / No export Request-ID |
| --request-id > 0 | Tracing / Export Request-ID | Tracing / Export Request-ID |

**NOTE2:**  
To trace each Kubernetes function, we need to add codes into related k8s function. So the following implementation is needed.
 - Case1. The function which is added Tracing codes by `Distributed Tracing` KEP
   - We add Request-ID codes over Tracing codes.
 - Case2. The function which is not added Tracing codes by `Distributed Tracing` KEP
   - We add both of Tracing codes and Request-id codes.

#### Detail design of Propagate Request-ID

TBD. I will write down the following things.
 - List up the target function that we add tracing and Request-id codes
 - Sample codes of case 1 and 2 of the above `NOTE2`

#### Design overview of Export Request-ID

We add Request-ID information into klog calls. Note that we don't associate Request-ID to all of operations. Our target is important operations such as `kubectl create/delete/etc.`, and our target klog calls is the only klogs which is called via such important operations. Request-ID feature does not change existing klog function, but changes each klog calls and their log format. Currently, there is [Structured logging](https://github.com/serathius/enhancements/blob/structured-logging/keps/sig-instrumentation/20191115-structured-logging.md) KEP, and this KEP also change specific klog calls. We will merge Request-ID feature after Structured logging KEP is merged.

#### Detail design of Export Request-ID

TBD. I will write down the following things.
 - List up the target klogs that we add Request-id
 - Sample codes of klog calls that we changes
 - Sample logs which is added Request-ID

#### Design overview of Control Request-ID

We should control Request-ID feature to avoid an impact to existing users who are retrieving logs and analyzing with existing log format. So we introduce `--request-id` parameter which enables/disables Request-ID feature. We also manage the range of operations which are added Request-ID. The effect of each parameter of `--request-id` is as follows.

| parameter | efficient |
| ------ | ------ |
| --request-id=0 | Request-ID feature is disabled (Default) |
| --request-id=1 | Request-ID feature is enabled, and Request-ID is added to klogs related to the `Alpha` target operations | 
| --request-id=2 | Request-ID feature is enabled, and Request-ID is added to klogs related to the `Alpha and Beta` target operations | 

Alpha and Beta target operations are described in Migration / Graduation Criteria section.

#### Detail design of Control Request-ID

TBD. I will write down the following things.
 - How to realize `--request-id` parameter in each Kubernetes component.
 - Sample codes which is used with `--request-id` and `--trace` option.

### Test Plan

 - test against the combination of following patterns.
   - --trace(OFF/ON) / --request-id(0/1/2)

### Migration / Graduation Criteria

#### Alpha

 - Add Request-ID against the following operations:
   - kubectl create/apply/delete
     - target resources: pod/deployment
   - kubectl drain
     - target resources: node
 - Implement `--request-id` parameter
 - E2e testing 
 - User-facing documentation

#### Beta

 - Add Request-ID against the following operations:
   - kubectl create/apply/delete
     - target resources: daemonset/pv/pvc/svc
   - kubectl scale/rollout
 - Update E2e testing
 - Update documentation

#### GA

 - All feedback is addressed.

