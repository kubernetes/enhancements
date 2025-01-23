# KEP-5032: Container log eviction on Disk perssure

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [What would you like to be added?](#what-would-you-like-to-be-added)
  - [Why is this needed?](#why-is-this-needed)
  - [Spec 1](#spec-1)
  - [Spec 2](#spec-2)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Only Rotation on disk pressure](#only-rotation-on-disk-pressure)
  - [DaemonSet to cleanup logs](#daemonset-to-cleanup-logs)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
    - [ ] e2e Tests for all Beta API Operations (endpoints)
    - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
    - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
    - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Clean and Rotate containers logs when there is disk pressure on kubelet host.

## Motivation

- We manage kubernetes ecosystem at our organization. A lot of our kubelet hosts experienced Disk pressure as a certain set of pods was generating very high logs. The rate was around 3-4Gib in 15 minutes. We had containerLogMaxSize set to 200Mib and containerLogMaxFiles set to 6. But the .gz files were of size around 500-600Gib. We observed that container log rotation was slow for us.

### What would you like to be added?

Log cleanup should be another form of eviction to make space like we do with Images and containers.

### Why is this needed?

It often happens that the containers generating heavy log data have compressed log file with size exceeding the containerLogMaxSize limit set in kubelet config.

For example, kubelet has
```
containerLogMaxSize = 200M
containerLogMaxFiles = 6
```

### Spec 1

Continuously generating 10Mib with 0.1 sec sleep in between
```
apiVersion: batch/v1
kind: Job
metadata:
  name: generate-huge-logs
spec:
  template:
    spec:
      containers:
      - name: log-generator
        image: busybox
        command: ["/bin/sh", "-c"]
        args:
          - |
            # Generate huge log entries to stdout
            start_time=$(date +%s)
            log_size=0
            target_size=$((4 * 1024 * 1024 * 1024))  # 4 GB target size in bytes
            while [ $log_size -lt $target_size ]; do
              # Generate 1 MB of random data and write it to stdout
              echo "Generating huge log entry at $(date) - $(dd if=/dev/urandom bs=10M count=1 2>/dev/null)"
              log_size=$(($log_size + 1048576))  # Increment size by 1MB
              sleep 0.1  # Sleep to control log generation speed
            done
            end_time=$(date +%s)
            echo "Log generation completed in $((end_time - start_time)) seconds"
      restartPolicy: Never
  backoffLimit: 4
```
File sizes
```
-rw-r----- 1 root root  24142862 Jan  1 11:41 0.log
-rw-r--r-- 1 root root 183335398 Jan  1 11:40 0.log.20250101-113948.gz
-rw-r--r-- 1 root root 364144934 Jan  1 11:40 0.log.20250101-114003.gz
-rw-r--r-- 1 root root 487803789 Jan  1 11:40 0.log.20250101-114023.gz
-rw-r--r-- 1 root root 577188544 Jan  1 11:41 0.log.20250101-114047.gz
-rw-r----- 1 root root 730449620 Jan  1 11:41 0.log.20250101-114115
```

### Spec 2

Continuously generating 10Mib with 10 sec sleep in between
```
apiVersion: batch/v1
kind: Job
metadata:
  name: generate-huge-logs
spec:
  template:
    spec:
      containers:
      - name: log-generator
        image: busybox
        command: ["/bin/sh", "-c"]
        args:
          - |
            # Generate huge log entries to stdout
            start_time=$(date +%s)
            log_size=0
            target_size=$((4 * 1024 * 1024 * 1024))  # 4 GB target size in bytes
            while [ $log_size -lt $target_size ]; do
              # Generate 1 MB of random data and write it to stdout
              echo "Generating huge log entry at $(date) - $(dd if=/dev/urandom bs=10M count=1 2>/dev/null)"
              log_size=$(($log_size + 1048576))  # Increment size by 1MB
              sleep 0.1  # Sleep to control log generation speed
            done
            end_time=$(date +%s)
            echo "Log generation completed in $((end_time - start_time)) seconds"
      restartPolicy: Never
  backoffLimit: 4
```

File sizes
```
-rw-r----- 1 root root 181176268 Jan  1 11:31 0.log
-rw-r--r-- 1 root root 183336647 Jan  1 11:20 0.log.20250101-111730.gz
-rw-r--r-- 1 root root 183323382 Jan  1 11:23 0.log.20250101-112026.gz
-rw-r--r-- 1 root root 183327676 Jan  1 11:26 0.log.20250101-112321.gz
-rw-r--r-- 1 root root 183336376 Jan  1 11:29 0.log.20250101-112616.gz
-rw-r----- 1 root root 205360966 Jan  1 11:29 0.log.20250101-112911
```


If the pod had been generating logs in Gigabytes with minimal delay, it can cause disk pressure on kubelet host and that can affect other pods running in the same kubelets.

### Goals

- Rotate and Clean all container logs on kubelet Disk pressure that has exceeded the configured log retention quota.

## Proposal
- On disk pressure, analyse log paths of all containers. Logs of containers with existing combined log size exceeding `containerLogMaxSize`*`containerLogMaxFiles` should be deleted till the combined log size is within `containerLogMaxSize`*`containerLogMaxFiles`.

### Risks and Mitigations

Risk of tmp copy creation of log failing as there is no disk space left.  

## Design Details

- When there is disk pressure in nodefs.available/imagefs.available, ContainerLogManager's rotateLogs (https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/logs/container_log_manager.go#L52-L60) should be added to list of functions to be run on disk pressure. (https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/eviction/helpers.go#L1195-#L1230)
- And container logs exceeding the kubelet config set for log limit should be deleted. On disk pressure, analyse log paths of all containers. Logs of containers with existing combined log size exceeding `containerLogMaxSize`*`containerLogMaxFiles` should be deleted till the combined log size is within `containerLogMaxSize`*`containerLogMaxFiles`.


### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests
- Add detailed unit tests with 100% coverage.
- `<package>`: `<date>` - `<test coverage>`

##### Integration tests
- Scenarios will be covered in e2e tests. 

##### e2e tests
- Add test under `kubernetes/test/e2e_node`.
- Set low value for `containerLogMaxSize` and `containerLogMaxFiles`.
- Create a pod with generating heavy logs and expect the container's combined log size to bw within `containerLogMaxSize`*`containerLogMaxFiles`.
- 
### Graduation Criteria

**Note:** *Not required until targeted at a release.*


###### How can this feature be enabled / disabled in a live cluster?

- [X] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane? Yes (kubelet restart)
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? No, restart of kubelet with updated configurations and version should work.

###### Does enabling the feature change any default behavior?
No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?
Yes

###### What happens if we reenable the feature if it was previously rolled back?


###### Are there any tests for feature enablement/disablement?
Add UTs.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?
No identified risk.

###### What specific metrics should inform a rollback?
No identified risk.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?
e2e tests covered.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?
No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?
Emit cleanup logs.

###### How can someone using this feature know that it is working for their instance?
Yes, from logs.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?
Na

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?
NA

###### Are there any missing metrics that would be useful to have to improve observability of this feature?
NA

### Dependencies

###### Does this feature depend on any specific services running in the cluster?
No

### Scalability

###### Will enabling / using this feature result in any new API calls?
No

###### Will enabling / using this feature result in introducing new API types?
No

###### Will enabling / using this feature result in any new calls to the cloud provider?

###### Will enabling / using this feature result in increasing size or count of the existing API objects?
No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?
No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?
CPU cycles usage of ContainerLogManager of kubelet will increase.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?
No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?
NA

###### What steps should be taken if SLOs are not being met to determine the problem?
NA

## Implementation History
NA

## Drawbacks
No identified drawbacks.

## Alternatives

### Only Rotation on disk pressure
Define 2 new flags `logRotateDiskCheckInterval`, `logRotateDiskPressureThreshold` in kubelet config.

- `logRotateDiskCheckInterval` is the time interval within which the ContainerLogManager will check Disk usage on the kubelet host.
- `logRotateDiskPressureThreshold` is the threshold of overall Disk usage on the kubelet. If actual Disk usage is equal or more than this threshold, it will rotate logs of all the containers of the kubelet.

### DaemonSet to cleanup logs
Provide a means for an external tool to trigger the kubelet to rotate its logs. That would move the policy decisions outside of the kubelet, for example, into a DaemonSet.