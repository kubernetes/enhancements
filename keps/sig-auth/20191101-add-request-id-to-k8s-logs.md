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

The following two features are necessary to realize log tracking

### Propagate ID information

- Add Request-ID and Parent-ID information to objects related to a single API request
- Example: When creating a deployment resource, objects with the following Request-ID and Parent-ID are created

| Object | Object-UUID | Parent-ID  | Request-ID |
| ------ | ------ | ------ | ------ |
| Deployment | 1000 | null | 100 |
| ReplicaSet | 1010 | 1000 | 100 |
| Pod1 | 1020 | 1000 | 100 |
| Pod2 | 1030 | 1000 | 100 |
| Pod3 | 1040 | 1000 | 100 |

- Replicasets and pods related to the above Deployment are linked by Object-UUID and Parent-ID.
- In the above example, Replicaset and Pods have “1000” as the Parent-ID. In this case, these objects are linked to an object which has “1000” as Object-UUID
- Also, Reuest-ID should be associated with Request-ID in Audit log

### Add ID information to the logs

 - Add the above Object-UUID and Parent-ID information to the logs of each k8s component
 - Current log example:

```
Kube-apiserver
I1028 11:21:02.700432   11190 httplog.go:90] POST /api/v1/namespaces/kube-system/pods/kube-dns-68496566b5-24cwk/binding: (6.841722ms) 201 [hyperkube/v1.16.3 (linux/amd64) kubernetes/e76a12b/scheduler [::1]:35762]

Kube-scheduler
I1028 11:21:02.692701   11463 scheduler.go:530] Attempting to schedule pod: kube-system/kube-dns-68496566b5-24cwk
I1028 11:21:02.693154   11463 factory.go:610] Attempting to bind kube-dns-68496566b5-24cwk to 127.0.0.1
I1028 11:21:02.700681   11463 scheduler.go:667] pod kube-system/kube-dns-68496566b5-24cwk is bound successfully on node "127.0.0.1", 1 nodes evaluated, 1 nodes were found feasible. Bound node resource: "Capacity: CPU<4>|Memory<8037268Ki>|Pods<110>|StorageEphemeral<71724152Ki>.

Kubelet
I1028 11:21:02.699986   11594 kubelet.go:1901] SyncLoop (ADD, "api"): "kube-dns-68496566b5-24cwk_kube-system(7b0e128d-2a58-4c2c-8374-1ef872eefa65)"
```

 - New log example: Add ID information to the head of each log

```
I1028 11:21:02.700432   11190 httplog.go:90] [RequestID, ObjectID, ParentID] POST /api/v1/namespaces/kube-system/pods/kube-dns-68496566b5-24cwk/binding:(6.841722ms) 201 [hyperkube/v1.16.3 (linux/amd64) kubernetes/e76a12b/scheduler [::1]:35762]
I1028 11:21:02.699986   11594 kubelet.go:1901] [RequestID, ObjectID, ParentID] Attempting to bind kube-dns-68496566b5-24cwk to 127.0.0.1
I1028 11:21:02.692701   11463 scheduler.go:530] [RequestID, ObjectID , ParentID] Attempting to schedule pod: kube-system/kube-dns-68496566b5-24cwk
```

- By combining the above two functions, we can easily search for logs related to a single API by using Request-ID information as a query key.
- In addition, we can also know more detail information about API request (e.g. when and who requested what API) by linking to Audit logs.
