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
# KEP-5304: DRA Device Attributes Downward API

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
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes a Downward API for Dynamic Resource Allocation (DRA) device attributes, implemented entirely in the kubelet. Workloads like KubeVirt need device identifiers (e.g., PCIe bus address for physical GPUs, mediated device UUID for virtual GPUs) to configure device access inside guests. While these identifiers exist in DRA objects (`ResourceClaim`, `ResourceSlice`), they are not currently consumable via the Pod's Downward API. This proposal adds a new Downward API selector (`draDeviceFieldRef`) for environment variables and projected volumes. The kubelet will run a local DRA attributes controller that watches `ResourceClaim` and `ResourceSlice` objects, caches resolved attributes per Pod resource claim/request, and resolves `draDeviceFieldRef` on demand.

## Motivation

Workloads that need to interact with DRA-allocated devices (like KubeVirt virtual machines) require access to device-specific identifiers such as PCIe bus addresses or mediated device UUIDs. These attributes are already stored in DRA `ResourceClaim` and `ResourceSlice` objects, but there is currently no standardized way for Pods to consume them. Users must resort to custom controllers that watch these objects and inject attributes via annotations or custom resources, which is fragile and non-portable.

The Kubernetes Downward API provides a standard mechanism for exposing Pod and container metadata to workloads. Extending this API to support DRA device attributes would enable workloads to discover device information without requiring additional custom controllers or privileged access to the Kubernetes API.

### Goals

- Provide a stable Downward API path for device attributes associated with `pod.spec.resourceClaims[*]` requests
- Support any attributes defined in ResourceSlice objects, controlled by the allowlist mechanism
- Allow resource owners to explicitly allowlist which attributes are exposed (maximum 8 per request) for security and control
- Maintain compatibility with existing DRA drivers without requiring changes to driver interfaces

### Non-Goals

- Expose the entirety of `ResourceClaim`/`ResourceSlice` objects in the Downward API
- Allow arbitrary JSONPath into external objects via Downward API
- Change or extend DRA driver interfaces
- Support dynamic updates to device attributes after Pod container startup (for env vars)

## Proposal

This proposal introduces a new Downward API selector (`draDeviceFieldRef`) that allows Pods to reference DRA device attributes in both environment variables and projected volumes. The kubelet will implement a local DRA attributes controller that:

1. Watches Pods scheduled to the node and identifies those with `pod.spec.resourceClaims`
2. Watches `ResourceClaim` objects in the Pod's namespace to retrieve allocation information
3. Watches `ResourceSlice` objects for the node and driver to resolve device attributes
4. Maintains a per-Pod cache of `(claimName, requestName) -> {attribute: value}` mappings
5. Resolves `draDeviceFieldRef` references when containers start (for env vars) or continuously updates projected volumes

To limit and control the amount of data surfaced, users must specify a per-request allowlist (maximum 8 attributes) in `ResourceClaim.spec.devices.requests[].downwardAPIAttributes` to explicitly control which attributes may be exposed via the Downward API.

### User Stories (Optional)

#### Story 1

As a KubeVirt developer, I want the virt-launcher Pod to automatically discover the PCIe root address of an allocated physical GPU via environment variables, so that it can construct the libvirt domain XML to pass through the device to the virtual machine guest without requiring a custom controller.

#### Story 2

As a DRA driver author, I want my driver to remain unchanged while allowing applications to consume device attributes (like `resource.kubernetes.io/pcieRoot` or `dra.kubervirt.io/mdevUUID`) through the native Kubernetes Downward API, with control over which attributes are exposed via the allowlist mechanism

### Notes/Constraints/Caveats (Optional)

- Environment variables are set at container start time: Once a container starts, its environment variables are immutable. If device attributes change after container start, env vars will not reflect the change.
- Projected volumes support updates: Files in projected volumes will be updated atomically when the kubelet's cache detects changes to the underlying ResourceClaim or ResourceSlice.
- Allowlist enforcement: The kubelet MUST enforce the per-request allowlist. If an attribute is not in the allowlist, attempts to reference it via `draDeviceFieldRef` will fail.
- Attribute names: Any attribute name that exists in the ResourceSlice can be referenced, as long as it is allowlisted. The API server does not validate attribute names against a hardcoded list, allowing vendors to define their own attributes (e.g., `dra.kubervirt.io/mdevUUID`).

