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
last-updated: 2019-11-01
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
