# KEP-1967: Sizable memory backed volumes

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
    - [How can this feature be enabled / disabled in a live cluster?](#how-can-this-feature-be-enabled--disabled-in-a-live-cluster)
    - [Does enabling the feature change any default behavior?](#does-enabling-the-feature-change-any-default-behavior)
    - [Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement?](#can-the-feature-be-disabled-once-it-has-been-enabled-ie-can-we-roll-back-the-enablement)
    - [What happens if we reenable the feature if it was previously rolled back?](#what-happens-if-we-reenable-the-feature-if-it-was-previously-rolled-back)
    - [Are there any tests for feature enablement/disablement?](#are-there-any-tests-for-feature-enablementdisablement)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
    - [How can a rollout fail? Can it impact already running workloads?](#how-can-a-rollout-fail-can-it-impact-already-running-workloads)
    - [What specific metrics should inform a rollback?](#what-specific-metrics-should-inform-a-rollback)
    - [Were upgrade and rollback tested? Was the upgrade-&gt;downgrade-&gt;upgrade path tested?](#were-upgrade-and-rollback-tested-was-the-upgrade-downgrade-upgrade-path-tested)
    - [Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?](#is-the-rollout-accompanied-by-any-deprecations-andor-removals-of-features-apis-fields-of-api-types-flags-etc)
  - [Monitoring Requirements](#monitoring-requirements)
    - [How can an operator determine if the feature is in use by workloads?](#how-can-an-operator-determine-if-the-feature-is-in-use-by-workloads)
    - [What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?](#what-are-the-slis-service-level-indicators-an-operator-can-use-to-determine-the-health-of-the-service)
    - [How can someone using this feature know that it is working for their instance?](#how-can-someone-using-this-feature-know-that-it-is-working-for-their-instance)
    - [What are the reasonable SLOs (Service Level Objectives) for the above SLIs?](#what-are-the-reasonable-slos-service-level-objectives-for-the-above-slis)
    - [Are there any missing metrics that would be useful to have to improve observability of this feature?](#are-there-any-missing-metrics-that-would-be-useful-to-have-to-improve-observability-of-this-feature)
  - [Dependencies](#dependencies)
    - [Does this feature depend on any specific services running in the cluster?](#does-this-feature-depend-on-any-specific-services-running-in-the-cluster)
  - [Scalability](#scalability)
    - [Will enabling / using this feature result in any new API calls?](#will-enabling--using-this-feature-result-in-any-new-api-calls)
    - [Will enabling / using this feature result in introducing new API types?](#will-enabling--using-this-feature-result-in-introducing-new-api-types)
    - [Will enabling / using this feature result in any new calls to the cloud provider?](#will-enabling--using-this-feature-result-in-any-new-calls-to-the-cloud-provider)
    - [Will enabling / using this feature result in increasing size or count of the existing API objects?](#will-enabling--using-this-feature-result-in-increasing-size-or-count-of-the-existing-api-objects)
    - [Will enabling / using this feature result in increasing time taken by any operations covered by [existing SLIs/SLOs]?](#will-enabling--using-this-feature-result-in-increasing-time-taken-by-any-operations-covered-by-existing-slisslos)
    - [Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?](#will-enabling--using-this-feature-result-in-non-negligible-increase-of-resource-usage-cpu-ram-disk-io--in-any-components)
    - [Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?](#can-enabling--using-this-feature-result-in-resource-exhaustion-of-some-node-resources-pids-sockets-inodes-etc)
  - [Troubleshooting](#troubleshooting)
    - [How does this feature react if the API server and/or etcd is unavailable?](#how-does-this-feature-react-if-the-api-server-andor-etcd-is-unavailable)
    - [What are other known failure modes?](#what-are-other-known-failure-modes)
    - [What steps should be taken if SLOs are not being met to determine the problem?](#what-steps-should-be-taken-if-slos-are-not-being-met-to-determine-the-problem)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This KEP improves the portability of pod definitions that use memory backed empty dir
volumes by sizing an empty dir memory backed volume as the minimum of pod allocatable
memory on a host and an optional explicit user provided value.

## Motivation

Kubernetes supports emptyDir volumes whose backing storage is memory (i.e. tmpfs).
The size of this memory backed volume is defaulted to 50% of the memory on a Linux host.
The coupling of default memory backed volume size with the host that runs the pod makes
pod definitions less portable across node instance types and providers.

This impacts workloads that make heavy use of /dev/shm or other use cases oriented around
memory backed volume usage (AI/ML, etc.)

### Goals

- Size a memory backed volume to match the pod allocatable memory
- Enable a user to size the memory backed volume less than the pod allocatable memory

### Non-Goals

- Address memory chargeback of empty dir volumes across container restarts

## Proposal

Define a new feature gate: `SizeMemoryBackedVolumes`.

If enabled, the `kubelet` will change the behavior when building memory backed
volume to specify a non-zero size that is the following:

`min(nodeAllocatable[memory], podAllocatable[memory], emptyDir.sizeLimit)`

This is an improvement over present behavior as pods will see emptyDir memory
backed volumes sized based on actual allowed usage rather than a heuristic
based on the node that is executing the pod.

### Risks and Mitigations

The risks for this proposal are minimal.

The empty dir volume will now be sized consistently with pod level cgroup
memory limit.  A container that writes to a memory backed volume is charged
for that write while accounting memory.  If a container restarts, the charge
goes to the pod cgroup.  Sizing the emptyDir volume to match the actual amount
of memory that can be charged to a pod basically avoids undersizing or oversizing
the appearance of more memory.

## Design Details

The design for this implementation makes the existing `emptyDir.sizeLimit`
not just used during eviction heuristics, but for sizing of the volume.
Since the user is unable to write more to the volume than what the pod
cgroup bounds, there is no material difference to enforcement around
memory consumption, it just provides better sizing across node types.

### Test Plan

##### Unit tests

- k8s.io/kubernetes/pkg/volume/emptydir: October 9th 2024 - 59.8

##### Integration tests

Sig-node related tests are tested with e2e tests.

##### e2e tests

Node e2e testing will capture the following:

- verify empty dir volume size matches sizeLimit (if specified) OR
- verify empty dir volume size matches pod available memory

To verify the pod available memory scenario, we will verify the
memory backed volume size is equivalent to the pod cgroup memory
or node allocatable memory limit.

We have eviction tests that make sure size backed volumes cannot exceed the size limit.

See [eviction test](https://github.com/kubernetes/kubernetes/blob/b2031b3cb46e946ee72eab7bda87b046db138d62/test/e2e_node/eviction_test.go#L366).

### Graduation Criteria

#### Alpha -> Beta Graduation

- All feedback gathered from users of memory backed volumes (expect to be minimal)
- Adequate test signal quality for node e2e
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- Allowing time for additional user feedback and bug reports

### Upgrade / Downgrade Strategy

Not applicable.

The `kubelet` will size the memory backed volume to map how writes
are charged.  If downgrade to a prior kubelet, the volume size would
default to linux host behavior.

### Version Skew Strategy

The feature changes the operating environment presented to a pod,
so a pod will either get an accurate empty dir volume size, or a
potentially inaccurate volume size based on node configuration.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

#### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: SizeMemoryBackedVolumes
  - Components depending on the feature gate: kubelet
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No

#### Does enabling the feature change any default behavior?

Yes, the kubelet will size the empty dir volume to match the precise
amount of memory the pod is able to write rather than over or undersizing.
Prior behavior is node dependent, and so pod authors had no mechanism
to control this behavior properly.

#### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement?

Yes

#### What happens if we reenable the feature if it was previously rolled back?

Pods that run on that node will have memory backed volumes sized based on Linux
host default.  The sizing may not align with actual available memory for an app.

#### Are there any tests for feature enablement/disablement?

Yes, we have unit tests in k8s.io/kubernetes/pkg/volume/emptydir that verify the sizes based on the feature gate.

### Rollout, Upgrade and Rollback Planning

#### How can a rollout fail? Can it impact already running workloads?

If a pod has more allocatable memory than the default node instance behavior
of taking 50% node instance memory for sizing emptyDir, a pod could potentially
write more content to the empty dir volume than previously.  This should have
no impact on rollout of the cluster or workload.  In practice, applications
that did exhaust the size of the memory backed volume were not portable across
instance types or would have had to handle running out of room in that volume.

#### What specific metrics should inform a rollback?

None.

#### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Kind was used to verify the following path.

The following kind configs were used to provision a cluster with this feature gate off:

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
featureGates:
  # any feature gate can be enabled here with "Name": true
  # or disabled here with "Name": false
  # not all feature gates are tested, however
  "SizeMemoryBackedVolumes": false
```

The following pod yaml was used to verify size limits for emptyDir volumes:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: example
  labels:
    app: test-pd
spec:
  securityContext:
    seccompProfile:
      type: RuntimeDefault
  containers:
  - image: busybox
    command:
      - /bin/sh
      - -c
      - |
         sleep infinity
    name: test-pd
    resources:
      limits:
        memory: 2Gi
    volumeMounts:
    - mountPath: /dev/shm
      name: dshm
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
  volumes:
  - name: dshm
    emptyDir:
      medium: Memory
```

With this feature off, we can confirm that the volumes do not obey the pod memory limits:

```bash
~/Work/KubeExamples/EmptyDirTest$ k exec -it example -- df -h /dev/shm
Filesystem                Size      Used Available Use% Mounted on
tmpfs                    31.2G         0     31.2G   0% /dev/shm
```

With the feature on, the volumes are sized to the pod memory limit:

```bash
~/Work/KubeExamples/EmptyDirTest$ k exec -it example -- df -h /dev/shm
Filesystem                Size      Used Available Use% Mounted on
tmpfs                     2.0G         0      2.0G   0% /dev/shm
```

#### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

#### How can an operator determine if the feature is in use by workloads?

An operator can audit for pods whose emptyDir medium is memory and a size limit
is specified.  It's not clear there is a benefit to track this because it only
impacts how the kubelet better enforces an existing API.

#### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

{Pod startup latency}[https://github.com/kubernetes/community/blob/master/sig-scalability/slos/pod_startup_latency.md] can be used for stateless pods.
By definition, emptyDirs are still considered to be stateless so the official metric for stateless pods is sufficient.

#### How can someone using this feature know that it is working for their instance?

- [x] Other
  - Details: An operator can audit for pods whose emptyDir medium is memory and a size limit
  is specified.

#### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

{Pod startup latency}[https://github.com/kubernetes/community/blob/master/sig-scalability/slos/pod_startup_latency.md] can be used for stateless pods.
By definition, emptyDirs are still considered to be stateless so the official metric for stateless pods is sufficient.

#### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

#### Does this feature depend on any specific services running in the cluster?

No

### Scalability

#### Will enabling / using this feature result in any new API calls?

No.

#### Will enabling / using this feature result in introducing new API types?

No

#### Will enabling / using this feature result in any new calls to the cloud provider?

No

#### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

#### Will enabling / using this feature result in increasing time taken by any operations covered by [existing SLIs/SLOs]?

No

#### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

#### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

#### How does this feature react if the API server and/or etcd is unavailable?

No impact.

#### What are other known failure modes?

Not applicable.

#### What steps should be taken if SLOs are not being met to determine the problem?

Not applicable

## Implementation History

v1.20: Launched to `Alpha`
v1.22: Moved to `Beta`
v1.32: Moved to `Stable`

## Drawbacks

None.

This eliminates an unintentional coupling of pod and node.

## Alternatives

None.

## Infrastructure Needed (Optional)

None.