### Risks and Mitigations

Risk: Exposing device attributes might leak sensitive information.
Mitigation: Attributes are limited to device identifiers needed for legitimate device configuration. The per-request allowlist (max 8) gives resource owners explicit control. The NodeAuthorizer ensures kubelet only accesses ResourceClaims and ResourceSlices for Pods scheduled to that node.

Risk: Kubelet performance impact from watching ResourceClaim and ResourceSlice objects.
Mitigation: Use shared informers with proper indexing to minimize API server load. Scope watches to node-local resources only. Monitor cache memory usage and set reasonable limits.

Risk: API surface expansion could make the Downward API overly complex.
Mitigation: Keep the API minimal and type-safe. Use a closed set of supported attributes rather than arbitrary JSONPath. Require feature gate enablement in alpha to gather feedback before expanding.

Risk: Compatibility with future DRA changes.
Mitigation: Implementation is decoupled from DRA driver interfaces. Changes to DRA object structure are handled in the kubelet controller. Attribute names are standardized and versioned.

## Design Details

### API Changes

#### New Downward API Selector: DRADeviceFieldRef

A new typed selector is introduced in `core/v1` that can be used in both environment variable sources and projected downward API volume items:

```go
// DRADeviceFieldRef selects a DRA-resolved device attribute for a given claim+request.
// +featureGate=DRADownwardDeviceAttributes
// +structType=atomic
type DRADeviceFieldRef struct {
    // ClaimName must match pod.spec.resourceClaims[].name.
    // +required
    ClaimName string `json:"claimName"`
    
    // RequestName must match the corresponding ResourceClaim.spec.devices.requests[].name.
    // +required
    RequestName string `json:"requestName"`
    
    // Attribute specifies which device attribute to expose from the ResourceSlice.
    // The attribute name must be present in the ResourceSlice's device attributes
    // and must be allowlisted in the corresponding ResourceClaim request's downwardAPIAttributes field.
    // +required
    Attribute string `json:"attribute"`
}

// In core/v1 EnvVarSource:
type EnvVarSource struct {
    // ...existing fields...
    
    // DRADeviceFieldRef selects a DRA device attribute for a given claim+request.
    // +featureGate=DRADownwardDeviceAttributes
    // +optional
    DRADeviceFieldRef *DRADeviceFieldRef `json:"draDeviceFieldRef,omitempty"`
}

// In core/v1 DownwardAPIVolumeFile:
type DownwardAPIVolumeFile struct {
    // ...existing fields...
    
    // DRADeviceFieldRef selects a DRA device attribute for a given claim+request.
    // +featureGate=DRADownwardDeviceAttributes
    // +optional
    DRADeviceFieldRef *DRADeviceFieldRef `json:"draDeviceFieldRef,omitempty"`
}
```

Validation:
- Enforce exactly one of `fieldRef`, `resourceFieldRef`, or `draDeviceFieldRef` in both env and volume items
- Validate `claimName` and `requestName` against DNS label rules
- Validate `attribute` against the supported set (`resource.kubernetes.io/pcieRoot`, `dra.kubervirt.io/mdevUUID` in alpha)

#### Resource API: Per-Request Allowlist

Add an optional allowlist to `ResourceClaim.spec.devices.requests[]` to restrict which attributes may be exposed:

```go
// In resource.k8s.io DeviceRequest (v1alpha3+), behind DRADownwardDeviceAttributes gate
type DeviceRequest struct {
    // ...existing fields...
    
    // DownwardAPIAttributes allowlists which device attributes (e.g., "resource.kubernetes.io/pcieRoot")
    // may be exposed to pods via the Downward API for allocations that satisfy this request.
    // If unset or empty, no attributes are exposed for this request.
    // Maximum of 8 entries.
    // +optional
    // +listType=set
    // +listMaxItems=8
    DownwardAPIAttributes []string `json:"downwardAPIAttributes,omitempty"`
}
```

Validation:
- Enforce maximum length of 8
- Enforce membership in the supported attribute set (alpha: `resource.kubernetes.io/pcieRoot`, `dra.kubervirt.io/mdevUUID`)
- Reject duplicates (treat as a set)
- If the list is empty or omitted, no attributes are exposed for that request

### Kubelet Implementation

The kubelet runs a local DRA attributes controller that:

