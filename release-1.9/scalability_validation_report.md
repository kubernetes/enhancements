Scalability validation report for release-1.9
=============================================

Written as per [this template](https://github.com/kubernetes/sig-release/blob/master/ephemera/scalability-validation.md#concretely-define-test-configuration).

Large Cluster Performance
-------------------------

Validated under the following configuration:
- Cloud-provider - GCE
- No. of nodes - 5000
- Node machine-type - n1-standard-1; OS - gci; Disk size - 50GB
- Master machine-type - n1-standard-64 Intel Broadwell or better; OS - gci; Disk size - [auto-calculated]
- Any non-default config used:
  - `KUBE_ENABLE_CLUSTER_MONITORING=none`
  - `KUBE_GCE_ENABLE_IP_ALIASES=true`
  - `ENABLE_APISERVER_ADVANCED_AUDIT=false`
  - `ENABLE_BIG_CLUSTER_SUBNETS=true`
  - `PREPULL_E2E_IMAGES=false`
  - `APISERVER_TEST_ARGS=--max-requests-inflight=3000 --max-mutating-requests-inflight=1000`
  - `SCHEDULER_TEST_ARGS=--kube-api-qps=100`
  - `CONTROLLER_MANAGER_TEST_ARGS=--kube-api-qps=100 --kube-api-burst=100`
  - `TEST_CLUSTER_LOG_LEVEL=--v=1`
  - `TEST_CLUSTER_RESYNC_PERIOD=--min-resync-period=12h`
  - `TEST_CLUSTER_DELETE_COLLECTION_WORKERS=--delete-collection-workers=16`

- Any important test details:
  - Services disabled in load test
  - SLO used for 99%ile for api call latency:
    - For clusters with <= 500 nodes:
      - <= 1s for all calls
    - For clusters with > 500 nodes:
      - <= 1s for non-list calls
      - <= 5s for namespaced list calls
      - <= 10s for cluster-scoped list calls
- See the [output from the validating performance test job run](https://k8s-gubernator.appspot.com/build/kubernetes-jenkins/logs/ci-kubernetes-e2e-gce-scale-performance/79) for other specific details from the logs.

Large cluster correctness
-------------------------

Validated under the following configuration:
- Cloud-provider - GCE
- No. of nodes - 5000
- Node machine-type - g1-small; OS - gci; Disk size - 50GB
- One special n1-standard-8 node (out of the 5k nodes) used for heapster
- Master machine-type - n1-standard-64 Intel Broadwell or better; OS - gci; Disk size - [auto-calculated]
- Any non-default config used:
  - `KUBE_ENABLE_CLUSTER_MONITORING=standalone`
  - `APISERVER_TEST_ARGS=--max-requests-inflight=1500 --max-mutating-requests-inflight=500`
  - `CONTROLLER_MANAGER_TEST_ARGS=--kube-api-qps=100 --kube-api-burst=100 --concurrent-service-syncs=5`
  - `PREPULL_E2E_IMAGES (default, true)`
  - (rest same as above - performance test)
- Any important test details:
  - Several tests were disabled because they did not pass at scale and fixing
    the related code was not feasible before the release:
    - [Horizontal Pod Autoscaler](https://github.com/kubernetes/kubernetes/blob/7335c41ebe076b/test/e2e/autoscaling/horizontal_pod_autoscaling.go#L69): https://github.com/kubernetes/kubernetes/issues/55887
    - [Advanced API call auditing](https://github.com/kubernetes/kubernetes/blob/7335c41ebe076b/test/e2e/auth/audit.go#L59): http://github.com/kubernetes/kubernetes/issues/53455
    - [Changing type and ports of a Service](https://github.com/kubernetes/kubernetes/blob/7335c41ebe076b/test/e2e/network/service.go#L486): http://github.com/kubernetes/kubernetes/issues/52495
- See the [output from the validating correcness test job run](https://k8s-gubernator.appspot.com/build/kubernetes-jenkins/logs/ci-kubernetes-e2e-gce-scale-correctness/46) for other specific details from the logs.
