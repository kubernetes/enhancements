# KEP-2008: Forensic Container Checkpointing

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation](#implementation)
    - [CRI Updates](#cri-updates)
  - [User Stories](#user-stories)
    - [Forensic Container Checkpointing](#forensic-container-checkpointing)
    - [Fast Container Startup](#fast-container-startup)
    - [Container Migration](#container-migration)
      - [Fault Tolerance](#fault-tolerance)
      - [Load Balancing](#load-balancing)
      - [Spot Instances](#spot-instances)
      - [Scheduler Integration](#scheduler-integration)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Future Enhancements](#future-enhancements)
    - [Checkpoint Archive Management](#checkpoint-archive-management)
    - [CLI (kubectl) Integration](#cli-kubectl-integration)
    - [Checkpoint Options](#checkpoint-options)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha to Beta Graduation](#alpha-to-beta-graduation)
    - [Beta to GA Graduation](#beta-to-ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Provide an interface to trigger a container checkpoint for forensic analysis.

## Motivation

Container checkpointing provides the functionality to take a snapshot of a
running container. The checkpointed container can be transferred to another
node and the original container will never know that it was checkpointed.

Restoring the container in a sandboxed environment provides a mean to
forensically analyse a copy of the container to understand if it might
have been a possible threat. As the analysis is happening on a copy of
the original container a possible attacker of the original container
will not be aware of any sandboxed analysis.

### Goals

The goal of this KEP is to introduce *checkpoint* to the CRI API.
This includes extending the *kubelet* API to support checkpointing single
containers with the forensic use case in mind.

### Non-Goals

Although *checkpoint* and *restore* can be used to implement container
migration this KEP is only about enabling the forensic use case. Checkpointing
a pod is not part of this proposal and left for future enhancements.

## Proposal

### Implementation

For the forensic use case we want to offer the functionality to checkpoint a
container out of a running Pod without stopping the checkpointed container or
letting the container know that it was checkpointed.

The corresponding code changes for the forensic use case can be found in the
following pull request:

* https://github.com/kubernetes/kubernetes/pull/104907

The goal is to introduce *checkpoint* and *restore* in a bottom-up approach.
In a first step we only want to extend the CRI API to trigger a checkpoint
by the container engine and to have the low level primitives in the *kubelet*
to trigger a checkpoint. It is necessary to enable the feature gate
`ContainerCheckpoint` to be able to checkpoint containers.

In the corresponding pull request a checkpoint is triggered using the *kubelet*
API:

```
curl -skv -X POST "https://localhost:10250/checkpoint/default/counters/wildfly"
```

For the first implementation we do not want to support restore in the
*kubelet*. With the focus on the forensic use case the restore should happen
outside of Kubernetes. The restore is a container engine only operation
in this first step.

A high level view on the implementation is that triggering the *kubelet* API
endpoint will trigger the `ContainerCheckpoint` CRI API endpoint to create a
checkpoint at the location defined by the *kubelet*.  In the checkpoint request
the kubelet will specify the name of the checkpoint archive as
`checkpoint-<podFullName>-<containerName>-<timestamp>.tar` and also request to
store the checkpoint archive in the `checkpoints` directory below its root
directory (as defined by `--root-dir`). This defaults to
`/var/lib/kubelet/checkpoints`.

To trigger a checkpoint following HTTP Request has to be made against the *kubelet*:

- `POST /checkpoint/{namespace}/{pod}/{container}``
- Parameters
  - namespace (in path): string, required, Namespace
  - pod (in path): string, required, Pod
  - container (in path): string, required, Container
  - timeout (in query): integer, Timeout in seconds to wait until the checkpoint
  creation is finished. If zero or no timeout is specified the default CRI
  timeout value will be used. Checkpoint creation time depends directly on the
  used memory of the container. The more memory a container uses the more time
  is required to create the corresponding checkpoint.
- Response
  - 200: OK
  - 401: Unauthorized
  - 404: Not Found (if the ContainerCheckpoint feature gate is disabled)
  - 404: Not Found (if the specified namespace, pod or container cannot be found)
  - 500: Internal Server Error (if the CRI implementation encounter an error during checkpointing (see error message for further details))
  - 500: Internal Server Error (if the CRI implementation does not implement the checkpoint CRI API (see error message for further details))

The kubelet APIs are usually restricted to cluster admins and is only accessible
via `localhost`. Users will not have access to this for now. If, in the future
the checkpoint API endpoint is moved out of the kubelet it can then have proper
RBAC. This is something that cannot be provided as a *kubelet* API. Also see
<https://kubernetes.io/blog/2022/12/05/forensic-container-checkpointing-alpha/>

To further secure the *kubelet* API endpoint there will be a kubelet auth endpoint
added to the checkpoint sub-resource. The goal is to allow administrators to
restrict the API endpoint and to ensure that users do not have access to the
endpoint via the kubernetes API server proxy mode.

Expected latency depends directly on size of the used memory of the processes in
the container. The more memory is used the longer the operation will require.
The newly introduced CRI API includes a `timeout` parameter to automatically cancel
the request if it requires more time than requested. If the `timeout` parameter
is not specified the CRI default timeout from the *kubelet* is used (2 minutes).

#### CRI Updates

The CRI API will be extended to introduce one new RPC:
```
    // CheckpointContainer checkpoints a container
    rpc CheckpointContainer(CheckpointContainerRequest) returns (CheckpointContainerResponse) {}
```
with the following parameters:
```
message CheckpointContainerRequest {
    // ID of the container to be checkpointed.
    string container_id = 1;
    // Location of the checkpoint archive used for export/import
    string location = 2;
    // Timeout in seconds for the checkpoint to complete.
    // Timeout of zero means to use the CRI default.
    // Timeout > 0 means to use the user specified timeout.
    int64 timeout = 3;
}

message CheckpointContainerResponse {}
```

### User Stories

For the initial Alpha release this KEP was focusing on "Forensic Container
Checkpointing". The checkpoint/restore technology, however, opens up the
possibility to many different use cases. Since the introduction in Kubernetes
1.25 there has been feedback from users that were using the checkpoint
functionality for some of those other use cases. In the following some
of the possible use cases are described starting with the original
"Forensic Container Checkpointing" use case. At which point any of these
use cases will be supported in Kubernetes is not defined yet. At this point
any of the use cases can be used with the currently available implementation.
One question for the future will be in how far any of the possible use cases can
be made more user friendly by additional Kubernetes features.  Especially the
container migration use case has many possibilities for optimization. CRIU,
which is on the lowest level of the checkpoint/restore stack, offers the
possibility to decrease container downtime during migration by techniques well
known in virtual machine migration like pre-copy or post-copy migration.

#### Forensic Container Checkpointing

To analyze unusual activities in a container, the container should
be checkpointed without stopping the container or without the container
knowing it was checkpointed. Using checkpointing it is possible to take
a copy of a running container for forensic analysis. The container will
continue to run without knowing a copy was created. This copy can then
be restored in another (sandboxed) environment in the context of another
container engine for detailed analysis of a possible attack.

#### Fast Container Startup

In addition to forensic analysis of a container checkpointing can be used to
offer a way to quickly start containers. This is especially useful for
containers that need a long time start. Either the software in the container
needs a long time to initialize by loading many libraries or the container
requires time to read data from a storage device. Using checkpointing it is
possible to wait once until the container finished the initialization and save
the initialized state to a checkpoint archive. Based on this checkpoint archive
one or multiple copies of the container can be created without the need to wait
for the initialization to finish. The startup time is reduced to the time
necessary to read back all memory pages to their previous location.

This feature is already used in production to decrease startup time of
containers.

Another similar use case for quicker starting containers has been reported in
combination with persistent memory systems. The combination of checkpointed
containers and persistent memory systems can reduce startup time after a reboot
tremendously.

#### Container Migration

One of the main use cases for checkpointing and restoring containers is
container migration. An open issue asking for container migration in
Kubernetes exists since 2015: [#3949][migration-issue].

With the current Alpha based implementation container migration is already
possible as documented in [Forensic Container Checkpointing Alpha][kubernetes-blog-post].

The following tries to give an overview of possible use cases for
container migration.

##### Fault Tolerance

Container migration for fault tolerance is one of the typical reasons to
migrate containers or processes. It is a well researched topic especially
in the field of high performance computing (HPC). To avoid loss of work
already done by a container the container is migrated to another node before
the current node crashes. There are many scientific papers describing how
to detect a node that might soon have a hardware problem. The main goal
of using container migration for fault tolerance is to avoid loss of already
done work. This is, in contrast to the forensic container checkpointing use
case, only useful for stateful containers.

With GPUs becoming a costly commodity, there is an opportunity to help
users save on costs by leveraging container checkpointing to prevent
re-computation if there are any faults.

##### Load Balancing

Container migration for load balancing is something where checkpoint/restore
as implemented by CRIU is already used in production today. A prominent example
is Google as presented at the Linux Plumbers conference in 2018:
[Task Migration at Scale Using CRIU]<[task-migration]>

If multiple containers are running on the same physical node in a cluster,
checkpoint/restore, and thus container migration, open up the possibility
of migrating containers across cluster nodes in case there are not enough
computational resources (e.g., CPU, memory) available on the current node.
While high-priority workloads can continue to run on the same node, containers
with lower priority can be migrated. This way, stateful applications with low
priority can continue to run on a different node without loosing their progress
or state.

This functionality is especially valuable in distributed infrastructure services
for AI workloads, as it helps reduce the cost of AI by maximizing the aggregate
useful throughput on a given pool with a fixed capacity of hardware accelerators.
Microsoft's globally distributed scheduling service, [Singularity]<[singularity]>,
is an example that demonstrates the efficiency and reliability of this mechanism
with deep learning training and inference workloads.

##### Spot Instances

Yet another possible use case where checkpoint/restore is already used today
are spot instances. Spot instances are usually resources that are cheaper but
with the drawback that they might shut down with very short notice. With the
help of checkpoint/restore workloads on spot instances can either be
checkpointed regularly or the checkpointing can be triggered by a signal.
Once checkpointed the container can be moved to another instance and
continue to run without having to start from the beginning.

##### Scheduler Integration

All of the above-mentioned container migration use cases currently require manual
checkpointing, manual transfer of the checkpoint archive, and manual restoration
of the container (see [Forensic Container Checkpointing Alpha][kubernetes-blog-post]
for details). If all these steps could be automatically performed by the scheduler,
it would greatly improve the user experience and enable more efficient resource
utilization. For example, the scheduler could transparently checkpoint, preempt,
and migrate workloads across nodes while keeping track of available resources and
identifying suitable nodes (with compatible hardware accelerators) where a container
can be migrated. However, scheduler integration is likely to be implemented at a later
stage and is a subject for future enhancements.

### Risks and Mitigations

In its first implementation the risks are low as it tries to be a CRI API change
with minimal changes to the kubelet and it is gated by the feature
gate `ContainerCheckpoint`.

One possible risk that was identified during Alpha is that the disk of the node
requesting the checkpoints could fill up if too many checkpoints are created and
the node will be marked as bot healthy. One approach to solve this was some kind
of garbage collection of checkpoint archives. A pull request to implement
garbage collection was opened
([#115888](https://github.com/kubernetes/kubernetes/pull/115888)) but during
review it became clear that the kubelet might not be the right place to
implement checkpoint archive garbage collection and the pull request was closed
again. Currently the most likely solution seems to be to implement the garbage
collection in an operator. Garbage collection via an operator could
clean up the checkpoint directory and the node would stay healthy.
Currently manual cleanup might be required to ensure the node does not run
out of disk space and stays healthy. Especially in situation where checkpoint
creation requests are triggered automatically and not manually. As service that
requests many checkpoints should ensure to remove the requested checkpoints as
soon as possible.

## Design Details

The feature gate `ContainerCheckpoint` will ensure that the API
graduation can be done in the standard Kubernetes way.

A kubelet API to trigger the checkpointing of a container will be
introduced as described in [Implementation](#implementation).

Also see https://github.com/kubernetes/kubernetes/pull/104907 for details.

### Future Enhancements

The initial implementation is only about checkpointing specific containers
out of a pod. In future versions we probably want to support checkpointing
complete pods. To checkpoint a complete pod the expectation on the container
engine would be to do a pod level cgroup freeze before checkpointing the
containers in the pod to ensure that all containers are checkpointed at the
same point in time and that the containers do not keep running while other
containers in the pod are checkpointed.

One possible result of being able to checkpoint and restore containers and pods
might be the possibility to migrate containers and pods in the future as
discussed in [#3949][migration-issue].

#### Checkpoint Archive Management

One of the questions from users has been what happens with old checkpoint archives.
Especially if there are multiple checkpoints on a single node theses checkpoint
archives can occupy node local disk space. Depending on the checkpoint archive size
this could result in a situation where the node runs out of local disk space.

One approach to avoid out of disk space situations would be some kind of
checkpoint archive management or garbage collection of old checkpoint archives.

One possible argument against checkpoint archive management could be that, especially
for the forensic use case, once the checkpoint archive has been created the user should
retrieve it from the node and delete it. As there are, however, many different use
cases for container checkpointing it sounds more realistic to have an existing checkpoint
archive management to automatically clean up old checkpoint archives.

In its simplest form a checkpoint archive management could just start deleting checkpoint
archives once the number of checkpoints reaches a certain threshold. If more checkpoint
archives than the configurable threshold exist older checkpoint archives are deleted (see [#115888][checkpoint-management]).

Another way to manage checkpoint archives would be to delete checkpoint archives once
a certain amount disk space is used or if not enough free space is available.

A third way to manage checkpoint archives would be the possibility to keep one checkpoint
archive per day/week/month. The way to manage checkpoint archives probably depends on
the checkpointing use case. For the forensic use case other checkpoint archives might
be of interest in contrast to checkpointing and restoring containers in combination
with spot instances where probably only the latest checkpoint is of interest to
be able to continue preempted work.

#### CLI (kubectl) Integration

The current (Alpha) implementation only offers access to the checkpoint functionality
through the *kubelet* API endpoint. A more user friendly interface would be a *kubectl*
integration. As of this writing a pull request adding the *checkpoint* verb to
*kubectl* exists: [#120898][kubectl-checkpoint]

#### Checkpoint Options

The current (Alpha) implementation does not allow additional checkpoint parameters to be
passed to CRIU. During the integration of checkpoint/restore in other container projects
(CRI-O, Docker, Podman, runc, crun, lxc) many CRIU specific options were exposed to the user.
Common options are things like handle TCP established connections (`--tcp-established`),
stop container after checkpointing, use pre-copy or post-copy algorithms to decrease
container downtime during migration or compression method of the checkpoint archive (currently
uncompressed but things like zstd or gzip possible).

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.
All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.
[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

For Alpha to Beta graduation existing tests will extended with following tests:

- [] For Alpha tests are trying to be clever and automatically skip the test if
  disabled. The new tests should explicitly test following situation without
  automatic skipping to ensure we are not hiding potential errors.
  - [ ] Test to ensure the feature does not work with the feature gate disabled.
  - [ ] Test to ensure the feature does work if enabled.
- [ ] Test CRI metrics related to `ContainerCheckpoint` CRI RPC.
- [ ] Test kubelet metrics related to kubelet `checkpoint` API endpoint.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit
This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- Test coverage before Alpha graduation
  - `pkg/kubelet`: 06-17-2022 - 64.5
  - `pkg/kubelet/container`: 06-17-2022 - 52.1
  - `pkg/kubelet/server`: 06-17-2022 - 64.3
  - `pkg/kubelet/cri/remote`: 06-17-2022 - 13.2
- Test coverage before Beta graduation
  - `pkg/kubelet`: 02-08-2024 - 68.9
  - `pkg/kubelet/container`: 02-08-2024 - 55.7
  - `pkg/kubelet/server`: 02-08-2024 - 65.1
  - `pkg/kubelet/cri/remote`: 02-08-2024 - 18.9

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- CRI API changes need to be implemented by at least one
  container engine

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- Alpha will include e2e tests with the expectation that
  no CRI implementation provides the newly added RPC calls.
  Once CRI implementation provide the relevant RPC calls
  the e2e tests will not fail but need to be extended.

- Once the initial Alpha release  CRI-O supports the
  `CheckpointContainer` CRI RPC and tests have been
  enhanced to support CRI implementation that implement
  the `CheckpointContainer` CRI RPC

- Once Kubernetes was released with the `CheckpointContainer` CRI RPC
  CRI-O has been updated to support the new CRI RPC.
  The tests have been enhanced to work with CRI implementations
  that support the `CheckpointContainer` CRI RPC as well as
  CRI implementations that do not support it. The tests also handle
  if the corresponding feature gate is disabled or enabled:
  <https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/checkpoint_container.go>

- As the tests are hidden behind the feature gate `ContainerCheckpoint` during
  Alpha phase and only available in combination with CRI-O automatic, the tests
  have been skipped so far. With graduation to Beta the tests should appear in
  CRI-O based setups. Due to way the current setup of not running all Alpha
  features enabled with CRI-O, no results have been collected and tests have
  been skipped.

### Graduation Criteria

#### Alpha

- [X] Implement the new feature gate and kubelet implementation
- [X] Ensure proper tests are in place
- [X] Update documentation to make the feature visible
  - <https://kubernetes.io/docs/reference/node/kubelet-checkpoint-api/>
  - <https://kubernetes.io/blog/2022/12/05/forensic-container-checkpointing-alpha/>
  - <https://kubernetes.io/blog/2023/03/10/forensic-container-analysis/>

#### Alpha to Beta Graduation

At least one container engine implemented the corresponding CRI APIs:

- [x] CRI-O

In Kubernetes:

- [x] No major bugs reported in the previous cycle
- [ ] Enable the feature per default
- [ ] Add separate sub-resource permission to control permissions
  at <https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/server/auth.go#L101-L108>
- [ ] Add necessary metrics as described in the PRR sections and update the KEP with the metrics
  names once they exist
  - [ ] Add CRI metrics
  - [ ] Add kubelet metrics (this already exist under the name `checkpoint`)
    <https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/server/server.go#L442>

#### Beta to GA Graduation

CRI-O as well as containerd have to have implemented the corresponding CRI APIs:

- [x] CRI-O
- [ ] containerd (<https://github.com/containerd/containerd/pull/6965>)

Ensure that e2e tests are working with

- [x] CRI-O
- [ ] containerd (<https://github.com/containerd/containerd/pull/6965>)

### Upgrade / Downgrade Strategy

No changes are required on upgrade if the container engine supports
the corresponding CRI API changes.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `ContainerCheckpoint`

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. By disabling the feature gate `ContainerCheckpoint` again.

###### What happens if we reenable the feature if it was previously rolled back?

Checkpointing containers will be possible again.

###### Are there any tests for feature enablement/disablement?

Currently the test will automatically be skipped if the feature is not enabled.
Tests will be extended to explicitly test if the feature is disabled as well
as if it is enabled.

### Rollout, Upgrade and Rollback Planning

If it is not enabled via the feature gate it will return `404 page not found`.
If it is not enabled in the underlying container engine a `500` will be returned
with an error message from the container engine. If it is enabled the API
endpoint exists if disabled then it does not exist. No planning necessary.

Documented at <https://kubernetes.io/docs/reference/node/kubelet-checkpoint-api/>
<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

The feature depends on the existence of the `CheckpointContainer` CRI RPC.
If the underlying container engine does not support or if it is not implemented,
the kubelet API endpoint will fail with `500`. The error code is the same if the
container engine does not implement it explicitly or if the underlying container
engine is too old. The difference between does two failures are only visible in
the error message returned from the container engine.

If the underlying container engine does not support the CRI RPC the kubelet API endpoint
will always return `500`.

It cannot directly impact running workloads, but if the *kubelet* API endpoint is
called if the underlying container engine does no longer support it, the checkpoint
request will fail.

###### What specific metrics should inform a rollback?

CRI metrics will be added to track checkpointing failures to inform a rollback
decision.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No data is stored, so re-enabling starts from a clean slate.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

Querying the state of the feature gate offers the possibility to detect
if the API endpoint will return `404` or not.

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

As it is not exposed in the Kubernetes API it cannot be determined. This is
only visible in the kubelet. Also, this is a feature workloads are not using
directly, but which is only external entities can trigger. Access to the
*kubelet* API endpoints is needed. It might be detectable if operators can
query the state of different feature gates. An operator could also use metrics
to determine that this feature is in use. Metrics seem to be already collected
at <https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/server/server.go#L442>.

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

The kubelet API endpoint can return following codes:

- 200: checkpoint archive was successfully created
- 404: feature is not enabled
- 500: underlying container engine does not support checkpointing containers

Documented at <https://kubernetes.io/docs/reference/node/kubelet-checkpoint-api/>

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The expectation is that it should always succeed. A failed checkpoint does not
break the actual workload. A failed checkpoint only means that the checkpoint
request failed without effects on the workload. The expectation is also that
checkpointing either is always successful or never. From today's point of view this
means that the expectation is 100% availability or 0% availability. Experience
in Podman/Docker and other container engines so far indicates that.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Currently the *kubelet* collects metrics in the bucket `checkpoint`. This can be
used to determine the health of the service.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

CRI stats will be added for this as well as kubelet metrics tracking whether an
operation failed or succeeded.

### Dependencies

CRIU needs to be installed on the node, but on most distributions it is already
a dependency of runc/crun. It does not require any specific services on the
cluster.

###### Does this feature depend on any specific services running in the cluster?

Yes, the container engine must support the checkpoint CRI API call.

### Scalability

###### Will enabling / using this feature result in any new API calls?

The newly introduced CRI API call to checkpoint a container/pod will be
used by this feature. The kubelet will make the CRI API calls and it
will only be done when a checkpoint is triggered. No periodic API calls
will happen.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. It will only affect checkpoint CRI API calls.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

During checkpointing each memory page will be written to disk. Disk usage will increase by
the size of all memory pages in the checkpointed container. Each file in the container that
has been changed compared to the original version will also be part of the checkpoint.
Disk usage will overall increase by the used memory of the container and the changed files.
Checkpoint archive written to disk can optionally be compressed. The current implementation
does not compress the checkpoint archive on disk.  The cluster administrator is
responsible for monitoring disk usage and removing excess data.

The kubelet will request a checkpoint from the underlying CRI implementation. In
the checkpoint request the kubelet will specify the name of the checkpoint
archive as `checkpoint-<podFullName>-<containerName>-<timestamp>.tar` and also
request to store the checkpoint archive in the `checkpoints` directory below its
root directory (as defined by `--root-dir`). This defaults to
`/var/lib/kubelet/checkpoints`.

To avoid running out of disk space an operator has been introduced: <https://github.com/checkpoint-restore/checkpoint-restore-operator>

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

During checkpointing each memory page will be written to disk. Disk usage will increase by
the size of all memory pages in the checkpointed container. Each file in the container that
has been changed compared to the original version will also be part of the checkpoint.
Disk usage will overall increase by the used memory of the container and the changed files.
Checkpoint archive written to disk can optionally be compressed. The current implementation
does not compress the checkpoint archive on disk.  The cluster administrator is
responsible for monitoring disk usage and removing excess data.

The kubelet will request a checkpoint from the underlying CRI implementation. In
the checkpoint request the kubelet will specify the name of the checkpoint
archive as `checkpoint-<podFullName>-<containerName>-<timestamp>.tar` and also
request to store the checkpoint archive in the `checkpoints` directory below its
root directory (as defined by `--root-dir`). This defaults to
`/var/lib/kubelet/checkpoints`.

To avoid running out of disk space an operator has been introduced: <https://github.com/checkpoint-restore/checkpoint-restore-operator>

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

Like any other kubelet API endpoint this will fail if the API server is not available.

###### What are other known failure modes?

- The creation of the checkpoint archive can fail.
  - Detection: Possible return codes are:
    - 401: Unauthorized
    - 404: Not Found (if the ContainerCheckpoint feature gate is disabled)
    - 404: Not Found (if the specified namespace, pod or container cannot be found)
    - 500: Internal Server Error (if the CRI implementation encounter an error
    during checkpointing (see error message for further details))
    - 500: Internal Server Error (if the CRI implementation does not implement
    the checkpoint CRI API (see error message for further details))
    - Also see: https://kubernetes.io/docs/reference/node/kubelet-checkpoint-api/
  - Mitigation: Do not checkpoint a container that cannot be checkpointed by CRIU.
  - Diagnostics: The container engine will provide the location of log file created
    by CRIU with more details.
  - Testing: Tests are currently covering if checkpointing is enabled in the kubelet
    or not as well as covering if the underlying container engine supports the
    corresponding CRI API calls. The most common checkpointing failure is if the
    container is using an external hardware device like a GPU or InfiniBand which
    usually do not exist in test systems.

Checkpointing anything with access to an external hardware device like a GPU or
InfiniBand can fail. For each device a specific plugin needs to be added to CRIU.
For AMD GPUs this exists already today, but other GPUs will fail to be checkpointed.
If an unsupported device is used in the container the *kubelet* API endpoint will
return `500` with additional information in the error message.

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

As checkpointing is an optional feature outside of the pod lifecycle SLOs probably should
not be impacted. If SLOs are impacted then administrators should no longer call the
checkpoint *kubelet* API endpoint. During Alpha and Beta phase the feature gate can
also be used to turn the feature of. At this point in time it is unclear, but for a
possible GA phase this is maybe a feature that needs to be opt-in or opt-out. Something
that can be turned off during startup or runtime configuration.

## Implementation History

* 2020-09-16: Initial version of this KEP
* 2020-12-10: Opened pull request showing an end-to-end implementation of a possible use case
* 2021-02-12: Changed KEP to mention the *experimental* API as suggested in the SIG Node meeting 2021-02-09
* 2021-04-08: Added section about Pod Lifecycle, Checkpoint Storage, Alternatives and Hooks
* 2021-07-08: Reworked structure and added missing details
* 2021-08-03: Added the forensic user story and highlight the goal to implement it in small steps
* 2021-08-10: Added future work with information about pod level cgroup freezing
* 2021-09-15: Removed references to first proof of concept implementation
* 2021-09-21: Mention feature gate `ContainerCheckpointRestore`
* 2021-09-22: Removed everything which is not directly related to the forensic use case
* 2022-01-06: Reworked based on review
* 2022-01-20: Reworked based on review and renamed feature gate to `ContainerCheckpoint`
* 2022-04-05: Added CRI API section and targeted 1.25
* 2022-05-17: Remove *restore* RPC from the CRI API
* 2024-02-08: Graduation to Beta.

## Drawbacks

During checkpointing each memory page of the checkpointed container is written to disk
which can result in slightly lower performance because each memory page is copied
to disk. It can also result in increased disk IO operations during checkpoint
creation.

In the current CRI-O implementation the checkpoint archive is created so that only
the `root` user can access it. As the checkpoint archive contains all memory pages
a checkpoint archive can potentially contain secrets which are expected to be
in memory only.

The current CRI-O implementations handles SELinux labels as well as seccomp and restores
these setting as they were before. A possibly restored container is as secure as
before, but it is important to be careful where the checkpoint archive is stored.

During checkpointing CRIU injects parasite code into the to be checkpointed process.
On a SELinux enabled system the access to the parasite code is limited to the
label of corresponding container. On a non SELinux system it is limited to the
`root` user (which can access the process in any way).

## Alternatives

Another possibility to use checkpoint restore would be, for example, to trigger
the checkpoint by a privileged sidecar container (`CAP_SYS_ADMIN`) and do the
restore through an Init container.

The reason to integrate checkpoint restore directly into Kubernetes and not
with helpers like sidecar and init containers is that checkpointing is already,
for many years, deeply integrated into multiple container runtimes and engines
and this integration has been reliable and well tested. Going another way in
Kubernetes would make the whole process much more complicated and fragile. Not
using checkpoint and restore in Kubernetes through the existing paths of
runtimes and engines is not well known and maybe not even possible as
checkpointing and restoring is tightly integrated as it requires much
information only available by working closely with runtimes and engines.

[kubernetes-blog-post]: https://kubernetes.io/blog/2022/12/05/forensic-container-checkpointing-alpha/
[migration-issue]: https://github.com/kubernetes/kubernetes/issues/3949
[task-migration]: https://lpc.events/event/2/contributions/69/attachments/205/374/Task_Migration_at_Scale_Using_CRIU_-_LPC_2018.pdf
[singularity]: https://arxiv.org/abs/2202.07848
[kubectl-checkpoint]: https://github.com/kubernetes/kubernetes/pull/120898
[checkpoint-management]: https://github.com/kubernetes/kubernetes/pull/115888