1. Watches Pods: Identifies Pods on the node with `pod.spec.resourceClaims` and tracks their `pod.status.resourceClaimStatuses` to discover generated ResourceClaim names
2. Watches ResourceClaims: For each relevant claim, reads `status.allocation.devices.results[*]` and maps entries by request name
3. Watches ResourceSlices: Resolves standardized attributes from `spec.devices[*].attributes` for the matching device name
4. Maintains Cache: Keeps a per-Pod map of `(claimName, requestName) -> {attribute: value}` with a readiness flag

Resolution Semantics:
- Cache entries are updated on claim/slice changes
- For container environment variables, resolution happens at container start using the latest ready values
- For projected volumes, kubelet updates files via AtomicWriter when cache changes
- Kubelet MUST honor the per-request allowlist; if `draDeviceFieldRef.attribute` is not in the allowlist, the value is not exposed
- If data is not ready when a container starts, resolution fails for required env vars or leaves optional ones unset

Security & RBAC:
- Node kubelet uses NodeAuthorizer to watch/read `ResourceClaim` and `ResourceSlice` objects related to Pods scheduled to the node
- Scope access to only necessary fields
- No cluster-wide escalation; all data flows through node-local caches

### Usage Examples

Environment Variable Example (Physical GPU Passthrough):

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: virt-launcher-gpu-passthrough
spec:
  resourceClaims:
  - name: pgpu-claim
    resourceClaimName: my-physical-gpu-claim
  containers:
  - name: compute
    image: virt-launcher:latest
    env:
    - name: PGPU_CLAIM_PCI_ROOT
      valueFrom:
        draDeviceFieldRef:
          claimName: pgpu-claim
          requestName: pgpu-request
          attribute: resource.kubernetes.io/pcieRoot
```

Projected Volume Example (Virtual GPU via Mediated Device):

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: virt-launcher-vgpu
spec:
  resourceClaims:
  - name: vgpu-claim
    resourceClaimName: my-vgpu-claim
  containers:
  - name: app
    image: virt-launcher:latest
    volumeMounts:
    - name: device-attrs
      mountPath: /etc/device-attrs
  volumes:
  - name: device-attrs
    projected:
      sources:
      - downwardAPI:
          items:
          - path: VGPU_CLAIM_MDEV_UUID
            draDeviceFieldRef:
              claimName: vgpu-claim
              requestName: vgpu-request
              attribute: dra.kubevirt.io/mdevUUID
```

### Feature Gate

- Name: `DRADownwardDeviceAttributes`
- Stage: Alpha (v1.35)
- Components: kube-apiserver, kubelet, kube-scheduler
- Enables: 
  - New `DRADeviceFieldRef` selector in env/volumes
  - DRA attributes controller in kubelet
  - Request-level allowlist `DeviceRequest.downwardAPIAttributes`

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No additional prerequisite testing updates are required. Existing DRA test infrastructure will be leveraged.

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

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

Integration tests will cover:

- Feature gate toggling: Verify API rejects `draDeviceFieldRef` when feature gate is disabled
- End-to-end resolution: Create Pod with resourceClaims, verify env vars and projected volume files contain correct attribute values
- Allowlist enforcement: Verify attributes not in allowlist are rejected
- Negative cases: Missing allocation, invalid claim/request names, unsupported attributes

Tests will be added to `test/integration/kubelet/` and `test/integration/dra/`.

##### e2e tests

E2E tests will validate real-world scenarios:

- KubeVirt-like workload: Pod with GPU claim consumes `resource.kubernetes.io/pcieRoot` via environment variable
- Multi-claim Pod: Pod with multiple resource claims, each with different attributes
- Projected volume updates: Verify volume files update when ResourceSlice changes
- Feature gate disabled: Verify graceful degradation when gate is off
- Node failure scenarios: Test behavior when kubelet restarts or node is drained

Tests will be added to `test/e2e/dra/` and `test/e2e_node/downwardapi_test.go`.

### Graduation Criteria

#### Alpha (v1.35)

- Feature implemented behind `DRADownwardDeviceAttributes` feature gate
- API types added: `DRADeviceFieldRef` in `core/v1.EnvVarSource` and `core/v1.DownwardAPIVolumeFile`
- Resource API extension: `DeviceRequest.downwardAPIAttributes` allowlist (max 8)
- Kubelet DRA attributes controller implemented
- Support for `resource.kubernetes.io/pcieRoot` and `dra.kubervirt.io/mdevUUID` attributes
- Unit tests for validation, cache, and resolution logic
- Initial integration and e2e tests completed
- Documentation published for API usage

