# KEP-2172: Warn about implicitly insecure running containers

> **Note:** This proposal is based on the original draft by Tim Hockin
> ([thockin/k8s-enhancements#2168](https://github.com/thockin/k8s-enhancements/pull/2168)).
> It has been updated here to match the latest KEP template and current
> proposed implementation details.

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1 (Optional)](#story-1-optional)
    - [Story 2 (Optional)](#story-2-optional)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Condition when running implicitly-root](#condition-when-running-implicitly-root)
  - [Events when running implicitly-root](#events-when-running-implicitly-root)
  - [kubectl](#kubectl)
    - [Visibility in <code>kubectl describe pod</code>](#visibility-in-kubectl-describe-pod)
    - [Color](#color)
  - [Metrics](#metrics)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubernetes does not make it hard enough to do the wrong thing.  Specifically,
there are things that users do ALL THE TIME that they really should not, such
as running containers as root.  This KEP aims to make it more obvious that they
are doing that, and to encourage them to declare their intent better.

## Motivation

Running containers as root, when they don't need to, has repeatedly made
container-escape CVEs worse: root inside the container is often also root on
the host, so any escape or misconfiguration hands an attacker far more than it
would for a non-root process. Examples include
[CVE-2019-5736](https://nvd.nist.gov/vuln/detail/CVE-2019-5736) (host runc
binary overwritten from a container) and, more recently,
[CVE-2024-21626](https://github.com/opencontainers/runc/security/advisories/GHSA-xr7r-f8xq-vfvv)
(a leaked file descriptor let a container process access, and as root
overwrite, the host filesystem). It's just not obvious when a pod is running
this way, which is why many people don't realize they are doing it.

Many container images do not use a non-root UID, because the default is root.
Then users run these containers without specifying `runAsUser`, and Kubernetes
happily runs the container as root, whether it needs root or not.

The following terms are defined for clarity:

"implicitly-root": containers which run with UID 0 without `runAsUser` set
to 0, or with GID 0 (as the primary GID or as a supplemental group)
without `runAsGroup`, `fsGroup`, or `supplementalGroups` set to 0.

"explicitly-root": containers which run with UID 0 with `runAsUser` set
to 0, or with GID 0 (as the primary GID or as a supplemental group) with
`runAsGroup`, `fsGroup`, or `supplementalGroups` set to 0.

### Goals

1) To make it more obvious when a pod has implicitly-root containers.
2) To subtly suggest to users that running as root is a bad idea.

### Non-Goals

1) To make it impossible or opt-in to run as root.
2) To actively impact users who run as root.
3) To make noise about explicitly-root containers (though the proposed
   `kubelet_insecure_pods{declaration="explicit"}` metric below does track
   them).

## Proposal

This proposal includes several parts. It depends on KEP-3619 (Fine-grained
SupplementalGroups control), which adds the running UID and GID to Pods'
status. KEP-3619 is already stable (as of v1.35): the effective UID/GID and
resolved supplemental groups are reported in
`status.containerStatuses[].user.linux.{uid,gid,supplementalGroups}`, so no
new API field is needed here; the parts below just read those existing
fields.

As of now, only Linux containers are covered, matching the scope of KEP-3619:
the effective UID/GID field this depends on is only populated for Linux
containers today. Windows support may be revisited later if that field gains
Windows support.

All container types are covered: regular containers, init containers, and
ephemeral (debug) containers.

### User Stories (Optional)

#### Story 1 (Optional)

Catie the cluster admin can run the following command and quickly see any
pods that are implicitly-root:

`kubectl get pods -A -o custom-columns=NAME:.metadata.name,INSECURE_UID:.status.conditions[?(@.type=="InsecureUserID")].status,INSECURE_GID:.status.conditions[?(@.type=="InsecureGroupID")].status`

#### Story 2 (Optional)

Pete the platform admin can track the `kubelet_insecure_pods` metric and
set alerts when it becomes non-zero. They can investigate and ask users to
set a specific `runAs...` or to set it to 0. Pete can also install
admission controllers to only allow approved users to set the `runAs...`
fields to 0.

#### Story 3

Usain the user will see `Warning` events with reasons
`ImplicitlyInsecureUserID`, `ImplicitlyInsecureGroupID`, or
`ImplicitlyInsecureUserAndGroupID` when they run `kubectl describe pod`, and
will eventually choose to make them go away by running as non-root.

### Notes/Constraints/Caveats (Optional)

This feature is purely observational. It adds pod conditions, warning
events, and a metric, and never blocks, denies, or changes how the pod is
admitted or run. If a container explicitly sets `runAsUser`, the UID
condition and event are skipped for it, no matter what value is set,
including 0. Likewise, if a container explicitly sets `runAsGroup`, the GID
condition and event are skipped for it, no matter what value is set,
including 0. The same applies if GID 0 is requested explicitly via
`fsGroup` or `supplementalGroups`.

### Risks and Mitigations

Realistically, many users will simply set `runAs...` to 0. This is still
considered a win, since it means they thought about it and are being explicit
about it.

Warning events could add up to a lot of noise across a cluster with many
implicitly-root pods. To bound this, events will be throttled to at most
1 event/pod/node/hour. The frequency can be reduced further if needed.

> Note: this throttle limits repeat events for the *same* pod, but each
> new pod gets its own fresh hour budget. So a workload that creates pods
> repeatedly (for example, a ReplicaSet recreating replicas) still gets
> one event per new pod, not one per hour overall. But this is no worse
> than the existing lifecycle events (Scheduled, Pulled, Created,
> Started) that kubelet already emits per pod, so it doesn't add
> proportionally more noise.

## Design Details

### Condition when running implicitly-root

Kubelet sets two pod conditions on every pod, `InsecureUserID` and
`InsecureGroupID`:

- `True`, if a container is observed running as UID/GID 0 without
  `runAsUser`/`runAsGroup` respectively.
- `False`, once the runtime has reported the container's UID/GID and it is
  confirmed not to be implicitly-root.
- `Unknown`, until the runtime reports `status.containerStatuses[].user.linux`
  for the container (e.g., before the container has started, or on a
  runtime too old to report it). `Unknown` self-resolves to `True`/`False`
  once the data is reported; no warning event fires while a condition is
  `Unknown`.

```go
const (
	// InsecureUserID indicates that one or more containers of the pod are running as
	// UID 0 without the pod or container spec explicitly requesting it via runAsUser.
	InsecureUserID PodConditionType = "InsecureUserID"
	// InsecureGroupID indicates that one or more containers of the pod are running as
	// GID 0 without the pod or container spec explicitly requesting it via runAsGroup.
	InsecureGroupID PodConditionType = "InsecureGroupID"
)

```

When `True`, the condition's message names the affected container(s). For example,
on a pod implicitly-root on both UID and GID:

```yaml
status:
  conditions:
  - type: InsecureUserID
    status: "True"
    reason: ImplicitlyInsecureUserID
    message: 'container(s) [c] running as UID 0 without runAsUser set'
  - type: InsecureGroupID
    status: "True"
    reason: ImplicitlyInsecureGroupID
    message: 'container(s) [c] running as GID 0 without runAsGroup set'
```

Users can also inspect `Pod.status.containerStatuses[].user.linux` for the
raw observed UID/GID.

These conditions will be bypassed if the user explicitly sets `runAsUser` or
`runAsGroup` in their pod.

`InsecureGroupID` also fires if a container's supplemental groups
(`status.containerStatuses[].user.linux.supplementalGroups`) include GID 0.
This can happen implicitly, since `supplementalGroupsPolicy: Merge` (the
[KEP-3619 default][supplemental-groups-policy-merge-default] when the
field is unset) merges group memberships from the image's `/etc/group`
into the supplemental groups list.

To avoid false positives, this check:

- Skips GID 0 requested explicitly via `fsGroup` or `supplementalGroups`,
  since both feed into the same resolved supplemental groups list
  regardless of `supplementalGroupsPolicy`.
- Skips a container's own primary GID, which the CRI runtime always
  copies into the reported supplemental groups list too, regardless of
  `supplementalGroupsPolicy` (see [containerd][containerd-gid-mirror] and
  [cri-o][cri-o-gid-mirror]).

  For example, a container with `runAsGroup: 0` reports
  `{"gid":0,"supplementalGroups":[0]}`. The `0` in `supplementalGroups`
  here is just the mirrored primary GID, not a separate finding, so this
  check skips it. The primary-GID check above decides the outcome
  instead: it bypasses this case since `runAsGroup: 0` is explicit, or
  reports it if GID 0 was implicit. Either way, the container is never
  reported twice.

[supplemental-groups-policy-merge-default]: https://github.com/kubernetes/kubernetes/blob/693b7b3db83afdf21f30d0037b528f86ee503b1f/staging/src/k8s.io/cri-api/pkg/apis/runtime/v1/api.pb.go#L233-L243
[containerd-gid-mirror]: https://github.com/containerd/containerd/blob/a8fc3a017297f9ac4a28b115f9b706a90f497851/pkg/oci/spec_opts.go#L134-L141
[cri-o-gid-mirror]: https://github.com/cri-o/cri-o/blob/efbce04159ead73850f34c289f333126ae9b7b88/server/container_create.go#L366-L367

Per KEP-127 (User Namespaces), pods with `spec.hostUsers: false` map
container UID/GID 0 to an unprivileged host UID/GID, so these conditions are
not evaluated for such pods.

### Events when running implicitly-root

Whenever kubelet sees an implicitly-root container, it will create a kubernetes
Event object warning the user. These events are throttled to at most
1 event/pod/node/hour unless kubelet restarts (also see
[Risks and Mitigations](#risks-and-mitigations) for the rapid pod creation
case).

Also, these events will be bypassed if the user explicitly sets `runAsUser` or
`runAsGroup` in their pod.

A pod that is implicitly-root on both UID and GID gets a single combined
event, not two, to avoid doubling the noise. This gives three possible event
reasons:

```
TYPE      REASON                             OBJECT                   MESSAGE
Warning   ImplicitlyInsecureUserID           pod/uid-only-insecure    container(s) [c] running as UID 0 without runAsUser set
Warning   ImplicitlyInsecureGroupID          pod/gid-only-insecure    container(s) [c] running as GID 0 without runAsGroup set
Warning   ImplicitlyInsecureGroupID          pod/merge-insecure       container(s) [c] running with GID 0 merged into supplementalGroups from the image (supplementalGroupsPolicy: Merge)
Warning   ImplicitlyInsecureUserAndGroupID   pod/both-insecure        container(s) [c] running as UID 0 without runAsUser set; container(s) [c] running as GID 0 without runAsGroup set
```

`ImplicitlyInsecureGroupID` is used for both the primary-GID and the
supplemental-groups case described above; the event text indicates which
one applies.

### kubectl

#### Visibility in `kubectl describe pod`

`kubectl describe pod` will print every entry in `pod.status.conditions`
generically, so the `InsecureUserID`/`InsecureGroupID` conditions and their
events will show up there with no kubectl code changes needed. For example,
on a pod implicitly-root on GID only via `supplementalGroupsPolicy: Merge`
(primary GID non-zero, but GID 0 merged in from the image's `/etc/group`),
the output will look something like:

```
$ kubectl get pod merge-insecure -o jsonpath='{.status.containerStatuses[0].user}'
{"linux":{"gid":1000,"supplementalGroups":[0,1000],"uid":1000}}

$ kubectl describe pod merge-insecure
...
Conditions:
  Type                        Status
  PodReadyToStartContainers   True
  Initialized                 True
  Ready                       True
  ContainersReady             True
  PodScheduled                True
  InsecureUserID              False
  InsecureGroupID             True
...
Events:
  Type     Reason                     Age  From     Message
  ----     ------                     ---- ----     -------
  Warning  ImplicitlyInsecureGroupID  17s  kubelet  container(s) [c] running with GID 0 merged into supplementalGroups from the image (supplementalGroupsPolicy: Merge)
```

#### Color

If possible, kubectl will detect whether it is printing to a console or not,
and if so it will color pods that are running as root in red.  For example,
`kubectl get pods` would highlight problematic pods.

> Note: this is an open design question, as of now. kubectl's table output is
> column-aligned by a tabwriter that sizes columns from each cell's raw byte
> length, with no general support for treating ANSI color escapes as
> zero-width, so naively embedding color codes would visibly widen output
> for every row, not just flagged ones. Needs either a fix to the shared
> table printer's width calculation, or a different surface (e.g. a
> plain-text column instead of coloring existing text). This will be
> evaluated for feasibility and implemented as part of a separate KEP owned
> by SIG CLI, if possible.

### Metrics

Kubelet will add one gauge metric, `kubelet_insecure_pods`, labeled by
`declaration` (`implicit` or `explicit`) and `id_type` (`uid`, `gid`, or
`supplementalgroups`):

- `declaration="implicit"`: number of pods with an implicitly-root
  container. A pod insecure on both UID and GID counts in both series. A
  container's primary GID being 0 counts only in `gid`, never also in
  `supplementalgroups`, per the exclusion described in
  [Condition when running implicitly-root](#condition-when-running-implicitly-root).
- `declaration="explicit"`: number of pods with a container that
  explicitly requests UID/GID 0, via `runAsUser`, `runAsGroup`, `fsGroup`,
  or `supplementalGroups`, and is observed running as that ID.

> Note: A `0` reading for a node can mean either "no insecure pods" or "this
> node's runtime doesn't report the data needed to detect them." To tell
> these apart, check `Node.Status.Features.SupplementalGroupsPolicy`
> (from KEP-3619): it reflects the runtime's live, per-node capability to
> report `user.linux`, the same data this metric depends on.

### Test Plan

[x] I/we understand the owners of the involved components may require updates
to existing tests to make this code solid enough prior to committing the
changes necessary to implement this enhancement.

##### Prerequisite testing updates

None. This reuses the existing pod condition, event, and status-reporting
machinery in kubelet; no changes to existing tests are required first.

##### Unit tests

- `pkg/kubelet/status/generate_test.go`: covers the `InsecureUserID`/
  `InsecureGroupID` conditions for implicitly-root, explicitly-root
  (bypassed), non-root, not-yet-reported (`Unknown`), and `hostUsers:
  false` (bypassed) pods. This will also cover the
  `supplementalGroupsPolicy: Merge` case (implicit and explicit GID 0 via
  `fsGroup`/`supplementalGroups`, and the primary-GID mirroring
  exclusion).
- `pkg/kubelet/kubelet_pods_test.go`: covers the combined vs. separate
  UID/GID event reasons and messages, and the `kubelet_insecure_pods`
  gauge.

Coverage of the touched packages, before this enhancement's tests were added:

- `pkg/securitycontext`: `2026-09-10` - `67.6%`
- `pkg/kubelet/status`: `2026-09-10` - `91.8%`
- `pkg/kubelet`: `2026-09-10` - `77.3%`

##### Integration tests

Not applicable. This feature is entirely within kubelet (reading an
existing status field and setting conditions/events/metric); it does not
add or change any kube-apiserver or controller-manager behavior, so there is
nothing for the integration test suite (which exercises apiserver +
controllers) to cover beyond what unit and node e2e tests already do.

##### e2e tests

`test/e2e_node/pod_conditions_test.go` covers this feature, with the feature
gate toggled on via `tempSetCurrentKubeletConfig`. Covered cases:

- a pod implicitly-root on both UID and GID (conditions, combined event,
  metric)
- a pod implicitly-root on UID only
- a pod implicitly-root on GID only
- a pod explicitly-root (bypassed)
- a non-root pod
- a `hostUsers: false` pod (skipped on runtimes without user namespace
  support)

### Graduation Criteria

#### Alpha

- Feature implemented behind the `InsecurePodWarnings` feature gate
- Initial unit tests completed
- e2e tests completed

#### Beta

- Gather feedback from users and providers
- e2e tests are in Testgrid and linked in KEP
- SIG-Scalability confirms the Events are a non-issue

#### GA

<!--
TBD
-->

#### Deprecation

N/A. This feature does not deprecate or replace any existing flag, field, or
behavior.

### Upgrade / Downgrade Strategy

A kubelet from a version without this feature fails to start if
`InsecurePodWarnings` is still set in its config, so the gate must be removed
before downgrading to such a version; the old kubelet then runs exactly as
before. Upgrading and (re-)enabling the gate needs no special steps beyond a
normal kubelet restart.

### Version Skew Strategy

This feature is entirely local to the kubelet; it does not coordinate with
the control plane or other nodes. A node running a kubelet without this
feature, or with the gate off, simply does not report the new conditions,
events, or metric for its own pods; other nodes are unaffected.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness/README.md.

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

_This section must be completed when targeting alpha to a release._

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `InsecurePodWarnings`
  - Components depending on the feature gate: `kubelet`
- [ ] Other

###### Does enabling the feature change any default behavior?

No behavioral change (this feature is purely observational). Enabling it
adds new pod conditions, one of three warning event reasons, and a new
metric; nothing about how a pod runs changes.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling stops new conditions/events/metric from being generated; it
does not affect running workloads. Already-set conditions on existing pod
objects are left as-is until the pod is otherwise resynced/recreated.

###### What happens if we reenable the feature if it was previously rolled back?

Verified on a kind cluster. Kubelet feature gates are only read at process
startup, so disabling or enabling the gate has no effect until kubelet
restarts. Editing the config file alone while kubelet keeps running was
confirmed to have no effect. After restarting kubelet with the gate
re-enabled, conditions, events, and the metric resume being generated correctly
on the next sync of each pod; nothing needs to be reconciled or backfilled.

###### Are there any tests for feature enablement/disablement?

Yes. Tests added in `test/e2e_node/pod_conditions_test.go` use
`tempSetCurrentKubeletConfig` to toggle the `InsecurePodWarnings` feature gate
on for their test context, and cover the pod conditions, event, and metric
with the gate enabled. Unit tests added in `pkg/kubelet/status`,
`pkg/kubelet`, and `pkg/securitycontext` cover the underlying logic with and
without the feature.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta to a release._

###### How can a rollout or rollback fail? Can it impact already running workloads?

It cannot fail in a way that affects running workloads: this feature only
toggles kubelet-computed conditions/events/metric and reads an existing
field (`status.containerStatuses[].user.linux`, see Proposal) rather than
adding one.

###### What specific metrics should inform a rollback?

There is no metric that would indicate this feature itself is misbehaving in
a way that harms workloads, since it changes no runtime behavior. The one
thing worth watching after enabling it is kubelet's own CPU/memory usage, in
case computing the new conditions/events/metric on every pod sync adds
noticeable overhead on nodes with very large numbers of pods.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

See Upgrade / Downgrade Strategy above. Disabling and re-enabling the gate on
the same kubelet version (each requiring a restart) was verified directly on
a kind cluster: conditions, events, and the metric correctly resume.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

_This section must be completed when targeting beta to a release._

###### How can an operator determine if the feature is in use by workloads?

The `kubelet_insecure_pods` metric (labeled by `declaration` and `id_type`)
becomes non-zero on any node running implicitly-root or explicitly-root
pods.

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: `ImplicitlyInsecureUserID`, `ImplicitlyInsecureGroupID`,
    `ImplicitlyInsecureUserAndGroupID`
- [x] API .status
  - Condition name: `InsecureUserID`, `InsecureGroupID`
- [ ] Other

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A. This feature does not serve requests and has no latency or error-rate
path of its own; it only adds conditions, events, and a metric as a side
effect of the pod sync kubelet already performs.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `kubelet_insecure_pods`
  - Components exposing the metric: `kubelet`
- [ ] Other

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No, the `kubelet_insecure_pods` metric described above already covers this.

### Dependencies

_This section must be completed when targeting beta to a release._

###### Does this feature depend on any specific services running in the cluster?

Yes, on the CRI runtime reporting `status.containerStatuses[].user.linux`
(see Proposal for the KEP-3619 background). No other services are required.

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

###### Will enabling / using this feature result in any new API calls?

No new API calls. Conditions/events are set as part of the existing kubelet
pod status sync path, which already calls the API server. Metrics involve no
API calls; they are only scraped from kubelet's `/metrics` endpoint.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes: two new `PodCondition` entries (`InsecureUserID`, `InsecureGroupID`) on
implicitly-root pods, and `Event` objects (throttled to 1 event/pod/hour),
both small and bounded per pod.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No; the added checks are simple field comparisons done once per pod sync.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No; the work is O(number of containers in a pod) per sync and negligible.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No new PIDs, sockets, or file descriptors are created.

### Troubleshooting

<!--
The Troubleshooting section currently serves the `Playbook` role. We may
consider splitting it into a dedicated `Playbook` document (potentially with
some monitoring details). For now, we leave it here.
-->

_This section must be completed when targeting beta to a release._

###### How does this feature react if the API server and/or etcd is unavailable?

The pod conditions/events/metric update is skipped or retried along with
the rest of the kubelet's normal pod status sync; no special handling is
added.

###### What are other known failure modes?

- Older CRI runtime does not report `status.containerStatuses[].user.linux`
  (predates stable KEP-3619 support). Observed directly on containerd
  v1.7.33: kubelet ran without any error, but the `InsecureUserID`/
  `InsecureGroupID` conditions stayed `Unknown` (not `False`) for affected
  containers, correctly signaling that detection wasn't possible rather
  than implying they were secure. Upgrading to containerd v2.0+ (which
  populates that CRI field, per KEP-3619) resolved the conditions to their
  correct `True`/`False` value.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

* 2020-12-01: First draft
* 2026-09-10: KEP re-picked up and updated for Alpha in v1.38; implementation
  started now that the depended-upon KEP-3619 (Fine-grained
  SupplementalGroups control) is stable

## Drawbacks

* This will annoy some users.
* This does not solve the root problems.

## Alternatives

Changing defaults and making users opt-in to running as root was considered.
This is a breaking change and was discarded.

Injecting artificial slowdowns on insecure container startup was considered,
but this is user-hostile and was discarded.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

N/A
