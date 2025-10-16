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

This KEP proposes a Downward API for Dynamic Resource Allocation (DRA) device attributes, implemented entirely in the kubelet. Workloads like KubeVirt need device identifiers (e.g., PCIe bus address for physical GPUs, mediated device UUID for virtual GPUs) to configure device access inside guests. While these identifiers exist in DRA objects (`ResourceClaim`, `ResourceSlice`), they are not currently consumable via the Pod's Downward API. This proposal adds a new Downward API selector (`resourceSliceAttributeRef`) for environment variables. The kubelet will run a local DRA attributes controller that watches `ResourceClaim` and `ResourceSlice` objects, caches resolved attributes per Pod resource claim/request, and resolves `resourceSliceAttributeRef` on demand.

## Motivation

Workloads that need to interact with DRA-allocated devices (like KubeVirt virtual machines) require access to device-specific identifiers such as PCIe bus addresses or mediated device UUIDs. In order to fetch the attributes from allocated device, users first have to go to ResourceClaimStatus, find the request and device name, and then look up the resource slice with device name to get the attribute value. Ecosystem project like KubeVirt must resort to custom controllers that watch these objects and inject attributes via annotations/labels or other custom mechanisms, often leading to fragile, error-prone and racy designs.

The Kubernetes Downward API provides a standard mechanism for exposing Pod and container metadata to workloads. Extending this API to support DRA device attributes would enable workloads to discover device information without requiring additional custom controllers or privileged access to the Kubernetes API.

### Goals

- Provide a stable Downward API path for device attributes associated with `pod.spec.resourceClaims[*]` requests
- Support device attributes from `ResourceSlice` that are requested by user in pod spec
- Maintain compatibility with existing DRA drivers without requiring changes to driver interfaces

### Non-Goals

- Expose the entirety of `ResourceClaim`/`ResourceSlice` objects in the Downward API
- Allow arbitrary JSONPath into external objects via Downward API
- Change or extend DRA driver interfaces
- Support dynamic updates to device attributes after Pod container startup (for env vars)
 - Propagate or snapshot attributes from `ResourceSlice` into `ResourceClaim` at allocation time (no scheduler involvement)

## Proposal

This proposal introduces a new Downward API selector (`resourceSliceAttributeRef`) that allows Pods to reference DRA device attributes in environment variables. The kubelet will implement a local DRA attributes controller that:

1. Watches Pods scheduled to the node and identifies those with `pod.spec.resourceClaims`
2. Watches `ResourceClaim` objects in the Pod's namespace to retrieve allocation information
3. Watches `ResourceSlice` objects for the node and driver to resolve device attributes
4. Maintains a per-Pod cache of `(claimName, requestName) -> {attribute: value}` mappings
5. Resolves `resourceSliceAttributeRef` references when containers start

Downward API references expose one attribute per reference. The kubelet resolves only the attribute explicitly referenced via `ResourceSliceAttributeSelector`.

### User Stories (Optional)

#### Story 1

As a KubeVirt developer, I want the virt-launcher Pod to automatically discover the PCIe root address of an allocated physical GPU via environment variables, so that it can construct the libvirt domain XML to pass through the device to the virtual machine guest without requiring a custom controller.

#### Story 2

As a DRA driver author, I want my driver to remain unchanged while allowing applications to consume device attributes (like `resource.kubernetes.io/pcieRoot` or `dra.kubervirt.io/mdevUUID`) through the native Kubernetes Downward API.

### Notes/Constraints/Caveats (Optional)

- Environment variables are set at container start time: Once a container starts, its environment variables are immutable. If device attributes change after container start, env vars will not reflect the change.
- Resolution timing: Attributes are resolved at container start time (not at allocation time). There is no scheduler-side copying of attributes into `ResourceClaim`.
- ResourceSlice churn: Resolution uses the contents of the matching `ResourceSlice` at container start. If the `ResourceSlice` (or the requested attribute) is missing at that time, kubelet records an event and fails the pod start.
- Attribute names: Any attribute name that exists in the ResourceSlice can be referenced. 

### Risks and Mitigations

Risk: Exposing device attributes might leak sensitive information.
Mitigation: Only one attribute is exposed per reference. The NodeAuthorizer ensures kubelet only accesses ResourceClaims and ResourceSlices for Pods scheduled to that node. Attributes originate from the DRA driver via `ResourceSlice`; cluster policy should ensure sensitive data is not recorded there.

Risk: Kubelet performance impact from watching ResourceClaim and ResourceSlice objects.
Mitigation: Use shared informers with proper indexing to minimize API server load. Scope watches to node-local resources only. Monitor cache memory usage and set reasonable limits.

Risk: API surface expansion could make the Downward API overly complex.
Mitigation: Keep the API minimal and type-safe with a dedicated selector; no arbitrary JSONPath. Require feature gate enablement in alpha to gather feedback before expanding.