#### Beta

TBD

#### GA

TBD

### Upgrade / Downgrade Strategy

**Upgrade:**
- When upgrading to a release with this feature, the feature gate is disabled by default in alpha
- Existing Pods without `draDeviceFieldRef` are unaffected
- To use the feature, users must:
  1. Enable the `DRADownwardDeviceAttributes` feature gate on kube-apiserver and kubelet
  2. Add `downwardAPIAttributes` allowlist to ResourceClaim request specs
  3. Update Pod specs to use `draDeviceFieldRef` in env vars or volumes
- No changes to existing DRA drivers are required

**Downgrade:**
- If downgrading from a version with this feature to one without it:
  - Pods using `draDeviceFieldRef` will fail API validation and cannot be created
  - Existing Pods with `draDeviceFieldRef` will continue to run but cannot be updated
  - Users should remove `draDeviceFieldRef` from Pod specs before downgrade
- Feature gate can be disabled to reject new Pods with `draDeviceFieldRef`
- Kubelet will ignore `draDeviceFieldRef` when feature gate is disabled

### Version Skew Strategy

**Control Plane and Node Coordination:**
- This feature primarily involves the kubelet and API server
- The API server validates `draDeviceFieldRef` (feature gate enabled)
- The kubelet resolves `draDeviceFieldRef` at runtime

**Version Skew Scenarios:**

1. **Older kubelet (n-1, n-2, n-3) with newer API server:**
   - API server accepts Pods with `draDeviceFieldRef` (if feature gate is enabled)
   - Older kubelet without the feature will fail to resolve `draDeviceFieldRef`
   - Pod container start will fail with an error about unknown field
   - **Mitigation**: Do not enable feature gate until all kubelets are upgraded

2. **Newer kubelet with older API server:**
   - Older API server rejects Pods with `draDeviceFieldRef` (unknown field)
   - This is safe - users cannot create invalid Pods
   - No special handling required

3. **Scheduler and Controller Manager:**
   - This feature does not require changes to scheduler or controller manager
   - No version skew concerns with these components

