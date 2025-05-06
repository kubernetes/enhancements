<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

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
# [KEP-5027](https://github.com/kubernetes/enhancements/issues/5027): DRA: admin-controlled device attributes


<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
  - [Merging ResourceSlicePatches and ResourceSlices](#merging-resourceslicepatches-and-resourceslices)
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
  - [Admin-intent in ResourceSlice](#admin-intent-in-resourceslice)
  - [Storing result of patching in ResourceSlice](#storing-result-of-patching-in-resourceslice)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
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

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

With Dynamic Resource Allocation (DRA), DRA drivers publish information about
the devices that they manage in ResourceSlices. This information is used by the
scheduler when selecting devices for user requests in ResourceClaims.

This KEP adds a Kubernetes API that privileged users, typically cluster
administrators or control plane controllers, can use to override or extend that information. This can be
permanent as part of the installation of a DRA driver to adapt the driver to
the cluster or temporary as part of cluster maintenance. An extension of the
API adds [taints](../5055-dra-device-taints-and-tolerations/README.md).

## Motivation

### Goals

- Enable [admin-controlled](../5055-dra-device-taints-and-tolerations/README.md) device taints.

- Enable updating how devices are seen in the cluster without having to use
  driver-specific APIs which influence what a driver puts into ResourceSlices.

### Non-Goals

- At least for alpha: extend `kubectl` to provide a unified view of devices
  together with all patches that apply to them.

## Proposal

The intent to patch device attributes must be recorded persistently so that
it is preserved even when a ResourceSlice gets removed or updated. To achieve
this, a new cluster-scoped ResourceSlicePatch type gets added. A single
ResourceSlicePatch object specifies device attributes that apply to all
devices matching a CEL expression (i.e. the same way as users select devices in
a ResourceClaim) and/or some additional criteria (device class,
driver/pool/device name).

The scheduler must merge these additional attributes with the ones provided by
the DRA drivers on-the-fly while it gathers information about available
devices.

### Notes/Constraints/Caveats

Users who look at ResourceSlices to figure out which devices are available also
need to consider ResourceSlicePatches to get the full picture. Copying from
the ResourceSlicePatch spec into the ResourceSlice status could help here,
but would not be instantaneous and potentially cause write amplification (one
ResourceSlicePatch affecting many different devices) and therefore is not
part of this proposal.

Perhaps `kubectl describe resourceslices` can be extended to include the
additional information. For now this is out of scope.

### Risks and Mitigations

Creating a ResourceSlicePatch is racing with on-going scheduling attempts.
This is unavoidable. Removing a device from a ResourceSlice has the same
problem: updates need to reach the scheduler before it can consider them.
Evaluating a patch on the client-side instead of [having a controller update
slices]((#storing-result-of-patching-in-resourceslice) mitigates this risk by
shortening the time window where updates must be sent to the scheduler.

From a security perspective, permission to patch device attributes is
expected to be limited to privileged users who already have the ability to add
or remove DRA drivers, so there won't be a substantial difference.

Performance in the scheduler could be an issue. This will be mitigated by
caching the patched devices and
(re-)applying patches only when they or the device definitions change, which
should be rare.

Patching directly in the informer event handlers may be fast enough. If it
turns out to slow down those handlers too much, then a workqueue with workers
may be needed to decouple updating the cache from the events which trigger
updating and to avoid slowing down the informers.

The scheduler's "slice changed" cluster events must be driven by that cache,
not the original informers, otherwise a ResourceSlice or ResourceSlicePatch
change could trigger a pod scheduling attempt before the slice cache is
up-to-date again.

## Design Details

### API

The ResourceSlicePatch is a cluster-scoped type in the `resource.k8s.io` API
group, initially in `v1alpha3` (the alpha version in Kubernetes 1.32). Because
it may be useful to clean up after disabling the feature and because the
device taint feature also uses this type, it gets served unconditionally as long as
the `v1alpha3` version is enabled. Fields related specifically to this KEP
are feature-gated.

```Go
type ResourceSlicePatch struct {
    metav1.TypeMeta
    // Standard object metadata
    // +optional
    metav1.ObjectMeta

    // Changing the spec automatically increments the metadata.generation number.
    Spec ResourceSlicePatchSpec
}

type ResourceSlicePatchSpec struct {
    // Devices defines how to patch device attributes and taints.
    Devices DevicePatch
}

// DevicePatch selects one or more devices by class, driver, pool, device names
// and/or CEL selectors. All of these criteria must be satisfied by a device, otherwise
// it is ignored by the patch. A DevicePatch with no selection criteria is
// valid and matches all devices.
type DevicePatch struct {
    // Filter defines which device(s) the patch is applied to.
    //
    // +optional
    Filter *DevicePatchFilter

    // If a ResourceSlice and a DevicePatch define the same attribute or
    // capacity, the value of the DevicePatch is used. If multiple
    // different DevicePatches match the same device, then the one with
    // the highest priority wins. If priorities are equal, the older
    // patch wins, where "older" is determined based on the creation time.
    // This ensures that adding a new patch does not
    // accidentally change the effect of some existing patch unless
    // that is clearly intended according to the priority. Updates
    // do not change the creation time, so it could still happen that
    // a more recent change is preferred because it happens to be in
    // an older DevicePatch. Overall it is better to set the
    // priority to different values to avoid such ambiguities.
    //
    // +optional
    Priority *int

    // Attributes defines the set of attributes to patch for matching devices.
    // The name of each attribute must be unique in that set and
    // include the domain prefix.
    //
    // In contrast to attributes in a ResourceSlice, entries here are allowed to
    // be marked as empty by setting their null field. Such entries remove the
    // corresponding attribute in a ResourceSlice, if there is one, instead of
    // overriding it. Because entries get removed and are not allowed in
    // slices, CEL expressions do not need to deal with null values.
    //
    // The maximum number of attributes and capacities in the DevicePatch combined is 32.
    // This is an alpha field and requires enabling the DRAAdminControlledDeviceAttributes
    // feature gate.
    //
    // +optional
    // +featureGate:DRAAdminControlledDeviceAttributes
    Attributes map[FullyQualifiedName]NullableDeviceAttribute

    // ^^^
    // The size limit is the same as for attributes and capacities in a ResourceSlice.
    // We could make it larger here because we are less constrained by overall object
    // size, but it seems unnecessary.

    // Capacity defines the set of capacities to patch for matching devices.
    // The name of each capacity must be unique in that set and
    // include the domain prefix.
    //
    // Removing a capacity is not supported. It can be reduced to 0 instead.
    //
    // The maximum number of attributes and capacities in the DevicePatch combined is 32.
    // This is an alpha field and requires enabling the DRAAdminControlledDeviceAttributes
    // feature gate.
    //
    // +optional
    // +featureGate:DRAAdminControlledDeviceAttributes
    Capacity map[FullyQualifiedName]DeviceCapacity

    // ^^^^
    // The assumption here is that all device types will have attributes and capacities,
    // similar to the current BasicDevice type. Therefore the overrides are not made
    // specific to certain device types.
}

// DevicePatchFilter defines which device(s) a DevicePatch applies to.
// All criteria defined here must be satisfied for a device to be
// patched.
type DevicePatchFilter struct {
    // If DeviceClassName is set, the selectors defined there must be
    // satisfied by a device to be patched. This field corresponds
    // to class.metadata.name.
    //
    // +optional
    DeviceClassName *string

    // If driver is set, only devices from that driver are patched.
    // This fields corresponds to slice.spec.driver.
    //
    // +optional
    Driver *string

    // If pool is set, only devices in that pool are patched.
    // This fields corresponds to slice.spec.pool.name.
    //
    // Setting also the driver name may be useful to avoid
    // ambiguity when different drivers use the same pool name,
    // but this is not required because selecting pools from
    // different drivers may also be useful, for example when
    // drivers with node-local devices use the node name as
    // their pool name.
    //
    // +optional
    Pool *string

    // If device is set, only devices with that name are patched.
    // This field corresponds to slice.spec.devices[].name.
    //
    // Setting also driver and pool may be required to avoid ambiguity,
    // but is not required.
    //
    // +optional
    Device *string

    // Selectors define criteria which must be satisfied by a
    // device to be patched. All selectors must be satisfied.
    //
    // +optional
    // +listType=atomic
    Selectors []DeviceSelector

    // ^^^
    //
    // Selectors is a list for the same reason why a request has a list: at some point
    // we might have entries which use some yet to be defined mechanism which isn't
    // CEL.
}
```

To distinguish intentionally empty attributes from attributes which have some
future, unknown content and thus only seem to be empty to an older client, a
special "null value" gets introduced:

```Go
// NullableDeviceAttribute must have exactly one field set.
// It has the exact same fields as a DeviceAttribute plus `null` as
// an additional alternative.
type NullableDeviceAttribute struct {
    DeviceAttribute

    // NullValue, if set, marks an intentionally empty attribute.
    //
    // +optional
    // +oneOf=ValueType
    NullValue *NullValue `json:"null,omitempty" ...`
}

// ^^^
// `NullableDeviceAttribute` as an extension ensures that the OpenAPI
// for ResourceSlice remains unchanged. Using the same type with
// a `NullValue` that can be set only in one type is less clear.

type NullValue struct {}
```

### Merging ResourceSlicePatches and ResourceSlices

Helper code which keeps an up-to-date list of devices with all patches added
to them will be provided as part of `k8s.io/dynamic-resource-allocation`. It
will be based on informers such that evaluating the filter only is
necessary when ResourceSlices or ResourceSlicePatches change.

A CEL expression that fails to evaluate to a boolean for a device (runtime
error like looking up an attribute that isn't defined, wrong result type, etc.)
is considered faulty. The patch then does not apply to the device where it
failed and an event will be generated for the ResourceSlicePatch with the
faulty CEL expression.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

<!--
Generated with:

go test -cover ./pkg/apis/resource/validation  ./staging/src/k8s.io/dynamic-resource-allocation/structured | sed -e 's/.*\(k8s.io[a-z/-]*\).*coverage: \(.*\) of statements/- `\1`: \2/' | sort

-->

v1.32.0:

- `k8s.io/dynamic-resource-allocation/structured`: 91.3%
- `k8s.io/kubernetes/pkg/apis/resource/validation`: 98.6%

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

Additional scenarios will be added to `test/integration/scheduler_perf`, not
just for correctness but also to evaluate a potential performance impact.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

One E2E test scenario is to change attributes and then run pods which select devices
based on those modified attributes such that unmodified devices don't match.

- <test>: <link to test coverage>

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Additional tests are in Testgrid and linked in KEP

#### GA

- 3 examples of real-world usage
- Allowing time for feedback
- [Conformance tests]

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

### Upgrade / Downgrade Strategy

Patching devices gets disabled when downgrading to a release without support
for it or when disabling the feature. The effect is that pods get scheduled as
if the ResourceSlicePatches didn't exist. Because they are completely
stand-alone, there is no effect on ResourceSlices or ResourceClaims.

### Version Skew Strategy

During version skew where the apiserver supports the feature and the scheduler
doesn't, users can create ResourceSlicePatches without encountering errors or
warnings, but they won't have any effect.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

It is possible to disable the feature through the feature gate while leaving
the API group enabled. This enables cleanup through the API.

Re-enabling is supported because ResourceSlicePatches remain in etcd even
if they are inaccessible.

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DRAAdminControlledDeviceAttributes
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler
- [X] Other
  - Describe the mechanism: resource.k8s.io/v1alpha3 API group
  - Will enabling / disabling the feature require downtime of the control
    plane? Yes, in the apiserver.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. The behavior of scheduling changes when it was in use.
Running applications are not affected.

###### What happens if we reenable the feature if it was previously rolled back?

Admin-controlled attributes take effect again for scheduling.
Running applications are not affected because allocations are
never updated once they are made.

Note that this is different for reenabling device taints (KEP 5055): that can
cause pod to get evicted.

###### Are there any tests for feature enablement/disablement?

This will be covered through unit tests for the apiserver and scheduler.

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

Applying patches to devices scales with `number of ResourceSlicePatches` *
`number of devices` when CEL selectors need to be evaluated. Without them,
filtering scales with `number of ResourceSlicePatches` * `number of
ResourceSlices` but then may still need to compare device names and of course
modify selected devices.

###### Will enabling / using this feature result in any new API calls?

A fixed, small number of clients (primarily the scheduler) need to start
watching ResourceSlicePatches.

###### Will enabling / using this feature result in introducing new API types?

ResourceSlicePatches must be created explicitly by admins or controller
operated by admins. Kubernetes itself does not create them.

The number of ResourceSlicePatches is expected to be orders of
magnitude smaller than the number of ResourceSlices.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Pod scheduling should be as fast as would be without this feature, because in
both cases it starts with listing all devices. That information is local and
comes either from an informer cache or a cache of patched devices.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Filtering and patching are local operations, with no impact on the cluster. To
prevent doing the same work repeatedly, it will be implemented so that it gets
done once and then only processes changes. This increases CPU and RAM
consumption. But even if all devices should get patched (which is unlikely), memory
will be shared between objects in the informer cache and in the patch cache, so
it will not be doubled.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No, because the feature is not used on nodes.

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

- 1.33: first KEP revision, implementation postponed until there is a more specific need for it

## Drawbacks

Distributing information across different objects of different types makes it
harder for users to get a complete view.

## Alternatives

### Admin-intent in ResourceSlice

Instead of ResourceSlicePatch as a separate type, new fields in the
ResourceSlice status could be modified by an admin. That has the problem that
the ResourceSlice object might get deleted while doing cluster maintenance like
a driver update, in which case the admin intent would get lost.

### Storing result of patching in ResourceSlice

A controller could read ResourceSlicePatches and apply them to
ResourceSlices. Then consumers like the scheduler and users would only need to
look at ResourceSlices. This has several drawbacks.

We would need to duplicate the attributes in the slice status. If we didn't and
directly modified the spec, this patch controller and the CSI driver as the
owner of the slice spec would fight against each other. Also, after removing a
patch the original state must be available somewhere, otherwise the controller
cannot restore it.

Duplicating the attributes might make a slice too large. The limits were chosen
so that we have some space left for a status, but not enough for a status that
is potentially as large as the spec.

Creating a single ResourceSlicePatch could force the controller to update a
potentially large number of ResourceSlices. When using rate limiting, updating
them all will take longer than client-side patching. When not using rate
limiting, this could overwhelm the apiserver.
