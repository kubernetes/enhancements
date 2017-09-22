This is the scalability validation report for release-1.8 written as per [this](https://github.com/kubernetes/community/blob/master/contributors/devel/release/scalability-validation.md#concretely-define-test-configuration) template.

Validated large cluster performance under the following configuration:
- Cloud-provider - GCE
- No. of nodes - 5000
- Node machine-type - n1-standard-1; OS - gci; Disk size - 50GB
- Master machine-type - [auto-calculated]; OS - gci; Disk size - [auto-calculated]
- Any non-default config used:
  - KUBE_ENABLE_CLUSTER_MONITORING=none
  - ENABLE_BIG_CLUSTER_SUBNETS=true
  - SCHEDULER_TEST_ARGS=--kube-api-qps=100
  - CONTROLLER_MANAGER_TEST_ARGS=--kube-api-qps=100 --kube-api-burst=100
  - APISERVER_TEST_ARGS=--max-requests-inflight=3000 --max-mutating-requests-inflight=1000
  - TEST_CLUSTER_RESYNC_PERIOD=--min-resync-period=12h
  - TEST_CLUSTER_DELETE_COLLECTION_WORKERS=--delete-collection-workers=16
- Any important test details:
  - Services disabled in load test
  - SLO used for 99%ile for api call latency:
    - For clusters with <= 500 nodes:
      - <= 1s for all calls
    - For clusters with > 500 nodes:
      - <= 1s for non-list calls
      - <= 5s for namespaced list calls
      - <= 10s for cluster-scoped list calls
- <job-name, run#> of the validating run (to know other specific details from the logs): https://k8s-gubernator.appspot.com/build/kubernetes-jenkins/logs/ci-kubernetes-e2e-gce-scale-performance/36

Validated large cluster correctness under the following configuration:
- Cloud-provider - GCE
- No. of nodes - 5000
- Node machine-type - g1-small; OS - gci; Disk size - 50GB
- One special n1-standard-8 node (out of the 5k nodes) used for heapster
- Master machine-type - [auto-calculated]; OS - gci; Disk size - [auto-calculated]
- Any non-default config used:
  - KUBE_ENABLE_CLUSTER_MONITORING=standalone
  - APISERVER_TEST_ARGS=--max-requests-inflight=1500 --max-mutating-requests-inflight=500
  - (rest same as above)
- Any important test details:
  - A few e2es around external loadbalancer timing out due to GCE-side issues ([#52495](https://github.com/kubernetes/kubernetes/issues/52495)) - being fixed
  - A few e2es around heapster and stackdriver logging are flaking - need stabilization
- <job-name, run#> of the validating run (to know other specific details from the logs): https://k8s-gubernator.appspot.com/build/kubernetes-jenkins/logs/ci-kubernetes-e2e-gce-scale-correctness/13

Misc:
- Seeing a performance regression in density test on 5k-node with api call latency (most notably 'patch node-status') - [#51899](https://github.com/kubernetes/kubernetes/issues/51899)
