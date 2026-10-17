# KEP-6432: Configurable Container Level field for Pod Readiness

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Design](#api-design)
  - [kubectl Ready column](#kubectl-ready-column)
  - [Feature Gate Behavior](#feature-gate-behavior)
  - [Test Plan](#test-plan)
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
- [Infrastructure Needed](#infrastructure-needed)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

Kubelet marks a Pod `Ready` only when every regular container and every restartable init container is ready,([ref](https://github.com/kubernetes/kubernetes/blob/1d54bff3606e44a9d5178698816fc8af340e85a9/pkg/kubelet/status/generate.go#L127)).Pod readiness determines whether a Pod is considered ready to receive traffic through a Service, unless `publishNotReadyAddresses` is set to true. At present, there is no way to configure or change this behavior.

This KEP adds an optional `podReadinessPolicy` field on the container spec. Value `Required` keeps today's behavior and is the default when `podReadinessPolicy` is unset. Value `Ignored` tells kubelet to leave that container out while evaluating pod readiness. The container still reports its own ready status and the same will be reflected in `pod.status.containerStatuses`.
Pod readiness will still be reported through the `Ready` condition (`pod.status.conditions[] | select(.type == "Ready")`), retaining same behaviour. If every sidecar and regular container has `podReadinessPolicy` set to `Ignored`, the Pod will not be ready and an Event will be created.

The `podReadinessPolicy` field is allowed on regular containers and on native sidecars (init containers with `restartPolicy: Always`). It is rejected on init containers without `restartPolicy: Always`. The value can be changed on a running Pod without recreating the container.The `podReadinessPolicy` field is intentionally an enum rather than a bool, so later modes can be added if needed for cases such as dynamic Pods.

Controllers that already depend on pod ready condition will not need any changes like Deployment, EndpointSlice controllers etc. They pick up the condition kubelet writes and do not gain new logic for `podReadinessPolicy`.

CLI client like kubectl which displays container readiness will now report both ready containers and pod readiness. For example `kubectl get po` keeps the container ready count in the READY column and adds the Pod `Ready`/`NotReady` condition in parentheses, for example `1/2 (Ready)`. This follows the same pattern as the RESTARTS column, which prints extra detail beside the count.



## Motivation

The EndpointSlice controller publishes a Pod when the Pod is ready. One unready sidecar is enough to remove the application from a Service. There are some cases in which this behavior is appropriate and others where it is undesirable.

Cases where this behavior is appropriate, and the Pod should not be ready when the sidecar container is not ready:

- If a  service mesh/ proxy container is not ready, respective pods should be removed from EndpointSlices.

Cases where this behavior is undesirable, and Pod readiness should not be affected by an unready sidecar container:

- A log collector container
- A monitoring agent
- A backup container

Both scenarios are possible in practice, and choosing one default satisfies only one set of requirements and breaks the other. The practical solution is to make this configurable by the user.The only realistic way to achieve this is to add a field where users specify how container readiness affects overall Pod readiness. The user running the application knows which value is appropriate and sets it.

### Goals

- Let a regular container or a native sidecar opt out of Pod readiness.
- Preserve today's behavior for every Pod that does not set `podReadinessPolicy`.
- Preserve today's behavior for container status reflective of container readiness.
- Allow `podReadinessPolicy` to change in place, with no container recreation needed.
- Use an enum for `podReadinessPolicy` so later modes can be added without a new field or an API change.
- Exclude Ephemeral containers for Pod readiness, matching current behavior.
- Leave every controller that already consumes the Pod `Ready` condition unchanged.
- Show the Pod `Ready` condition in the `kubectl get po` READY column, beside the existing container ready count.

### Non-Goals

- Changing Pod status fields.
- Introducing a new field such as `sidecarContainer`.

## Proposal

Add an optional `podReadinessPolicy` enum to `core/v1` `Container` in `PodSpec`, gated by `ContainerPodReadinessPolicy`.

```go
// +enum
type PodReadinessPolicy string

const (
    // Required. Include this container's readiness when evaluating overall Pod readiness.
    PodReadinessPolicyRequired PodReadinessPolicy = "Required"

    // Ignored. Do not include this container's readiness when evaluating overall Pod readiness.
    PodReadinessPolicyIgnored PodReadinessPolicy = "Ignored"
)
```

### User Stories

#### Story 1

As a platform engineer ,running an application container alongside a log collector as a native sidecar, I want the application to continue serving traffic even when the log collector container is not running or is not ready.

#### Story 2

As a database administrator, running a database Pod that contains a database container and backup containers, I want the database to continue serving traffic even when a backup container is not running or is not ready.

#### Story 3

As an engineer running a metrics agent alongside an application container to collect metrics for observability, I want the application to continue serving traffic even when the metrics container is not running or is not ready.

### Notes/Constraints/Caveats

- When the user sets `podReadinessPolicy` to `Ignored` on every sidecar and regular container, the Pod `Ready` condition is `False`.
- `podReadinessPolicy` does not apply to ephemeral containers.

### Risks and Mitigations

- Setting `podReadinessPolicy` to `Ignored` on a service mesh/ proxy side car container keeps the Pod in EndpointSlices even when the proxy cannot serve traffic.
- An in-place change of `podReadinessPolicy` from `Ignored` to `Required` can remove a Pod from EndpointSlices if that container is not ready. This has the same effect as a readiness probe failure.
- If a user sets `podReadinessPolicy` to `Ignored` on every container, the Pod will never receive traffic.

## Design Details

### API Design

The `podReadinessPolicy` field is added to [Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#container-v1-core). The container array is used in both `containers` and `initContainers`. Validation must allow `podReadinessPolicy` only on `initContainers` that have `restartPolicy: Always`. `podReadinessPolicy` is not added to `EphemeralContainer` in `core/v1`, so no extra validation is needed to exclude ephemeral containers.

**API Changes:**

```go
// +enum
type PodReadinessPolicy string

const (
    // Required. Include this container's readiness when evaluating overall Pod readiness.
    PodReadinessPolicyRequired PodReadinessPolicy = "Required"

    // Ignored. Do not include this container's readiness when evaluating overall Pod readiness.
    PodReadinessPolicyIgnored PodReadinessPolicy = "Ignored"
)

type Container struct {
    // +optional
    PodReadinessPolicy PodReadinessPolicy
}
```
The same additions are made in the external API versions.

**Kubelet Changes:**

The kubelet [status generator](https://github.com/kubernetes/kubernetes/blob/1d54bff3606e44a9d5178698816fc8af340e85a9/pkg/kubelet/status/generate.go) will follow logic similar to the following:

1. Loop for every sidecar container and regular container:
2. If the container has `podReadinessPolicy` is `Ignored`, exclude it from Pod readiness calculation
3. After the loop, if all the side car and regular containers have `podReadinessPolicy` set to `Ignored`, set the Pod `Ready` condition to `False`.

**Kubectl changes:**

Output formatter of kubectl will change to print pod readiness along with `readyContainers/totalContainers`. This is similar to the `RESTARTS` field.
Change will be applicable only for the commands which displays pod readiness, rest of the behaviour is preserved.

For example: 
Output without new feature
```text
$ kubectl get po
NAME       READY    STATUS    RESTARTS   AGE
web-abc    2/2      Running   0          1m
web-def    1/2      Running   0          1m
web-ghi    1/2      Running   0          1m
```

Output with new feature
```text
$ kubectl get po
NAME       READY           STATUS    RESTARTS   AGE
web-abc    2/2 (Ready)     Running   0          1m
web-def    1/2 (Ready)     Running   0          1m
web-ghi    1/2 (NotReady)  Running   0          1m
```

Other custom Kubernetes clients may or may not adopt this output formatting, depending on user preference.

### Feature Gate Behavior

When the `ContainerPodReadinessPolicy` feature gate is **disabled**:

- Existing behavior continues, and kubelet does not consider `podReadinessPolicy` when evaluating Pod readiness.

When the `ContainerPodReadinessPolicy` feature gate is **enabled**:

- Kubelet reads `podReadinessPolicy` when it evaluates Pod readiness.

### Test Plan

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Unit tests

When the feature gate is enabled:

- For a Pod whose regular containers and native sidecars all have `podReadinessPolicy` set to `Ignored`, verify that the Pod `Ready` condition is `False`.
- For a Pod with one container that is always ready (no readiness probe, or a probe set to `/bin/true`) and another container with `podReadinessPolicy` set to `Ignored`, verify that the Pod `Ready` condition is `True`.
- For a Pod with a container that has `podReadinessPolicy` set to `Required`, verify that the current behavior is preserved.
- For a regular container or native sidecar with `podReadinessPolicy` set to `Ignored`, kill the container and verify that Pod readiness is not affected.
- For the client changes, verify that the READY column displays the Pod status `Ready`/`NotReady`.

Existing tests cover the same cases with the feature gate disabled.

##### Integration tests

- Validate that a Deployment or DaemonSet rolling update preserves the existing behaviour with `podReadinessPolicy` set to `Ignored`.


##### e2e tests

- Create a Pod with a container that has `podReadinessPolicy` set while the feature gate is disabled, and verify that `podReadinessPolicy` is dropped.
- Create a Pod with a container that leaves `podReadinessPolicy` unset while the feature gate is enabled, and verify that the Pod is created.
- Create a Pod with an init container that does not set `restartPolicy: Always` and sets `podReadinessPolicy`, and verify that Pod creation fails validation.
- Patch `podReadinessPolicy` on a running Pod while the feature gate is enabled, and verify that the container is not restarted and the Pod remains `Running`.
- Verify that a Service routes traffic / doesn't route traffic appropriately when pod is `Ready` / `NotReady`.

### Graduation Criteria

#### Alpha

- The `podReadinessPolicy` field is implemented,functional and documented.
- Required tests are passing.
- Details of any future enhancements are captured.

#### Beta

- No major bugs were reported during Alpha.
- Feedback has been gathered from users.

#### GA

- The feature has been in Beta, and stable, for at least two releases.

### Upgrade / Downgrade Strategy

After upgrading to a version that supports this KEP, the `ContainerPodReadinessPolicy` feature gate can be enabled at any time.

On downgrade:

- kube-apiserver drops `podReadinessPolicy` from new writes and keeps it on objects that already have it.
- kubelet ignores the stored `podReadinessPolicy` value, because the feature gate is not present, and includes every container in Pod readiness.
- Pods that relied on `Ignored` return to today's behavior and may become unready.

### Version Skew Strategy

The feature requires the feature gate on both kube-apiserver and kubelet.

- **kube-apiserver newer than kubelet.** The newer kube-apiserver stores `podReadinessPolicy`. The older kubelet drops the unknown field and includes every container in Pod readiness. `podReadinessPolicy` has no effect until kubelet is upgraded.
- **kubelet newer than kube-apiserver.** The older kube-apiserver drops `podReadinessPolicy`, so kubelet will include all the containers for Pod readiness.
- **kubectl older than the apiserver** Output which displays READY column of the pod, just displays the `readyContainers/totalContainers` and this might not reflect the actual pod readiness. There is no functional impact.
- **kubectl newer than the apiserver** Output which displays READY column of the pod will show Pod readiness

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ContainerPodReadinessPolicy`
  - Components depending on the feature gate:
    - `kube-apiserver` (validation and field dropping)
    - `kubelet` (readiness aggregation)

###### Does enabling the feature change any default behavior?

No. An unset `podReadinessPolicy` includes the container in Pod readiness, which is today's behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### Recommended rollback procedure for Pods with `podReadinessPolicy` set to `Ignored`

- Where feasible, update podReadinessPolicy from Ignored to Required. Plan ahead of the rollback, as Pods may become unready.

###### What happens if we reenable the feature if it was previously rolled back?

Pods that are not ready may become ready if their containers have `podReadinessPolicy` set to `Ignored`. Rest of the behaviour remains the same.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Both kubelet and kube-apiserver need a version that supports this feature. Otherwise the rollout will not produce the expected results. A failed rollout does not affect running workloads.

No foreseeable rollback failures. If rollback fails, running workloads are unaffected.If rollback fails and kubelet and kube-apiserver are skewed, so that one component supports the feature gate and the other does not, Pods whose containers have `podReadinessPolicy` set to `Ignored` can become unready.

###### What specific metrics should inform a rollback?

Not Applicable.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?



###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Check `pod.spec.containers[].podReadinessPolicy`.

###### How can someone using this feature know that it is working for their instance?

A container with `podReadinessPolicy` set to `Ignored` does not affect Pod readiness.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No changes to the current values.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Not Applicable.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

The object count does not change. The Pod object grows by one string, which is negligible.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

This feature adds no further impact. The usual consequences of kube-apiserver and/or etcd being unavailable still apply.

###### What are other known failure modes?

No foreseeable failure modes.

###### What steps should be taken if SLOs are not being met to determine the problem?

Investigate kubelet execution.

## Implementation History

- 2026-09-25: KEP drafted.

## Drawbacks

None.

## Alternatives

**Do not change anything, and document that sidecars affect readiness.** A large set of use cases, such as log collector containers and metrics containers, cannot run as intended.

**Change the default so native sidecars do not affect readiness.** This breaks service meshes and violates API compatibility.


## Infrastructure Needed

None.