**Recommendation:**
- Enable feature gate cluster-wide (API server and all kubelets) at the same time
- Test in a non-production environment first
- Use rolling upgrade strategy for kubelets

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

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DRADownwardDeviceAttributes`
  - Components depending on the feature gate: kube-apiserver, kubelet
  - Enabling requires restarting kube-apiserver and kubelet with `--feature-gates=DRADownwardDeviceAttributes=true`
  - No downtime of control plane is required for enabling/disabling (rolling restart is sufficient)
  - Kubelet restart is required on each node

###### Does enabling the feature change any default behavior?

No. Enabling the feature gate only adds new optional API fields (`draDeviceFieldRef`) to `EnvVarSource` and `DownwardAPIVolumeFile`. Existing Pods and workloads are unaffected. Users must explicitly opt in by:
1. Adding `downwardAPIAttributes` allowlist to ResourceClaim specs
2. Using `draDeviceFieldRef` in Pod env vars or volumes

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. The feature can be disabled by setting the feature gate to `false` and restarting the API server and kubelets.

**Consequences:**
- New Pods using `draDeviceFieldRef` will be rejected by the API server
- Existing running Pods with `draDeviceFieldRef` will continue to run, but:
  - Environment variables set at container start remain unchanged
  - Projected volumes will stop updating (files remain with last-known values)
- Pods with `draDeviceFieldRef` cannot be updated while feature gate is disabled

**Recommendation:** Before disabling, ensure no critical workloads depend on `draDeviceFieldRef` or migrate them to alternative mechanisms (e.g., annotations).

###### What happens if we reenable the feature if it was previously rolled back?

Re-enabling the feature gate restores full functionality:
- API server accepts Pods with `draDeviceFieldRef`
- Kubelet resumes resolving attributes and updating projected volumes
- New Pods will work correctly
- Existing Pods (created while feature was disabled) are unaffected unless they already use `draDeviceFieldRef`

No data migration or special handling is required.

###### Are there any tests for feature enablement/disablement?

Yes:
- Unit tests will verify API validation behavior with feature gate on/off
- Integration tests will verify:
  - Pods with `draDeviceFieldRef` are rejected when feature gate is disabled
  - Pods with `draDeviceFieldRef` are accepted when feature gate is enabled
  - Kubelet correctly resolves attributes when feature gate is enabled
  - Kubelet ignores `draDeviceFieldRef` when feature gate is disabled

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

Yes. 

- WATCH ResourceClaim: Each kubelet will establish an informer based watch on ResourceClaim objects for Pods scheduled to its node
- WATCH ResourceSlice: Each kubelet will establish an informer based watch on ResourceSlice objects for devices on its node


###### Will enabling / using this feature result in introducing new API types?

No. This feature adds new fields to existing API types (`core/v1.EnvVarSource`, `core/v1.DownwardAPIVolumeFile`, and `resource.k8s.io/v1.DeviceRequest`) but does not introduce new API object types.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes
  - Pod, adds a field in pod
  - ResourceClaim, adds a list type field with max of 8

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes, but the impact should be minimal:

- Pod startup latency: Kubelet must resolve `draDeviceFieldRef` values before starting containers with environment variables, but the impact of this is minimized by local informer based lookup

- The feature does not affect existing SLIs/SLOs for clusters not using DRA or for Pods not using `draDeviceFieldRef`.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Yes, but the increase should be acceptable:

- Kubelet Memory:
  - In-memory cache for attribute mappings: ~1KB per Pod with DRA claims (assumes 5 attributes × 50 bytes per attribute × 4 requests)
  - Informer caches for ResourceClaim and ResourceSlice objects
  - Estimated total: 5-10MB for 110 pods per node (worst case: all pods use DRA)


- Kubelet CPU:
  - Watch processing: Minimal, only processes updates for node-local resources
  - Resolution: O(1) cache lookups, minimal CPU (<1% increase)
  
- API Server (RAM/CPU):
  - Additional watch connections: 2 per kubelet (ResourceClaim, ResourceSlice)
  - 5000-node cluster: 10,000 watch connections (~50-100MB RAM, negligible CPU)
  - Mitigation: Use informer field selectors to minimize data sent over watches

**Network IO:**
- Watch streams: Incremental updates only, minimal bandwidth
- Estimated: <10KB/s per kubelet under normal conditions

These increases are within acceptable limits for modern Kubernetes clusters.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No significant risk of resource exhaustion:

Sockets:
- Kubelet opens 2 watch connections to API server (ResourceClaim, ResourceSlice)

Memory:
- Pathological case: Malicious user creates many Pods with `draDeviceFieldRef` to exhaust kubelet memory
- Risk: Low. Cache size bounded by max pods per node (110) × cache entry size (~1KB) = ~110KB
- Mitigation: Existing pod limits prevent unbounded growth; cache eviction for terminated Pods

Inodes:
- Projected volumes create files on disk for each `draDeviceFieldRef` item
- Risk: Low. Limited by max volumes per pod (existing limits) and max pods per node
- Mitigation: Existing Kubernetes limits on volumes and pods

CPU:
- Pathological case: Rapid ResourceClaim/ResourceSlice updates trigger excessive cache updates
- Risk: Low. Updates are processed asynchronously; rate limited by API server watch semantics
- Mitigation: malicius updates can be prevented by AP&F at apiserver level

Performance tests will be added in beta to validate these assumptions under load (e.g., 110 pods/node, all using DRA).

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

- 2025-10-02: KEP created and initial proposal drafted
- 2025-10-03: KEP updated with complete PRR questionnaire responses

## Drawbacks

1. **Additional complexity in kubelet:** This feature adds a new controller to the kubelet that must watch and cache DRA objects. This increases kubelet complexity and maintenance burden.

2. **API surface expansion:** Adding `draDeviceFieldRef` to the Downward API increases the API surface area and creates a dependency between the core API and DRA, which is still an alpha/beta feature.

3. **Limited to kubelet resolution:** Unlike other Downward API fields that could theoretically be resolved by controllers, this feature requires kubelet-side resolution due to the need for node-local ResourceSlice data. This limits flexibility.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

None. This feature will be developed within existing Kubernetes repositories:
- API changes in `kubernetes/kubernetes` (staging/src/k8s.io/api)
- Kubelet implementation in `kubernetes/kubernetes` (pkg/kubelet)
- Tests in `kubernetes/kubernetes` (test/integration, test/e2e, test/e2e_node)
- Documentation in `kubernetes/website`