Risk: Compatibility with future DRA changes.
Mitigation: Implementation is decoupled from DRA driver interfaces. Changes to DRA object structure are handled in the kubelet controller. Attribute names are standardized and versioned.

## Design Details

### API Changes

#### New Downward API Selector: ResourceSliceAttributeSelector

A new typed selector is introduced in `core/v1` that can be used in both environment variable sources and projected downward API volume items:

```go
// ResourceSliceAttributeSelector selects a DRA-resolved device attribute for a given claim+request.
// +featureGate=DRADownwardDeviceAttributes
// +structType=atomic
type ResourceSliceAttributeSelector struct {
    // ClaimName must match pod.spec.resourceClaims[].name.
    // +required
    ClaimName string `json:"claimName"`
    
    // RequestName must match the corresponding ResourceClaim.spec.devices.requests[].name.
    // +required
    RequestName string `json:"requestName"`
    
    // Attribute specifies which device attribute to expose from the ResourceSlice.
    // The attribute name must be present in the ResourceSlice's device attributes.
    // +required
    Attribute string `json:"attribute"`
}

// In core/v1 EnvVarSource:
type EnvVarSource struct {
    // ...existing fields...
    
    // ResourceSliceAttributeRef selects a DRA device attribute for a given claim+request.
    // +featureGate=DRADownwardDeviceAttributes
    // +optional
    ResourceSliceAttributeRef *ResourceSliceAttributeSelector `json:"resourceSliceAttributeRef,omitempty"`
}

// (Projected volume support deferred to beta)
```

Validation:
- Enforce exactly one of `fieldRef`, `resourceFieldRef`, or `resourceSliceAttributeRef` in both env and volume items
- Validate `claimName` and `requestName` against DNS label rules
- No API-level enumeration of attribute names; kubelet resolves attributes that exist in the matching `ResourceSlice` at runtime


####

### Kubelet Implementation

The kubelet runs a local DRA attributes controller that:

1. Watches Pods: Identifies Pods on the node with `pod.spec.resourceClaims` and tracks their `pod.status.resourceClaimStatuses` to discover generated ResourceClaim names
2. Watches ResourceClaims: For each relevant claim, reads `status.allocation.devices.results[*]` and maps entries by request name
3. Watches ResourceSlices: Resolves standardized attributes from `spec.devices[*].attributes` for the matching device name
4. Maintains Cache: Keeps a per-Pod map of `(claimName, requestName) -> {attribute: value}` with a readiness flag

Resolution Semantics:
- Prioritized List compatibility:
  - Clients do not need to know the number of devices a priori. At container start, kubelet aggregates the attribute across all devices actually allocated for the request and joins the values with "," in allocation order. If any allocated device lacks the attribute, resolution fails and the pod start errors.
- Cache entries are updated on claim/slice changes
- For container environment variables, resolution happens at container start using the latest ready values
- Attributes are not frozen at allocation time; scheduler and controllers are not involved in copying attributes

- Failure on missing data: If the `ResourceSlice` is not found, or the attribute is absent on any allocated device at container start, kubelet records a warning event and returns an error to the sync loop. The pod start fails per standard semantics (e.g., `restartPolicy` governs restarts; Jobs will fail the pod).
- Multi-device requests: Kubelet resolves the attribute across all allocated devices for the request, preserving allocation order, and joins values with a comma (",") into a single string. If any allocated device does not report the attribute, resolution fails (pod start error).

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
        resourceSliceAttributeRef:
          claimName: pgpu-claim
          requestName: pgpu-request
          attribute: resource.kubernetes.io/pcieRoot
          # If multiple devices are allocated for this request, values are joined with "," in allocation order.
