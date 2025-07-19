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
# KEP-5234: DRA: ResourceSlice Mixins

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [ResourceSlice resources will be harder to understand](#resourceslice-resources-will-be-harder-to-understand)
    - [Flatting of ResourceSlices might be needed by all tools using the API](#flatting-of-resourceslices-might-be-needed-by-all-tools-using-the-api)
    - [Mixins and more counters might worsen worst-case scheduling](#mixins-and-more-counters-might-worsen-worst-case-scheduling)
- [Design Details](#design-details)
  - [API](#api)
  - [Implementation](#implementation)
  - [Limits](#limits)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
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

With Dynamic Resource Allocation (DRA), DRA drivers publish information about
the devices that they manage in ResourceSlices. This information is used by the
scheduler when selecting devices for user requests in ResourceClaims.

With this KEP, DRA drivers can define metadata in mixins separately from specific
devices and include them in a device by reference. This reduces the duplication
in ResourceSlices and allows for more compact device definitions. Mixins can
also be used in counter sets.

## Motivation

DRA requires that drivers publish all available devices on a node/cluster in
`ResourceSlice` objects. There are scenarios where the number of devices
can be pretty large and each device might have a relatively large amount
of metadata associated with it, primarily in the form of attributes and
capacity. This has a few consequences:

* Several of the devices might have similar metadata, resulting in a lot of
  duplication between the published devices.
* The size of the data required to specify each device reduces the number of
  devices that can be defined in a single ResourceSlice without hitting the
  limit on the total size of objects in Kubernetes (1.5MB by default, but can
  be changed).

The latter can be addressed by splitting the devices across multiple
`ResourceSlice`s within a single pool, but that isn't always an option.
In particular, DRA currently doesn't allow sharing counters across `ResourceSlice`s,
meaning that the number of devices that can fit into a single `ResourceSlice`
also limits the number of partitionable devices for a single physical device.

### Goals

- Enable a more compact way to define devices in ResourceSlices so duplication can
  be reduced and a larger number of devices can be published within a single
  ResourceSlice.
- Enable defining counter sets with more counters and devices with more counsumed
  counters.

### Non-Goals

- Not part of the plan for alpha: developing kubectl command or plugin to let
  users see the flattened device definitions. Mixins does make it harder to find
  the full definition for a specific device, so this might be added to the scope
  for Beta or GA.
- Enable devices to have more than 32 attributes and capacities. Increasing this would
  have implications for the CEL cost functions, so we are not looking to increase
  the limits as part of this KEP.

## Proposal

The proposal has two parts to it, the definition of mixins and the
mechanism for referencing mixins from devices and counter sets.

A new `Mixins` field will be added to the `ResourceSliceSpec` as an
optional field of type `ResourceSliceMixins`. It will have three properties, one for each of the three
types of mixins that will be supported:

1. The `CounterSet` field defines a list of named `CounterSetMixins`.
   These define counters that can be used to extend the counters
   explicitly defined in a `CounterSet`. This allows for reduced duplication
   if there are many identical physical devices that must be represented as
   `CounterSet`s. `CounterSetMixins` cannot be referenced directly by devices.

1. The `Device` field is a list of named `DeviceMixin`s. These define
   attributes and capacities that can be used to extend what is defined
   explicitly in `Device`. `DeviceMixin`s cannot be allocated directly, but can
   only be referenced by devices.

1. The `DeviceCounterConsumption` field defines a list of named
   `DeviceCounterConsumptionMixin`s. These define the pattern of consumption of
   counters, distinct from the specific underlying counter set from which they
   are being consumed. The `CounterSet` from which the counters will be consumed
   is not specified in the `DeviceCounterConsumptionMixin`, but rather provided
   when the mixin is referenced from the device.

The mixins are referenced using the same pattern in all three places. The field
is named `Includes` and will contain a list of references to the mixins. The mixins
are applied in the order listed, meaning that later mixins will overwrite earlier
ones in case of conflicts. Properties set directly on the `CounterSet`, `Device` or
`DeviceCounterConsumption` will always override mixins.

1. The `Includes` field on `CounterSet` is a list of references
   to mixins defined in the `CounterSet` field on the `ResourceSliceMixins`.

1. The `Includes` field on `Device` is a list of references to mixins defined
   in the `Device` field on the `ResourceSliceMixins`.

1. The `Includes` field on `DeviceCounterConsumption` is a list of references
   to mixins defined in the `DeviceCounterConsumption` field on the `ResourceSliceMixins`.

With these changes, attributes, capacity, and counters that are shared across
devices or counter sets can be split out into mixins, thereby reducing
duplication and reducing the size of the ResourceSlice object.

### Risks and Mitigations

This change doesn't really affect the functionality of DRA, it just
provides a more compact way to define devices in ResourceSlices. But it
still have some consequences worth pointing out here.

#### ResourceSlice resources will be harder to understand

The biggest challenge with this change is that it adds a level of
indirection for the `Device` and `CounterSet` definitions, meaning
that it gets harder to understand the ResourceSlice objects.

We have discussed adding a kubectl command or a plugin that will allow
users to see the fully flattened versions of a ResourceSlice. But this
is not in scope for alpha.

#### Flatting of ResourceSlices might be needed by all tools using the API

Any tool that needs to understand the full device definition will need to
flatten the ResourceSlice. This can lead to duplicate effort across different
tools and potential for implementations that differ in meaningful ways. This can
be addressed by providing reusable libraries that can be leveraged by other
tools.

#### Mixins and more counters might worsen worst-case scheduling

This will not negatively effect existing scheduling performance of existing
ResourceSlice definitions, but DRA driver authors taking advantage of mixins should
be made aware of possible performance effects due to this increased referential complexity.

This also demonstrates that DRA driver authors should consider performance when they
write drivers and decide how to structure devices into ResourceSlices and pools. Information
and best-practices about how to write drivers are available in
https://github.com/kubernetes-sigs/dra-example-driver and this will also include information
about performance and scalability.

## Design Details

### API

The exact set of proposed API changes can be seen below (`...` is used in places where new fields
are added to existing types):
```go
// ResourceSliceSpec contains the information published by the driver in one ResourceSlice.
type ResourceSliceSpec struct {
  ...

  // Mixins defines the mixins available for devices and counter sets
  // in the ResourceSlice.
  //
  // +featureGate=DRAResourceSliceMixins
  // +optional
  Mixins *ResourceSliceMixins
}

type CounterSet struct {
  ...

  // Includes defines a list of references to CounterSetMixin.
  // The counters listed in the mixins will be added to the counters
  // available in this CounterSet.
  //
  // The counters of each included mixin are applied to this counter set in
  // order. Conflicting counters from multiple mixins are taken from the
  // last mixin listed. Counters set on the CounterSet will always override
  // counters from mixins.
  //
  // The mixins referenced here must be defined in the same
  // ResourceSlice.
  //
  // The maximum number of includes is 8.
  //
  // +featureGate=DRAResourceSliceMixins
  // +listType=atomic
  // +optional
  Includes []string
}

// ResourceSliceMixins defines mixins for the ResourceSlice.
//
// The main purposes of these mixins is to reduce the memory footprint
// of devices since they can reference the mixins provided here rather
// than duplicate them.
type ResourceSliceMixins struct {
  // Device represents a list of device mixins, i.e. a collection of
  // shared attributes and capacities that an actual device can "include"
  // to extend the set of attributes and capacities it already defines.
  //
  // The maximum number of device mixins in a ResourceSlice is 128
  // and the total number of attributes and capacities across device
  // mixins and devices in a single ResourceSlice must not exceed 4096.
  //
  // +optional
  // +listType=atomic
  Device []DeviceMixin

  // DeviceCounterConsumption represents a list of counter
  // consumption mixins, each of which contains a set of counters
  // that a device will consume from a counter set.
  //
  // The maximum number of device counter consumption mixins in a
  // ResourceSlice is 128 the total number of consumed counters across device
  // counter consumption mixins and devices in a single
  // ResourceSlice must not exceed 2048.
  //
  // +optional
  // +listType=atomic
  DeviceCounterConsumption []DeviceCounterConsumptionMixin

  // CounterSet represents a list of counter set mixins, i.e.
  // a collection of counters that a CounterSet can "include"
  // to extend the set of counters it already defines.
  //
  // The maximum number of counter set mixins in a ResourceSlice is
  // 32 and the total number of counters across counter set mixins and
  // counter sets in a single ResourceSlice must not exceed 256.
  //
  // +optional
  // +listType=atomic
  CounterSet []CounterSetMixin
}

// DeviceMixin defines a mixin that can be referenced from a device.
type DeviceMixin struct {
  // Name is a unique identifier among all device mixins in the ResourceSlice.
  // It must be a DNS label.
  //
  // +required
  Name string

  // Attributes defines the set of attributes for this mixin.
  // The name of each attribute must be unique in that set.
  //
  // To ensure this uniqueness, attributes defined by the vendor
  // must be listed without the driver name as domain prefix in
  // their name. All others must be listed with their domain prefix.
  //
  // The maximum number of attributes and capacities across all devices
  // and device mixins in a ResourceSlice is 4096. When flattened, the
  // total number of attributes and capacities for each device must not
  // exceed 32.
  //
  // +optional
  Attributes map[QualifiedName]DeviceAttribute

  // Capacity defines the set of capacities for this mixin.
  // The name of each capacity must be unique in that set.
  //
  // To ensure this uniqueness, capacities defined by the vendor
  // must be listed without the driver name as domain prefix in
  // their name. All others must be listed with their domain prefix.
  //
  // The maximum number of attributes and capacities across all devices
  // and device mixins in a ResourceSlice is 4096. When flattened, the
  // total number of attributes and capacities for each device must not
  // exceed 32.
  //
  // +optional
  Capacity map[QualifiedName]DeviceCapacity
}

// DeviceCounterConsumptionMixin defines a mixin that
// devices can include to extend or override the set of counters
// that a device consumes from a counter set.
type DeviceCounterConsumptionMixin struct {
  // Name is a unique identifier among all device counter consumption
  // mixins in the ResourceSlice. It must be a DNS label.
  //
  // +required
  Name string

  // Counters defines a set of counters
  // that a device will consume from a counter set.
  //
  // The maximum number device counter consumption all device counter consumptions
  // and device counter consumption mixins in a ResourceSlice is 2048.
  //
  // +required
  Counters map[string]Counter
}

// CounterSetMixin defines a mixin that a capacity pool can include.
type CounterSetMixin struct {
  // Name is a unique identifier among all capacity pool mixins in the ResourceSlice.
  // It must be a DNS label.
  //
  // +required
  Name string

  // Counters defines the set of counters for this mixin.
  // The name of each counter must be unique in that set and must be a DNS label.
  //
  // The maximum number of counters across all counter sets and counter set
  // mixins in a ResourceSlice is 256.
  //
  // +required
  Counters map[string]Counter
}

type Device struct {
  ...

  // Includes defines a list of references to DeviceMixin. The attributes
  // and capacity listed in the mixins will be added to the device.
  //
  // The attributes and capacity of each included mixin are applied in
  // order. Conflicting attributes/capacity from multiple mixins are taken from the
  // last mixin listed. Attributes and capacity set on the device will
  // always override those from mixins.
  //
  // The mixins referenced here must be defined in the same
  // ResourceSlice.
  //
  // The maximum number of includes is 8.
  //
  // +featureGate=DRAResourceSliceMixins
  // +optional
  // +listType=atomic
  Includes []string
}

type DeviceCounterConsumption struct {
  ...

  // Includes defines a list of references to DeviceCounterConsumptionMixin.
  // The counters listed in the mixins will be added to the
  // counters that will be consumed by the device.
  //
  // The counters of each included mixin are applied in
  // order. Conflicting counters from multiple mixins are taken from the
  // last mixin listed. Counters set on the DeviceCounterConsumption will
  // always override counters from mixins.
  //
  // The mixins referenced here must be defined in the same
  // ResourceSlice.
  //
  // The maximum number of includes is 4.
  //
  // +featureGate=DRAResourceSliceMixins
  // +optional
  // +listType=atomic
  Includes []string
}
```

### Implementation

The DRA scheduler will keep the mixin structure throughout the scheduling process
as much as possible and avoid completely flattening the ResourceSlices. This is
to avoid additional memory usage that might come as a result. For example, we
plan to walk the mixins as part of the CEL variable lookup to avoid having to
flatten the device representation.

If the mixins feature is disabled, any devices or counter sets that
references mixins will be droppped. This also means that all devices that references
a dropped counter set will also be dropped. The result is that the scheduler will not
see those devices. From the users point of view, the consequence is that the scheduler
might pick a different device than expected or fails to allocate any device at all. But we
think this failure mode is preferable than allowing the scheduler to make allocation decisions
based on incomplete data.

### Limits

For DRA, we have gradually moved away from individual per-slice and per-map limits towards
aggregating at the higher level. The reason for this is to give users maximum flexibility between
defining a small number of complex devices or a large number of simple devices in a single
ResourceSlice without exceeding the limit on the size of Kubernetes objects.

For this KEP, we propose taking this to what is essentially the logical conclusion, where
we enforce most of the limits across all devices, mixins and counter sets in a ResourceSlice,
rather than setting separate limits for each of them.

The ResourceSlice-wide limits will be:
* Total number of devices is 128.
* Total combined number of attributes and capacity in a ResourceSlice is 4096 (so with the maximum number of devices, there can be 32 per device).
* Total number of counters is 256.
* Total number of consumed counters is 2048 (so with the maximum number of devices, there can be 16 per device).
* Total number of counter sets is 32.
* Total number of device mixins is 128 (same as max number of devices).
* Total number of counter set mixins is 32 (same as max number of counter sets).
* Total number of device counter consumption mixins is 128.


We will still enforce some per-field limits:
* The number of mixins that can be referenced from each device or counter set is 8.
* The number of mixins that can be referenced from each device counter consumption is 4.
* The number of taints per device is 4.
* The number of device counter consumptions per device is 4. This means that each device can only consume counters from a maximum of 4 different counter sets.


We will also enforce one limit on the flattened device:
* The combined number of attributes and capacities for a single device can not exceed 32. We do this
  to avoid increasing the cost of evaluation the CEL expressions for a device.

The limits on the number of counters across counter sets, mixins and device counter consumption in 1.33 for the
Partitionable Devices KEP will be removed, as those are still in alpha.
The current limit on the total number of attributes and capacities per device will be adjusted a bit to
be enforced on the device with mixins applied, rather than just based on what is defined directly on a device. This
doesn't change the current behavior since a device with more than 32 attributes/capacities defined directly
on the device will always fail the updated validation rule.

With these limits, the worst-case size for a ResourceSlice increases from 1,107,864 bytes to 1,288,825 bytes.

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

None

##### Unit tests

<!--
Generated with:

go test -cover ./pkg/apis/resource/validation  ./staging/src/k8s.io/dynamic-resource-allocation/structured | sed -e 's/.*\(k8s.io[a-z/-]*\).*coverage: \(.*\) of statements/- `\1`: \2/' | sort

-->

- `k8s.io/dynamic-resource-allocation/structured`: `04/11/2025` - 91.3%
- `k8s.io/kubernetes/pkg/apis/resource/validation`: `04/11/2025` - 97.8%

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

The integration test that verifies the theoretical maximum size of the ResourceSlice
resource will be updated.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

E2e tests will be added to verify that the mixins are properly applied and
used by the scheduler.

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

Mixins will no longer work when downgrading to a release without support for it.
As described in the [Implementation](#implementation) section, devices that uses
mixins or reference counter sets that is using mixins, will not be visible to the
scheduler.

### Version Skew Strategy

During version skew where the apiserver supports the feature and the scheduler
doesn't, the devices that is using the mixins feature will be dropped and
not visible to the scheduler (ref [Implementation](#implementation)).

The exception here is in 1.34 (the first version where this feature is in alpha).
If the APIServer is at 1.34 and the scheduler is at 1.33, the APIServer will send
the new fields, but the scheduler will not know what to do about them. It will end
up ignoring them, which can lead to incorrect scheduling decisions. Note that this
scenario only applies to the initial 1.34 release and will not apply for 1.35+.
The recommendation is that the user should not enable this alpha feature unless
the scheduler is updated to 1.34 and enables the alpha feature as well.

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

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DRAResourceSliceMixins
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler


###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Applications that were already running will continue to run and the allocated
devices will remain so.

###### What happens if we reenable the feature if it was previously rolled back?

It will take affect again and will impact allocation decisions.

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

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes and no. It does add additional fields, which increases the worst case
size of the ResourceSlice object. However, it also provides features that
allows drivers to represent devices and counter sets in a more compact way,
thereby potentially reducing the size of the ResourceSlice object.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Flattening the devices and counter sets will require slightly more work, but
this is unlikely to have any meaningful impact on the time used for allocation.

It does allow DRA driver authors to create more complex devices, with a larger
number of counters. It also allows for larger number of counters in the counter
sets. This can worsen the worst-case scheduling performance.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

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

- 1.33: first KEP revision as part of the Partitionable Devices KEP
- 1.34: split out into a separate KEP

## Drawbacks

Using mixins adds to the complexity and makes it harder to get a quick
overview of a device or a counter set.

## Alternatives

Several alternatives were considered as part of the
[Partitionable Devices KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-scheduling/4815-dra-partitionable-devices/README.md#alternatives)

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->