<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-6395: Dynamic Node-Local Ephemeral Volumes

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Autonomous Framework Governance &amp; Workspace Swapping](#story-1-autonomous-framework-governance--workspace-swapping)
    - [Story 2: Pre-Warmed Sandbox Pods for Agentic Workloads](#story-2-pre-warmed-sandbox-pods-for-agentic-workloads)
    - [Story 3: In-Place Sidecar &amp; Workload Upgrades](#story-3-in-place-sidecar--workload-upgrades)
    - [Story 4: Heterogeneous Task Workers &amp; Pipeline Runners](#story-4-heterogeneous-task-workers--pipeline-runners)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Third-party controllers assuming static pod volumes](#third-party-controllers-assuming-static-pod-volumes)
    - [Resource exhaustion and eviction threshold interference](#resource-exhaustion-and-eviction-threshold-interference)
    - [Security implications of dynamic <code>hostPath</code> volumes](#security-implications-of-dynamic-hostpath-volumes)
    - [Dynamic Secret exfiltration and unmount (&quot;Secret Scrubbing&quot;)](#dynamic-secret-exfiltration-and-unmount-secret-scrubbing)
    - [Desynchronized Secret lifetime during blocked host unmount](#desynchronized-secret-lifetime-during-blocked-host-unmount)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
    - [VolumeMount Dynamic Policy](#volumemount-dynamic-policy)
    - [Integration with the <code>/dynamic</code> Subresource](#integration-with-the-dynamic-subresource)
    - [Inspection via the <code>/allocated</code> Subresource](#inspection-via-the-allocated-subresource)
    - [Pod Volume Status API (<code>pod.Status.VolumeStatuses</code>)](#pod-volume-status-api-podstatusvolumestatuses)
      - [Lifecycle Phases](#lifecycle-phases)
      - [Sample <code>pod.Status</code> Lifecycle States](#sample-podstatus-lifecycle-states)
  - [API Server Validation Rules](#api-server-validation-rules)
  - [Two-Stage Kubelet Lifecycle: Allocation and Actuation](#two-stage-kubelet-lifecycle-allocation-and-actuation)
    - [Stage 1: Admission and Allocation (Node-Local)](#stage-1-admission-and-allocation-node-local)
    - [Stage 2: Actuation (Container Runtime)](#stage-2-actuation-container-runtime)
  - [Volume Teardown and Removal Lifecycle](#volume-teardown-and-removal-lifecycle)
  - [Limitations](#limitations)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
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
  - [Alternative 1: Out-of-band HostPath Bind Mounts](#alternative-1-out-of-band-hostpath-bind-mounts)
  - [Alternative 2: Full Pod Recreation](#alternative-2-full-pod-recreation)
  - [Alternative 3: Dedicated Dynamic Volume Container Type](#alternative-3-dedicated-dynamic-volume-container-type)
  - [Alternative 4: Live Hot-Plug into Running Containers for Alpha](#alternative-4-live-hot-plug-into-running-containers-for-alpha)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone /
release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in
  [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and
  SIG Testing input (including test refactors)
  - [ ] e2e Tests for all features
  - [ ] Tests run automatically on PRs
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date
- [ ] User-facing documentation has been created in [kubernetes/website], for
  publication to [kubernetes.io]
- [ ] Supporting documentation (e.g., additional design documents, links to
  mailing list discussions/SIG meetings, issues before this KEP) is included to
  the "Replaces" - "See Also" list

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

## Summary

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.
-->

This KEP enables Kubernetes workloads to dynamically add and remove
**node-local ephemeral volumes** (`configMap`, `secret`, `projected`,
`emptyDir`, and OCI `image`) to and from pods via the `/dynamic` subresource
without requiring the entire pod to be destroyed and recreated.

For the **Alpha** release, volume mutations actuate upon:
1. **Container Restart**: Modifying `volumeMounts` on an existing container via
   `/dynamic` explicitly triggers Kubelet to recreate the container.
2. **Container Addition**: Mounting newly added or existing volumes into
   dynamically added containers (e.g., dynamic containers via
   [KEP-5972: Dynamic Containers](https://github.com/kubernetes/enhancements/pull/6169)
   or ephemeral containers).
3. **Container Removal**: Safely unmounting and tearing down volumes once all
   containers referencing them have terminated.

Live hot-plug into running containers without restart is an explicit non-goal
for Alpha and will be reevaluated for Beta. This enhancement requires no changes
to the Container Runtime Interface (CRI), does not interact with control-plane
storage controllers (`AttachDetachController`, `PVController`), and does not
require scheduler coordination.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

In Kubernetes today, a Pod's storage configuration is strictly immutable once
scheduled. `pod.Spec.Volumes` cannot be altered, and container `volumeMounts`
cannot be updated. While this static model was sufficient for traditional
stateless microservices, it presents a severe bottleneck for modern workloads:

1. **Heterogeneous Task Workers & Batch Runners**: Any workload architecture
   utilizing "worker" pods or containers that execute varying sequential tasks
   benefits from dynamic volume swapping. In batch processing frameworks,
   continuous delivery runners, and generic worker pools, pods remain running to
   amortize scheduling, admission, image pull, and sandbox runtime initialization
   costs. As different tasks arrive, each task demands its own configuration,
   credentials, or scratch datasets. Dynamic volumes allow worker pods to swap in
   the volumes required for each specific task without pod recreation.
2. **Sandboxed Workload Runners & Agentic Sandboxes**: Modern sandboxed workload
   runners host long-running containers (e.g., sandboxed runners) that execute
   distinct jobs sequentially. Each job requires an isolated workspace volume
   (e.g., `emptyDir` scratch or projected tokens). Because volumes are
   immutable, platforms are forced to choose between deleting and recreating
   pods for every job (incurring seconds to minutes of scheduling and runtime
   initialization overhead) or using insecure out-of-band `hostPath` bind mounts
   that bypass Kubernetes security, quotas, and authorizers.
3. **Pre-Warmed Sandbox Pods**: Systems pre-warm pods on nodes with pre-pulled
   container images and initialized runtimes. When a tenant workload arrives,
   tenant-specific configuration (`configMap`, `secret`, `projected`) must be
   injected into the pre-warmed pod without destroying the warm sandbox.
4. **Dynamic Containers Synergy**: Dynamic Containers introduces the
   `/dynamic` subresource to add and remove containers dynamically. However,
   Dynamic Containers explicitly constrains dynamic containers to only mount
   volumes already defined at pod creation. This proposal removes this
   limitation, allowing dynamic containers to bring their own volumes.

By allowing node-local volumes to be added and removed dynamically, workloads
can mutate storage definitions natively through the Kubernetes API with
sub-second turnaround times.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

* Enable dynamically adding and removing node-local volumes (`configMap`,
  `secret`, `projected`, `emptyDir`, and OCI `image`) in `pod.Spec.Volumes` on
  running pods via the `/dynamic` subresource.
* Enable adding, removing, and updating `volumeMounts` in
  `container.VolumeMounts` for containers that are being restarted, added
  (ephemeral or dynamic containers), or removed.
* Maintain complete backward compatibility for standard `/pods` updates,
  ensuring default immutability guarantees remain intact.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

* **Live Hot-Plug into Running Containers (Alpha)**: Hot-mounting or
  hot-unmounting filesystems into currently running container processes without
  container restart is explicitly out of scope for Alpha. We will reevaluate
  this limitation for Beta.
* **HostPath Volumes (Alpha)**: Dynamic addition of `hostPath` volumes is out of
  scope for Alpha due to security implications. We will
  reevaluate supporting `hostPath` volumes for Beta.
* **PersistentVolumeClaims (PVCs)**: Dynamic addition and removal of PVCs,
  including local PVs and cloud block volumes, is out of scope and deferred to
  follow-up KEPs.
* **Volume Resizing**: Resizing existing volumes in-place is out of scope.
* **Kube-Scheduler Changes**: Inline node-local ephemeral volumes do not have
  persistent capacity tracking or node affinity; scheduling is unaffected.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

We propose extending the `/dynamic` subresource introduced in
[KEP-5972: Dynamic Containers](https://github.com/kubernetes/enhancements/pull/6169)
to permit mutations to `spec.volumes` and container
`spec.containers[*].volumeMounts` for supported node-local volume types.

When a client submits an update to `/dynamic`:
1. The API Server validates that newly added volumes are of supported types
   (`configMap`, `secret`, `projected`, `emptyDir`, `image`) and that any
   removed volumes are no longer referenced by active containers.
1. The Node Authorizer updates its graph, granting the assigned node
   authorization to read newly mounted Secrets or ConfigMaps.
1. Kubelet admits the request through an **Allocation** step, recording the
   admitted volume and container specifications into its internal state, which
   is exposed via the `/allocated` subresource.
1. Kubelet prepares and mounts the volume on the host filesystem.
1. In the **Actuation** step, Kubelet mounts or unmounts volumes as needed, triggering a restart if volume mounts are modified on an existing container.
1. When a volume is removed from `pod.Spec.Volumes`, Kubelet tears down the
   volume mount on the host once all containers referencing it have stopped, after which the Node Authorizer graph removes authorization edges for those Secrets or ConfigMaps.

### User Stories

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1: Autonomous Framework Governance & Workspace Swapping
An autonomous orchestrator or local runner maintains warm sandbox pods. As jobs
transition, the runner swaps out the tenant workspace. The runner submits a
request to `/dynamic` that removes the previous job's `emptyDir` scratch
volume, adds a new `emptyDir` volume, and restarts the task container.
The container starts up immediately with the fresh volume, avoiding a full 
pod recreation cycle.

#### Story 2: Pre-Warmed Sandbox Pods for Agentic Workloads
An agent platform pre-warms sandboxed runner pods on worker nodes. When an agent
requests a tool execution environment, the platform dynamically adds a
`projected` volume containing ephemeral credentials, tool configuration, and a
scratch disk, and launches a dynamic sidecar container referencing those mounts.
Execution begins in under 500ms.

#### Story 3: In-Place Sidecar & Workload Upgrades
A credential-rotation or logging sidecar needs to be upgraded or rotated. The
operator patches the pod via `/dynamic` to attach an updated `ConfigMap` volume
and restarts the sidecar container in-place, without disturbing the primary
application container running in the same pod.

#### Story 4: Heterogeneous Task Workers & Pipeline Runners
A long-running worker pod in a distributed data processing system or CI/CD
pipeline polls a queue for tasks. For Task 1, the orchestrator issues a
`/dynamic` update to mount a specific task `ConfigMap` and an `emptyDir` scratch
volume, triggering an explicit restart of the task worker container. When Task 1
completes, the orchestrator removes the scratch volume, mounts fresh task
credentials for Task 2, and continues execution within the same pod—eliminating
pod scheduling, CNI network setup, and container image pull latencies between
tasks.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

#### Third-party controllers assuming static pod volumes
* **Risk**: Admission webhooks, GitOps agents, or third-party controllers may
  assume that `.spec.volumes` is immutable after pod creation. An unexpected
  update could cause panics or desynchronized state in external controllers.
* **Mitigation**: Dynamic volume updates are strictly forbidden on the main
  `/pods` endpoint and require explicit routing to the `/dynamic` subresource
  (see
  [KEP-5972: Dynamic Containers](https://github.com/kubernetes/enhancements/pull/6169)
  for further details on subresource access control and ecosystem migration).
  RBAC permissions on `/dynamic` are restricted to cluster administrators by
  default. The feature is gated behind an opt-in feature gate
  (`DynamicNodeLocalEphemeralVolumes`).

#### Resource exhaustion and eviction threshold interference
* **Risk**: Uncontrolled addition of disk-backed or memory-backed `emptyDir`
  volumes could exhaust node ephemeral storage or memory, triggering node-level
  evictions.
* **Mitigation**: Memory-backed `emptyDir` volumes are accounted against
  container and pod memory limits. Disk-backed `emptyDir` volumes are subject to
  project quotas where supported, and Kubelet's existing ephemeral storage
  eviction manager monitors usage against node thresholds.

#### Security implications of dynamic `hostPath` volumes
* **Risk**: Pod Security Standards (PSA) enforce that pods admitted under the
  `baseline` or `restricted` profiles cannot mount `hostPath` volumes. If
  `hostPath` volumes were permitted to be dynamically added via `/dynamic` to an
  already-running pod, it would bypass initial creation-time PSA admission
  controls, enabling an attacker or privileged container to mount arbitrary node
  host paths.
  Additionally, `hostPath` volumes have no integration with the API Server's
  NodeAuthorizer graph.
* **Mitigation**: `hostPath` volumes are strictly rejected during API validation
  for the `/dynamic` subresource in Alpha. Any future `hostPath`
  support will require further security evaluation and review.

#### Dynamic Secret exfiltration and unmount ("Secret Scrubbing")
* **Risk**: A compromised or malicious container temporarily mounts a sensitive
  `Secret` volume via `/dynamic`, copies the secret payload to an unencrypted
  `emptyDir` scratch volume or transmits it across the network, and immediately
  removes the Secret volume from `pod.Spec.Volumes` to conceal evidence of having
  accessed the secret from future `kubectl get pod` audits.
* **Mitigation**: Once container code executes, Kubernetes cannot prevent an
  in-memory copy of secret data; therefore, protection relies on strict API
  access boundaries and auditability. The Kubernetes API Audit Log captures
  every `/dynamic` subresource request and patch chronologically with complete
  user identity, timestamp, and object payload, preserving an immutable record
  of every dynamically mounted and unmounted volume. Furthermore, RBAC
  permissions on `/dynamic` are restricted to cluster administrators and trusted
  controllers by default.

#### Desynchronized Secret lifetime during blocked host unmount
* **Risk**: A client removes a `Secret` volume from `pod.Spec.Volumes`, but
  unmount or teardown on the node is delayed or blocked (e.g., container restart
  is deferred or an open file descriptor holds the mount). During this interval,
  secret data remains physically resident on the node's filesystem
  (`/var/lib/kubelet/pods/<uid>/volumes/kubernetes.io~secret/<name>`) even though
  `spec.volumes` no longer lists the secret.
* **Mitigation**: Kubelet's `/allocated` subresource maintains the volume in the
  node's admitted state until container termination and host directory unmount
  (`TearDownAt`) are fully verified. Observers inspecting `/allocated` can
  verify active node-level volume allocations. Furthermore, Kubelet secret
  volumes are mounted on RAM-backed `tmpfs` filesystems; when Kubelet executes
  `TearDownAt`, unmounting the `tmpfs` immediately zeroes and releases the
  in-memory secret payload.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### API Changes

This proposal introduces the following API changes:
- A new API field on `v1.VolumeMount`: `dynamicPolicy` (type `*VolumeMountDynamicPolicy`), containing `restartPolicy`
  (type `VolumeMountRestartPolicy`).
- Though not introduced by this proposal, we expand the `/dynamic` subresources introduced in
  [KEP-5972: Dynamic Containers](https://github.com/kubernetes/enhancements/pull/6169),
  expanding their schema allowances to permit node-local ephemeral volume mutations.
- A new API field on `v1.PodStatus`: `volumeStatuses` (type `[]PodVolumeStatus`), analogous to `containerStatuses`,
  to track whether volumes are allocated and prepared on the host node without coarse pod conditions (actuated
  container mounts continue to be reported under `ContainerStatuses[*].VolumeMounts`). 

#### VolumeMount Dynamic Policy

To allow users to govern whether dynamic volume mutations require a container
restart (similar to `resizePolicy` in KEP-1287), a new `dynamicPolicy` field is
added to `v1.VolumeMount`:

```yaml
volumeMounts:
- name: scratch
  mountPath: /mnt/scratch
  dynamicPolicy:
    restartPolicy: RestartContainer
```

In go: 

```go
type VolumeMount struct {
    // ... existing fields ...

    // DynamicPolicy defines dynamic volume mutation behavior for this mount.
    // +featureGate=DynamicNodeLocalEphemeralVolumes
    // +optional
    DynamicPolicy *VolumeMountDynamicPolicy `json:"dynamicPolicy,omitempty"`
}
```

```go
// VolumeMountDynamicPolicy defines dynamic mutation behavior for a volume mount.
type VolumeMountDynamicPolicy struct {
    // RestartPolicy defines whether dynamic volume mutations require restarting
    // the container. Supported values: RestartContainer, NotRequired.
    // Defaults to NotRequired.
    // +optional
    RestartPolicy VolumeMountRestartPolicy `json:"restartPolicy,omitempty"`
}

// VolumeMountRestartPolicy defines the restart behavior applied to a container
// when a volume mount is dynamically mutated.
// +enum
type VolumeMountRestartPolicy string

const (
    // RestartContainer indicates that Kubelet must restart the container
    // in-place to actuate volume mount additions or removals on a running container.
    VolumeMountRestartContainer VolumeMountRestartPolicy = "RestartContainer"

    // NotRequired indicates that the volume mount does not require a container restart.
    // Under NotRequired, volume mounts can only be added when adding a container,
    // and removed when removing a container; adding or removing mounts on a running
    // container with NotRequired is rejected.
    VolumeMountNotRequired VolumeMountRestartPolicy = "NotRequired"
)
```

The semantics of `dynamicPolicy.restartPolicy` are defined as follows:
* **Default (`NotRequired`)**: When `dynamicPolicy` is omitted or
  `dynamicPolicy.restartPolicy` is `NotRequired`, the API server **rejects**
  requests to add or remove volume mounts while the container is running. Under
  policy `NotRequired`, volume mounts can only be added along with a container
  addition, and can only be removed along with a container removal.
* **`RestartContainer`**: Setting `dynamicPolicy.restartPolicy: RestartContainer`
  permits adding or removing volume mounts on an existing running container, and
  explicitly instructs Kubelet to restart (recreate) the container in-place to
  actuate the mount changes.

#### Integration with the `/dynamic` Subresource
[KEP-5972: Dynamic Containers](https://github.com/kubernetes/enhancements/pull/6169)
introduces the `/dynamic` subresource. Dynamic volume mutations are executed
exclusively via `PUT /api/v1/namespaces/{namespace}/pods/{name}/dynamic`.

Dynamic volume mutations permit:
1. Appending new volumes to `pod.Spec.Volumes`.
2. Removing volumes from `pod.Spec.Volumes` if they are no longer referenced by
   active containers.
3. Updating `volumeMounts` within `pod.Spec.Containers[*]` for containers
   undergoing restart or addition.

Any semantics regarding the `/dynamic` subresource that apply to dynamic containers
also apply to dynamic volumes.

#### Inspection via the `/allocated` Subresource
To provide observability into transactional state and decouple desired state
from admitted node state, this proposal integrates with the `/allocated`
subresource introduced by
[KEP-5972: Dynamic Containers](https://github.com/kubernetes/enhancements/pull/6169).

The `/allocated` subresource is served directly by the Kubelet and represents
the current transactional allocation on the node. When a pod is updated via
`/dynamic`:
* `pod.Spec.Volumes` reflects the *desired* volume configuration.
* The `/allocated` pod endpoint reflects the *admitted* volume configuration
  that Kubelet has accepted and is actively preparing or maintaining on the
  node.
* The already-existing `volumeMounts` field under container statuses
  (`pod.Status.ContainerStatuses[*].VolumeMounts`) reflects the actual actuated
  volume mounts in the running container.

#### Pod Volume Status API (`pod.Status.VolumeStatuses`)

To track the status of pod volumes across their allocation and host preparation
lifecycles, this proposal introduces `volumeStatuses` to `v1.PodStatus`.
Analogous to `containerStatuses`, this field provides granular, per-volume
observability into whether volumes are allocated and prepared on the host node.
Container mount actuation is not tracked in this field, but is surfaced via the
preexisting `pod.Status.ContainerStatuses[*].VolumeMounts`. 

The allocated pod, including any dynamically added or removed volumes, is
exposed via the already-existing `/allocated` subresource on Kubelet. 

Detailed failure reasons and diagnostics wil be emitted via Events, which will
be further designed in beta.

```go
type PodStatus struct {
    // ... existing fields ...

    // VolumeStatuses contains the status of volumes in the pod, tracking whether
    // volumes are allocated and prepared on the host node.
    // +listType=map
    // +listMapKey=name
    // +optional
    // +featureGate=DynamicNodeLocalEphemeralVolumes
    VolumeStatuses []PodVolumeStatus `json:"volumeStatuses,omitempty"`
}

// PodVolumeStatus represents the status of an individual volume in a pod.
type PodVolumeStatus struct {
    // Name is the name of the volume matching pod.Spec.Volumes.
    Name string `json:"name"`

    // HostPhase represents the host-level preparation phase of the volume:
    // Unallocated, HostMounted, ErrorMounting, Terminating.
    // +optional
    HostPhase PodVolumeHostPhase `json:"hostPhase,omitempty"`
}

// PodVolumeHostPhase represents the host preparation phase of a volume mounted to a pod.
type PodVolumeHostPhase string

const (
    // PodVolumeHostUnallocated indicates that the volume has been added to pod.Spec.Volumes,
    // but Kubelet has not yet admitted or allocated it on the node.
    PodVolumeHostUnallocated PodVolumeHostPhase = "Unallocated"

    // PodVolumeHostMounted indicates that VolumeManager has successfully prepared and
    // mounted the volume on the node host (marked ready in ActualStateOfWorld).
    PodVolumeHostMounted PodVolumeHostPhase = "HostMounted"

    // PodVolumeHostErrorMounting indicates that Kubelet admitted the volume into allocatedPod,
    // but VolumeManager encountered an error during SetUpAt while preparing or mounting the
    // volume on the host.
    PodVolumeHostErrorMounting PodVolumeHostPhase = "ErrorMounting"

    // PodVolumeHostTerminating indicates that volume removal has been admitted and allocated
    // by Kubelet, and Kubelet is actively unmounting container bind mounts and executing host TearDownAt.
    PodVolumeHostTerminating PodVolumeHostPhase = "Terminating"
)
```

##### Lifecycle Phases

1. **`Unallocated`**: A dynamic volume mutation was submitted in `pod.Spec.Volumes`,
   but Kubelet has not yet admitted it into `allocatedPod` (for example, due to
   node ephemeral storage quota limits or a rapid re-add collision). If admission
   fails, the volume remains in `Unallocated`, and Kubelet emits a corresponding
   event (e.g. `FailedAllocation`).
2. **`HostMounted`**: Kubelet has admitted the volume into `allocatedPod`, `VolumeManager`
   has successfully executed `SetUpAt` on the node host, and the volume path
   (`/var/lib/kubelet/pods/<uid>/volumes/...`) is recorded ready in `ActualStateOfWorld` (ASW).
3. **`ErrorMounting`**: Kubelet admitted the volume into `allocatedPod`, but `VolumeManager`
   encountered an error during `SetUpAt` while preparing or mounting the volume on the host
   (for example, failure to resolve a referenced Secret or ConfigMap, an image pull failure,
   or a filesystem mount syscall failure). This ensures failures during the host mounting
   step are explicitly surfaced rather than leaving the volume in `Unallocated`. Detailed
   error context is recorded in Kubelet Events.
4. **`Terminating`**: The volume removal from `pod.Spec.Volumes` has been successfully
   admitted and committed to `allocatedPod`.  `Terminating` is set after the removal has been admitted and allocated by Kubelet. 
   If a volume removal is part of a transactional dynamic update that is rejected or deferred during
   Kubelet allocation, the volume removal is not committed to `allocatedPod`, and its `hostPhase`
   remains `HostMounted`. Once allocation succeeds, Kubelet retains the volume entry in
   `Terminating` while target containers unmount the volume and `VolumeManager` executes
   host `TearDownAt`. Once host unmount completes and the volume is cleared from ASW, the
   entry is pruned from `VolumeStatuses`.

##### Sample `pod.Status` Lifecycle States

###### State 1: Unallocated Volume and Container
When a dynamic mutation adding a new volume and a container mounting it is submitted
via `/dynamic`, but cannot yet be admitted by Kubelet:

```yaml
status:
  volumeStatuses:
    - name: dynamic-scratch
      hostPhase: Unallocated
  containerStatuses:
    - name: worker
      state:
        waiting:
          reason: Unallocated
          message: "Container addition pending node allocation"
```

###### State 2: Host Mounted, Container Restart Pending
When a dynamic mutation adding a new volume and mounting it into an existing container is submitted, Kubelet admits the volume into `allocatedPod`. `VolumeManager` mounts the host
directory in ASW (`hostPhase: HostMounted`). The following yaml shows the pod status after this step, but before the target container has restarted to pick up the new mount:

```yaml
status:
  volumeStatuses:
    - name: dynamic-scratch
      hostPhase: HostMounted
  containerStatuses:
    - name: runner
      ready: true
      state:
        running:
          startedAt: "2026-09-25T14:00:00Z"
      # volumeMounts still reflects the pre-restart container mounts
      volumeMounts: []
```

###### State 3: Fully Actuated
Kubelet recreates container `runner` with the volume mount. The container runtime actuates the bind mount into the container namespace, and `pod.Status.ContainerStatuses[*].VolumeMounts` is updated:

```yaml
status:
  volumeStatuses:
    - name: dynamic-scratch
      hostPhase: HostMounted
  containerStatuses:
    - name: runner
      ready: true
      state:
        running:
          startedAt: "2026-09-25T14:05:00Z"
      volumeMounts:
        - name: dynamic-scratch
          mountPath: /mnt/scratch
          readOnly: false
```

###### State 4: Host Mount Failure (`ErrorMounting`)
If Kubelet admits the volume into `allocatedPod`, but `VolumeManager` encounters an error during `SetUpAt` on the node:

```yaml
status:
  volumeStatuses:
    - name: dynamic-scratch
      hostPhase: ErrorMounting
  containerStatuses:
    - name: runner
      ready: true
      state:
        running:
          startedAt: "2026-09-25T14:00:00Z"
      volumeMounts: []
```

### API Server Validation Rules

When a client issues a `PUT` to `/dynamic`, the API server validates the
request before persisting the pod specification to etcd:
* **Dynamic Policy Validation**:
  - `volumeMount.dynamicPolicy.restartPolicy` defaults to `NotRequired` if
    `dynamicPolicy` is omitted or unspecified.
  - Adding or removing volume mounts on an already-running container requires
    `dynamicPolicy.restartPolicy: RestartContainer`. If a request attempts to
    add, remove, or mutate mounts on a running container with policy
    `NotRequired`, the update is rejected with a validation error.
  - Adding mounts with `NotRequired` is permitted only when simultaneously adding
    a new container (such as a dynamic or ephemeral container).
* **Exclusion of Init Containers (Alpha)**: Dynamic volume mutations targeting
  `pod.Spec.InitContainers[*]` are strictly rejected in Alpha. Any update
  attempting to add, remove, or mutate `volumeMounts` on init containers returns
  a validation error. Supporting init containers (including restartable sidecar
  init containers) will be reevaluated for Beta.
* **Supported Volume Sources**: Only node-local ephemeral volume types are
  permitted:
  * `v1.VolumeSource.ConfigMap`
  * `v1.VolumeSource.Secret`
  * `v1.VolumeSource.Projected`
  * `v1.VolumeSource.EmptyDir`
  * `v1.VolumeSource.Image`
* **Unsupported Types**: Updates attempting to add `PersistentVolumeClaim`,
  `HostPath`, `CSI`, or cloud volume sources are rejected with a validation
  error.
* **Volume Name Uniqueness and Collision Prevention**:
  - **API Server Validation**: Volume names in `spec.volumes` must remain unique. In addition, the API server rejects adding a new volume if its name is currently present in `pod.Status.VolumeStatuses` with `hostPhase: Terminating` or still actively reported in any container's `pod.Status.ContainerStatuses[*].VolumeMounts`. This prevents race conditions where a client removes a volume and rapidly re-adds one with the same name before container unmounting and host teardown complete, exactly mirroring Dynamic Containers.
* **Dangling Mount Protection**: A volume cannot be deleted from `spec.volumes`
  if any container in `spec.containers` still mounts it, unless that container
  is also being removed or updated to remove the mount in the same transaction.
* **Pod Phase**: Dynamic volume mutations are only allowed while the pod is in
  the `Running` phase and has completed initialization. Once `DeletionTimestamp`
  is set, mutations are rejected.

When pod volumes are updated via `/dynamic`, the Node Authorizer's existing
informer automatically updates its graph to grant the node access to newly
mounted Secrets or ConfigMaps. Any transient propagation delay is safely handled
by Kubelet's volume mounting retry loop.

### Two-Stage Kubelet Lifecycle: Allocation and Actuation

Mutations submitted through the `/dynamic` subresource follow a two-stage
Kubelet lifecycle: **Allocation** and **Actuation**. While the `/dynamic`
subresource itself comes from
[KEP-5972: Dynamic Containers](https://github.com/kubernetes/enhancements/pull/6169),
splitting node admission from container runtime configuration adopts the
two-stage allocation and actuation model established in
[KEP-1287: In-Place Pod Vertical Scaling][kep-1287]. Once the API server
validates the mutation and persists it to etcd, both stages are executed
locally on the node by Kubelet.

```
+-----------------------------------------------------------------------------+
| Stage 1: Allocation (Kubelet Admission & Async Host Preparation)            |
|                                                                             |
| 1. Kubelet admits volume mutation against node-local resources and quotas.  |
| 2. Kubelet checkpoints admitted state to allocatedPod.                      |
| 3. Kubelet reflects admitted volumes in the /allocated pod representation.  |
| 4. VolumeManager DSWP discovers volume in allocatedPod and mounts on host   |
|    asynchronously via SetUpAt in the background.                            |
+-----------------------------------------------------------------------------+
                                       │
                                       ▼
+-----------------------------------------------------------------------------+
| Stage 2: Actuation (Container-Scoped Runtime Configuration)                 |
|                                                                             |
| 1. Initial SyncPod (Pod Startup): Blocks on WaitForAttachAndMount before    |
|    starting any initial containers (preserving standard Kubernetes behavior)|
| 2. Subsequent SyncPods (Dynamic Mutations): Kubelet checks if volumes       |
|    required by target containers are ready in ASW.                          |
| 3. If a dynamic volume is still mounting, only the specific target container|
|    restart/addition is deferred; unaffected containers continue running.    |
| 4. Once ready, Kubelet recreates the target container (or starts the dynamic|
|    container) with bind mounts pointing to the prepared host paths.         |
| 5. Kubelet updates pod.Status.VolumeStatuses (hostPhase) and CRI mounts.   |
+-----------------------------------------------------------------------------+
```

#### Stage 1: Admission and Allocation (Node-Local)

When a pod update arrives on the node via watch from the API server:
1. **Kubelet Admission**: Kubelet validates that referenced Secrets and
   ConfigMaps are resolvable and that local ephemeral storage or memory quota is
   sufficient for requested `emptyDir` volumes.
2. **State Checkpointing**: Kubelet records the admitted volume and container
   specifications into its local state checkpoint (`allocatedPod`).
3. **Subresource Exposure**: Kubelet reflects the admitted configuration in the
   `/allocated` subresource, confirming to external observers that allocation
   succeeded.
4. **VolumeManager Asynchronous Host Preparation**:
   Kubelet's `VolumeManager` is only ever aware of the `allocatedPod`. The
   `DesiredStateOfWorldPopulator` reads volumes exclusively from `allocatedPod`
   rather than unadmitted pod specs. This guarantees that `VolumeManager` never
   prepares or mounts host directories for mutations that failed local
   admission. `VolumeManager` generates the desired state of world (DSW) entry
   and invokes `SetUpAt` on the appropriate volume plugin asynchronously in the
   background.

#### Stage 2: Actuation (Container Runtime)

Once host allocation and directory preparation are underway, volumes are
actuated into target containers during Kubelet's `SyncPod` execution:

1. **Initial Pod Startup vs. Subsequent Dynamic SyncPods**:
   - **Initial Pod Startup**: The very first `SyncPod` invocation for a new pod
     preserves standard Kubernetes behavior: Kubelet blocks on
     `WaitForAttachAndMount` to ensure all creation-time volumes are fully
     mounted on the host before any containers or init containers start.
   - **Subsequent Dynamic SyncPods**: For already-running pods processing
     dynamic mutations, Kubelet avoids globally blocking or stalling the pod
     worker. Instead, during container reconciliation (`computePodActions`),
     Kubelet inspects `ActualStateOfWorld` (ASW) for the specific volumes
     required by each container.
2. **Container-Scoped Deferral & Volume Readiness Trigger**:
   - **Determining Readiness via ASW**: Kubelet determines volume readiness by
     checking VolumeManager's `ActualStateOfWorld` (ASW) cache.
   - **Container-Scoped Deferral**: If a newly added volume required by a container
     has not yet been marked mounted in ASW, Kubelet defers actuation only for
     that specific container. Unaffected containers continue running uninterrupted.
   - **Event-Driven Trigger**: When VolumeManager finishes mounting a volume and
     marks it in ASW, it triggers a pod sync to immediately actuate the deferred container.
3. **Explicit Container Re-creation Decisions**:
   When the required volumes are verified ready in ASW, `computePodActions`
   detects that an existing container's `volumeMounts` in `allocatedPod` differ
   from the running container configuration and schedules an explicit container
   restart (stop and recreate). For newly introduced dynamic or ephemeral
   containers, Kubelet schedules container creation.
4. **CRI Invocation**:
   Kubelet stops the existing container if restarting, constructs the CRI
   container specification with bind mounts pointing to the prepared host paths,
   and invokes CRI `CreateContainer` followed by `StartContainer`.
5. **Status and Actuation Reflection**:
   Kubelet updates `pod.Status.VolumeStatuses` to reflect host-level volume readiness
   (`hostPhase: HostMounted`, or `ErrorMounting` on failure). Actuation into containers
   is reflected exclusively under `pod.Status.ContainerStatuses[*].VolumeMounts` as CRI
   confirms the active bind mounts. User observability into allocation details and failure
   causes is provided via Kubelet Events and the `/allocated` subresource rather than
   bloating `PodStatus`.

### Volume Teardown and Removal Lifecycle

Volume removal follows the same two-stage allocation and actuation pattern,
ensuring that container unmounting precedes host directory teardown:

1. **API Validation**: The API server strictly rejects removing a volume from
   `spec.volumes` if any container in `spec.containers` still mounts it, unless
   that container mount is also removed in the same `/dynamic` update.
2. **Allocation Stage (Atomic Admission)**:
   When a `/dynamic` update removes a volume along with its referencing
   container (or removes the mount from an existing container), Kubelet admits
   the mutation and commits the changes to `allocatedPod` atomically.
3. **Actuation Stage (Container Unmount)**:
   During actuation, Kubelet's `SyncPod` acts on the container first, stopping
   and recreating it with the updated container specification (excluding the
   removed volume mount). The container runtime releases the mount from the
   container's mount namespace.
4. **Teardown & Host Unmount via `PodStateProvider`**:
   `VolumeManager`'s `DesiredStateOfWorldPopulator` (DSWP) observes that the
   volume has been removed from `allocatedPod`. To prevent tearing down host
   directories while containers are still running, DSWP queries Kubelet via an
   extension to the `PodStateProvider` interface:
   ```go
   type PodStateProvider interface {
       // IsVolumeInUseByPod returns true if any running container in the pod
       // currently mounts the specified volume.
       IsVolumeInUseByPod(podUID types.UID, volumeName string) bool
   }
   ```
   Kubelet implements `IsVolumeInUseByPod` by checking its internal,
   in-memory rutime cache of running container statuses. Once CRI
   confirms the old container has stopped, the runtime cache clears the mount
   reference. Only then does DSWP
   remove the volume from `DesiredStateOfWorld` (DSW). The reconciler then unmounts and remove the host directory.
5. **Status Pruning**:
   When a volume removal is submitted via `/dynamic`, its entry in `pod.Status.VolumeStatuses`
   transitions to `hostPhase: Terminating` **only after the removal has been successfully
   allocated** by Kubelet. If the removal is part of a transactional dynamic update that is
   rejected or deferred during Kubelet allocation (for example, if submitted alongside an
   infeasible container resource resize), the volume removal is not committed to
   `allocatedPod`, and its `hostPhase` remains `HostMounted`.
   Once allocated, Kubelet retains `Terminating` while containers are restarted or stopped
   without the volume mount (updating `pod.Status.ContainerStatuses[*].VolumeMounts`).
   Once `PodStateProvider.IsVolumeInUseByPod` returns false and VolumeManager completes host
   `TearDownAt` (clearing ASW), Kubelet prunes the volume entry completely from
   `pod.Status.VolumeStatuses`.

### Limitations

* Actuation occurs only upon container restart, addition (ephemeral or dynamic
  containers), or removal.
* Live hot-plug into running containers without restart (`NotRequired`) is
  deferred to Beta.
* Dynamic volume mutations for `initContainers` (including restartable sidecar
  init containers) are not supported in Alpha and will be reevaluated for Beta.
* Dynamic addition of `hostPath` volumes is not supported in Alpha due to
  security considerations and will be reevaluated for Beta.

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


##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

None.

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

* API validation:
  - Adding supported node-local volume types (`configMap`, `secret`, `emptyDir`,
    `projected`, `image`) via `/dynamic`.
  - Rejection of unsupported types (`PVC`, `hostPath`, `CSI`).
  - Rejection of duplicate volume names and dangling container mounts.
* Node Authorizer:
  - Graph edge creation for dynamically mounted Secrets and ConfigMaps on
    running pods.
* Kubelet VolumeManager populator:
  - Discovery of newly added volumes from `allocatedPod` on running pods.
  - Teardown of removed volumes while containers remain running.
* Volume plugins (`configmap`, `secret`, `emptydir`, `projected`):
  - Setup and teardown lifecycle on dynamic add/remove.

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

* API Server `/dynamic` subresource endpoint:
  - RBAC enforcement (cluster-admin required).
  - Updates to `spec.volumes` and `volumeMounts` succeed.
* Node Authorizer integration:
  - Kubelet can retrieve Secrets mounted dynamically into a running pod.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

* Dynamic `emptyDir` on container restart:
  - Start a pod, dynamically add an `emptyDir` volume, restart the container
    in-place, and verify the container writes data to the new mount.
* Dynamic `ConfigMap` and `Secret` addition:
  - Add a `ConfigMap` and `Secret` volume dynamically, restart the container,
    and verify keys are mapped to files inside the container.
* Dynamic volume removal:
  - Remove a volume from the pod spec, verify the host directory is unmounted
    and cleaned up by Kubelet.
* Synergy with
  [KEP-5972: Dynamic Containers](https://github.com/kubernetes/enhancements/pull/6169):
  - Add a dynamic container and a new volume simultaneously via `/dynamic`;
    verify container starts successfully with the volume mounted.

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/
-->

#### Alpha
* Feature gate `DynamicNodeLocalEphemeralVolumes` implemented (default
  disabled).
* `dynamicPolicy` field added to `v1.VolumeMount` (with nested `restartPolicy`).
* Support for node-local volumes (`configMap`, `secret`, `projected`,
  `emptyDir`, `image`) via `/dynamic`.
* Allocation and actuation on container restart, container addition, and
  container removal.
* Unit, integration, and e2e tests passing.

#### Beta
* Consider enabling live hot-plug into running containers, relaxing validation on
  `volumeMount.dynamicPolicy.restartPolicy` to permit `NotRequired` for live mutations.
* Reevaluate supporting `hostPath` volumes with appropriate security safeguards.
* Reevaluate dynamic volume mutations for `initContainers` (including restartable
  sidecar init containers).
* Explore the feasibility of allowing Kubelet to retry admitting a rapidly re-added
  volume after the previous volume has completed host unmount and cleared ASW.
* Metrics for dynamic volume mount and unmount latency and error rates.
* Evaluate integration with `pod.status.volumeHealth` (KEP-1432: Volume Health Monitor)
  as `CSIVolumeHealth` stabilizes, particularly for reporting underlying volume faults
  alongside dynamic volume lifecycle states.

#### GA
* Allowing time for feedback, with at least 2 release cycles in Beta / enabled
  by default.
* Further GA criteria to be added in the beta update.

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

* **Upgrade**:
  - Enabling the feature gate introduces the `/dynamic` volume capability.
    Existing pods are unaffected.
* **Downgrade**:
  - Disabling the feature gate restores strict immutability. Pods that
    previously added volumes continue running with their current mounts, but
    further `/dynamic` updates to volumes are rejected.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

Version skew will be handled by `NodeDeclaredFeatures`. If a client issues an 
update request to a Node that does not support it, the API server will reject the request. 

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

* **Feature Gate**: `DynamicNodeLocalEphemeralVolumes`
* **Components**: `kube-apiserver`, `kubelet`

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Standard updates to `/pods` remain immutable, but third-party controllers
and ecosystem tools may assume that a pod's `volumes` field is stable and
immutable after creation. Enabling this feature changes that assumption, as
`spec.volumes` can now be mutated dynamically via the `/dynamic` subresource.

The `/dynamic` subresource was intended to capture the intent that "all fields may be made mutable under this subresource", 
so using it for this purpose should not be surprising to users. 

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes. Disabling the feature gate prevents any further dynamic volume mutations
via the API.

###### What happens if we reenable the feature if it was previously rolled back?

The API server resumes accepting dynamic volume updates on `/dynamic`.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

Yes. Unit and integration tests will verify that disabling the feature gate rejects
dynamic volume updates with a feature disabled validation error.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

Yes, updates to `/dynamic` generate write requests and pod update watch events.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No, there are new API types.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No. Node-local volumes have no cloud provider interactions.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.
 
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No. Dynamic volume mutation is a new operation not covered by existing SLIs/SLOs.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

Modest increase in the size of `v1.Pod` objects when additional volume entries
are declared or when the volume restart policy is set.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

Negligible CPU and memory impact on Kubelet. Disk usage corresponds to files
created in `emptyDir`.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No. Dynamic volumes do not affect this kind of node resources.

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

###### What are other known failure modes?

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

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

* **2026-09-21**: KEP created for Alpha.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

* Allowing `.spec.volumes` to mutate departs from Kubernetes''' historical
  assumption of static pod specs, requiring ecosystem controllers to adjust
  their reconciliation models.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### Alternative 1: Out-of-band HostPath Bind Mounts
Workloads mount a shared host directory and create subdirectories manually.
* *Why rejected*: Bypasses Kubernetes RBAC, volume authorization, quotas, and
  security boundaries.

### Alternative 2: Full Pod Recreation
Destroy and recreate the pod whenever storage requirements change.
* *Why rejected*: Unacceptable latency (seconds to minutes) for high-frequency
  agentic and sandboxed workloads.

### Alternative 3: Dedicated Dynamic Volume Container Type
Introduce a new volume type (e.g. `dynamicVolumePool`) that contains a mutable
list of nested volumes.
* *Why rejected*: Adds significant API surface complexity. Mutating
  `spec.volumes` directly via `/dynamic` provides a cleaner, uniform experience
  that naturally mirrors container dynamism in
  [KEP-5972: Dynamic Containers](https://github.com/kubernetes/enhancements/pull/6169).

### Alternative 4: Live Hot-Plug into Running Containers for Alpha
Support live in-place hot-mounting into running containers without restart in
Alpha via mount propagation.
* *Why rejected for Alpha*: While technically feasible, it adds operational
  complexity (requiring parent mount setup and specific directory conventions).
  Deferring to Beta allows the core API and VolumeManager lifecycle to stabilize
  first.

[kep-1287]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1287-in-place-update-pod-resources
[kep-1790]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/1790-recover-resize-failure
[kep-6030]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/6030-dynamic-resize-of-memory-backed-volumes