```

 

 

### Feature Gate

- Name: `DRADownwardDeviceAttributes`
- Stage: Alpha (v1.35)
- Components: kube-apiserver, kubelet, kube-scheduler
- Enables: 
  - New `resourceSliceAttributeRef` selector in env vars
  - DRA attributes controller in kubelet

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

- Feature gate toggling: Verify API rejects `resourceSliceAttributeRef` when feature gate is disabled
- End-to-end resolution: Create Pod with resourceClaims, verify env vars contain correct attribute values
- Negative cases: Missing allocation, missing `ResourceSlice`, missing attribute on any allocated device — expect warning event and pod start failure
- Multi-device semantics: Joining order and delimiter; mixed presence of attributes across allocated devices should cause failure

Tests will be added to `test/integration/kubelet/` and `test/integration/dra/`.

##### e2e tests

E2E tests will validate real-world scenarios:

- KubeVirt-like workload: Pod with GPU claim consumes `resource.kubernetes.io/pcieRoot` via environment variable
- Multi-claim Pod: Pod with multiple resource claims, each with different attributes
 
- Feature gate disabled: Verify graceful degradation when gate is off
- Node failure scenarios: Test behavior when kubelet restarts or node is drained

Tests will be added to `test/e2e/dra/` and `test/e2e_node/downwardapi_test.go`.

### Graduation Criteria

#### Alpha (v1.35)

- Feature implemented behind `DRADownwardDeviceAttributes` feature gate
- API types added: `resourceSliceAttributeRef` in `core/v1.EnvVarSource`
- Kubelet DRA attributes controller implemented
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
- Existing Pods without `resourceSliceAttributeRef` are unaffected
- To use the feature, users must:
  1. Enable the `DRADownwardDeviceAttributes` feature gate on kube-apiserver and kubelet
  2. Update Pod specs to use `resourceSliceAttributeRef` in env vars or volumes
- No changes to existing DRA drivers are required

**Downgrade:**
- If downgrading from a version with this feature to one without it:
- Pods using `resourceSliceAttributeRef` will fail API validation and cannot be created
- Existing Pods with `resourceSliceAttributeRef` will continue to run but cannot be updated
- Users should remove `resourceSliceAttributeRef` from Pod specs before downgrade
- Feature gate can be disabled to reject new Pods with `resourceSliceAttributeRef`
- Kubelet will ignore `resourceSliceAttributeRef` when feature gate is disabled

### Version Skew Strategy

**Control Plane and Node Coordination:**
- This feature primarily involves the kubelet and API server
- The API server validates `resourceSliceAttributeRef` (feature gate enabled)
- The kubelet resolves `resourceSliceAttributeRef` at runtime

**Version Skew Scenarios:**

1. **Older kubelet (n-1, n-2, n-3) with newer API server:**
   - API server accepts Pods with `resourceSliceAttributeRef` (if feature gate is enabled)
   - Older kubelet without the feature will ignore `resourceSliceAttributeRef` (it is dropped during decoding)
   - Containers still start; env vars/volumes referencing `resourceSliceAttributeRef` will not be populated
   - **Risk**: Workloads relying on these values may misbehave
   - **Mitigation**: Avoid relying on the field until all kubelets are upgraded; gate scheduling to upgraded nodes (e.g., using node labels/taints) or keep the feature gate disabled on the API server until nodes are updated

2. **Newer kubelet with older API server:**
   - Older API server rejects Pods with `resourceSliceAttributeRef` (unknown field)
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

No. Enabling the feature gate only adds new optional API fields (`resourceSliceAttributeRef`) to `EnvVarSource`. Existing Pods and workloads are unaffected. Users must explicitly opt in by:
1. Using `resourceSliceAttributeRef` in Pod env vars

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. The feature can be disabled by setting the feature gate to `false` and restarting the API server and kubelets.

**Consequences:**
- New Pods using `resourceSliceAttributeRef` will be rejected by the API server
- Existing running Pods with `resourceSliceAttributeRef` will continue to run, but environment variables set at container start remain unchanged
- Pods with `resourceSliceAttributeRef` cannot be updated while feature gate is disabled

**Recommendation:** Before disabling, ensure no critical workloads depend on `resourceSliceAttributeRef` or migrate them to alternative mechanisms (e.g., annotations).

###### What happens if we reenable the feature if it was previously rolled back?

Re-enabling the feature gate restores full functionality:
- API server accepts Pods with `resourceSliceAttributeRef`
 
- New Pods will work correctly
- Existing Pods (created while feature was disabled) are unaffected unless they already use `resourceSliceAttributeRef`

No data migration or special handling is required.

###### Are there any tests for feature enablement/disablement?

Yes:
- Unit tests will verify API validation behavior with feature gate on/off
- Integration tests will verify:
- Pods with `resourceSliceAttributeRef` are rejected when feature gate is disabled
- Pods with `resourceSliceAttributeRef` are accepted when feature gate is enabled
- Kubelet correctly resolves attributes when feature gate is enabled
- Kubelet ignores `resourceSliceAttributeRef` when feature gate is disabled

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

No. This feature adds new fields to existing API types (`core/v1.EnvVarSource` and `resource.k8s.io/v1.DeviceRequest`) but does not introduce new API object types.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes
  - Pod, adds a field in pod

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes, but the impact should be minimal:

- Pod startup latency: Kubelet must resolve `resourceSliceAttributeRef` values before starting containers with environment variables, but the impact of this is minimized by local informer based lookup

- The feature does not affect existing SLIs/SLOs for clusters not using DRA or for Pods not using `resourceSliceAttributeRef`.

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
- Pathological case: Malicious user creates many Pods with `resourceSliceAttributeRef` to exhaust kubelet memory
- Risk: Low. Cache size bounded by max pods per node (110) × cache entry size (~1KB) = ~110KB
- Mitigation: Existing pod limits prevent unbounded growth; cache eviction for terminated Pods

Inodes:
 
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

2. **API surface expansion:** Adding `resourceSliceAttributeRef` to the Downward API increases the API surface area and creates a dependency between the core API and DRA, which is still an alpha/beta feature.

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
