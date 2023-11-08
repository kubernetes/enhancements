# KEP-4331: Livez and Readyz Probes

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Definition for livez and readyz](#definition-for-livez-and-readyz)
- [Design Details](#design-details)
  - [API Design](#api-design)
  - [Failure Modes and Detection Methods](#failure-modes-and-detection-methods)
    - [Detection](#detection)
- [Implementation Plan](#implementation-plan)
- [Future Discussions/Improvements](#future-discussionsimprovements)
<!-- /toc -->

## Summary
This is the KEP of the original design doc of [etcd livez and readyz probes](https://docs.google.com/document/d/1PaUAp76j1X92h3jZF47m32oVlR8Y-p-arB5XOB7Nb6U)

## Motivation
The current etcd implementation has a single `/health` probe that is used to determine both the liveness and readiness of a node.

What does it do?
* check if local node has leader (detects network partition)
* check if local node has any alarm activated
* check if local node is capable of serving a linearizable read request within hard-coded timeout (5s + 2 * election-timeout) (default 7s)

The current health probe is not Kubernetes API compliant. 
It does not differentiate whether etcd needs to restart or stop taking traffic. etcd liveness and readiness probes configured with [kubeadm](https://github.com/kubernetes/kubernetes/blob/master/cmd/kubeadm/app/phases/etcd/local.go#L225-L226) using the same health probe is insufficient. 


### Goals
* define the APIs for the new livez/readyz probes.
* add some basic livez/readyz checks.

### Non-Goals
* define the complete set of livez/readyz checks. New checks could be added in the future.

## Proposal
Add two separate probes
1. Liveness: the liveness probe would check that the local individual node is up and running, or else restart the node. 

1. Readiness: the readiness probe would check that the cluster is ready to serve traffic. 

The existing health probe stays unchanged except bug fixes. 

## Definition for livez and readyz
|             | /livez      | /readyz     |
| ----------- | ----------- | ----------- |
| Definition | Refer to [k8s](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/), properly reflect the fact whether the process is alive or not hence if it needs a restart. <br><br>Being alive means all the internal processes and resources are running properly, regardless of external dependencies of the peers or clients. | Refer to [k8s](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/), properly reflect the fact that process is ready to serve traffic <br><br>Being ready means the process is able to serve as a good access point to perform strongly consistent KV and watch operations on the underlying distributed key value store. <br><br>It is an indicator suggesting the client should not send KV and watch requests to this server, but it does not mean the server is actively blocking the requests. <br>Readiness is not an indicator of performance. Slow response is not covered by readiness. <br>Readiness does not cover admin functions. Administrators should connect directly to members to do maintenance. |
| Expected behavior | <ul><li>Return true if defrag is active </li><li>Catch deadlock in the [raft loop](https://github.com/etcd-io/etcd/blob/aa97484166d2b3fb6afeb4390344e68b02afb566/server/etcdserver/raft.go#L170-L328)</li><li>Catch deadlock in writing to and reading from db</li></ul> | <ul><li>Return false if defrag is active</li><li>Return false if corruption alarm is activated</li><li>Return false if etcd does not have leader</li><li>Validate linearizable read can be processed</li></ul> |
| Examples of no failures | Data corruption: restarting the server alone would not make it better. | Out of quota: the server is still able to take read/delete requests, and still able forward write requests to the leader to write successfully in the cluster.<br>Linearizable read timeout: the timeout could be due to multiple different reasons such as slow follower, readiness does not cover performance issues. |
| Consumer | Supervisor running close to the process (like Kubelet in Kubernetes) that can restart the process. | Supervisor running close to the process (like Kubelet in Kubernetes) that sets the ready status. <br>L7 Loadbalancer running as gateway <br>Client-side loadbalancer (like grpc client, proposal) |
|Expected execution | Every 5 seconds, after 3 failures the process is restarted. | Every 1 second, after X failure traffic is no longer sent to the process. |

## Design Details

### API Design
There will be 2 main http endpoints installed on `listen-client-http-urls` if the user opts in and falls back to default on `listen-client-urls`. 

1. /livez
1. /readyz

The API would return OK if all of the checks within that check group listed in “Desired behavior” are OK. 

The API also supports excluding specific checks from that health check group with query parameters. For example
```
curl -k 'https://localhost:2379/readyz?exclude=defragmentation'
curl -k 'https://localhost:2379/livez?exclude=serializable_read'
```

Each individual health check exposes an HTTP endpoint and can be checked individually. The schema for the individual health checks is /livez/<healthcheck-name> or /readyz/<healthcheck-name>.
```
curl -k 'https://localhost:2379/readyz/defragmentation'
```

### Failure Modes and Detection Methods
For each of the critical etcd functions, we are listing some potential failure modes for it below:
|  Function   | Potential Failure Modes |
| ----------- | ----------- | 
| Serializable Read | disk read failure, data corruption alarm, defrag |
| Linearizable Read | disk read failure, data corruption alarm, linearizable read loop deadlock, no raft leader, raft loop deadlock, no raft quorum, defrag | 
| Watch | disk read failure, data corruption alarm, watch loop deadlock, defrag |
| Write | raft loop deadlock, stalled disk write,  no raft quorum, defrag *(failure to write to stable or memory storage would result in FATAL or panic already) |

In this initial iteration of the probes, we will only focus on the functions of Serializable Read and Linearizable Read. The probing of Watch and Write would merit their own dedicated discussions in the future.

#### Detection
The table below shows the checks we plan to implement to detect the aforementioned failure modes. This list just reflects the initial implementation, and is not supposed to be exhaustive. We will make the checks to be easily extensible for more checks to be added in the future. 

| Health Check Name | Health Check Group | Related Failure Modes | Health Check Method* |
| ----------- | ----------- | ----------- | ----------- | 
| data_corruption | /readyz | data corruption alarm | check for active alarm of AlarmType_CORRUPT. |
| read_index** | /readyz | no raft leader, raft loop deadlock | check if the server can get ReadIndex. |
| serializable_read | /readyz, /livez | mvcc read failure | check if a serializable range (limit 1) request returns error, precondition on: defrag is not active. |
| ... | | | |

*We expect to execute the readyz check every 1s, and livez check every 5s. Any health check within that group should be able to finish below that time scale under normal circumstances. 

**Current health check checks if a linearizable read could finish. We prefer just checking the read index instead of doing a full linearizable read because we expect to execute the ready check every 1s, and a full linearizable read could timeout while the local server is trying to catch up with the applied entries, which is not covered under readiness.

## Implementation Plan
The following steps would be required to implement this change:
1. Add two new probes to etcd: a liveness probe and a readiness probe.
1. Add http handlers that could detect the above failure modes.
1. Integration and E2E tests the changes with failure mode simulation to ensure that probers work as expected.
1. Back port the changes to supported versions (3.4 and 3.5).
1. Update the etcd documentation to reflect the changes.

## Future Discussions/Improvements
There are several remaining topics that are worth more discussion or more work in the future to improve livez/readyz:
1. Should readyz check include checking writes?
1. Should readyz check cover performance issues?
1. What checks can we do to make sure watch is working properly?
1. In order to catch deadlock in raft loop in livez prober, is there a way to do this without involving external dependencies in multi-node scenario?
