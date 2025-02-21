<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [x] **Create a PR for this KEP.**
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
# [KEP-4816](https://github.com/kubernetes/enhancements/issues/4816): DRA: Prioritized Alternatives in Device Requests

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
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
    - [Resource Quota](#resource-quota)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Scheduler Implementation](#scheduler-implementation)
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
  - [Higher Level Indirection](#higher-level-indirection)
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


The [DRA Structured
Parameters](https://git.k8s.io/enhancements/keps/sig-node/4381-dra-structured-parameters)
feature has added the ability to make requests for very specific types of
devices using a `ResourceClaim`. However, the current API does not allow the
user to indicate any priority when multiple types or configurations of devices
may meet the needs of the workload. This feature allows the user to specify
alternative requests that statisfy the workloads need, giving the scheduler more
flexiblity in scheduling the workload.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

"Obtainability" of certain types of scarce resources is a primary concern of
many AI/ML users. GPUs are in high demand, particularly the latest models. This
means that workloads that use DRA to specify a need for particular types of GPUs
may fail to schedule. In practice, a workload that needs a GPU can be written
such that it can discover the GPUs available to it, and work with what it is
given. A user may have a preference for the latest model, but would like to run
the workload even if only an older model is available.

Similarly, packaged workload authors may wish to configure a workload such that
it will work well in the widest selection of available clusters. That is, a
distributor of shared workload definitions would like to be able to specify
alternative types of devices with which their workload will function, without
requiring the user to modify the manifests.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

* Allow workload authors, when specifying a `ResourceClaim`, to provide a list
  of ways to satisfy the claim, with a preference ranking.
* Enable schedulers to evaluate those preferences and allocate devices for the
  claim based on them.
* Enable cluster autoscalers to evaluate those preferences and make scaling
  choices based on them.
* Provide some measure of ResourceQuota controls when users utilize claims with
  these types of requests.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

* Enable cross-claim consistency of request choices. For example, guaranteeing
  that all `ResourceClaim`s associated with a given `Deployment` are satisfied
  using the same choice from the list of possible alternatives.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

The `ResourceClaim` object contains a `DeviceClaim`, which in turn contains a
list of `DeviceRequest` objects. This allows the user to allocate different
types of devices for the same claim, and apply constraints and configuration
across those different requests. These changes allow some flexibility for the
user to create, say, a "gpu" request, but allow it to be satisfied by one of
several models of GPU.

To avoid introducing a breaking API change and to ensure round-tripping
between the APIs, the `v1beta1` API will remain backward compatible 
(only extended) and the final change will be made in `v1beta2`.

The `v1beta1` API will be extended with a new field called `FirstAvailable`
which will contain an ordered list of `DeviceSubRequest` objects. In order to
satisfy the main (containing) request, exactly one of the requests listed in
`FirstAvailable` must be satisfied. The order listed is considered a priority
order, such that the scheduler will only try to use the second item in the list
if it is unable to satisfy the first item, and so on. This extension to the API
is not breaking, but it makes for a somewhat awkward API where many fields on
the `DeviceRequest` (in fact all existing fields except `Name`) must not be
set whenever `FirstAvailable` is set. To avoid this API in the long term, we are
adding the `v1beta2` API described next.

For the `v1beta2` API, the `DeviceRequest` object will be restructured
so that there will be the `Name` field just like today, and for supporting
either a single request or a prioritized list of subrequests, there will
be separate fields:

* `Exactly` for a request of type `SpecificDeviceRequest` without alternatives.
`SpecificDeviceRequest` will include all the fields that exists on the
`DeviceRequest` type in `v1beta1`, except the `Name` field.
* `FirstAvailable` for providing a prioritized list of requests, each of
type `DeviceSubRequest`. The `DeviceSubRequest` type is similar to
`SpecificDeviceRequest`, except for the `AdminAccess` field that is not
available when providing multiple alternatives. The list provided in the
`FirstAvailable` field is considered a priority order, such that the
scheduler will use the first entry in the list that satisfies the
requirements. 

DRA does not yet implement scoring, which means that
the selected devices might not be optimal. For example, if a prioritized
list is provided in a request, DRA might choose entry number two on node A,
even though entry number one would meet the requirements on node B. This is
consistent with current behavior in DRA where the first match will be chosen.
Scoring is something that will be implemented later, with early discussions
in https://github.com/kubernetes/enhancements/issues/4970

### User Stories

#### Story 1

As a workload author, I want to run a workload that needs a GPU. The workoad
itself can work with a few different models of GPU, but may need different
numbers of them depending on the model chosen. If the latest model is available
in my cluster, I would like to use that, but if it is not I am willing to take
a model one generation older. If none of those are available, I am willing to
take two GPUs of an even older model.

#### Story 2

As a workload author, I want to distribute the manifests of my workloads online.
However, there are many different models of device out there, and so I do not
want to be too prescriptive in how I define my manifest. If I make it too
detailed, then I will either need multiple versions or the users will have to
edit the manifest. Instead, I would like to provide some optionality in the
types of devices that can meet my workload's needs. For best performance though,
I do have a preferred ordering of devices.

### Notes/Constraints/Caveats

#### Resource Quota

ResourceQuota will be enforced such that the user must have quota for each
`DeviceSubRequest` under every `FirstAvailable`. Thus, this "pick one" behavior
cannot be used to circumvent quota. This reduces the usefulness of the feature,
as it means it will not serve as a quota management feature. However, the
primary goal of the feature is about flexibility across clusters and
obtainability of underlying devices, not quota management.


### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

For the `v1beta2` API, all fields in the current `DeviceRequest` field except
`Name` will be moved into a new `SpecificDeviceRequest` type and the `DeviceRequest`
type will get a field called `Exactly` using this type. So for a request that doesn't
provide a prioritized list of alternatives, the user will provide the `Name` and
then set the `Exactly` field which contains the details about the request.

The `FirstAvailable` field is mutually exclusive with the `Exactly` field, and
allow users to specify a prioritized list of `DeviceSubRequest` objects. The
`DeviceSubRequest` type is similar to the `SpecificDeviceRequest`, but does not
expose the `AdminAccess` field. These will be kept as separate types to avoid having
fields that can't be used in certain contexts (like `AdminAccess` for entries in the
`FirstAvailable` list) and to allow these to evolve separately going forward.

```go
// DeviceRequest is a request for devices required for a claim.
// This is typically a request for a single resource like a device, but can
// also ask for several identical devices. With FirstAvailable it is also
// possible to provide a prioritized list of requests.
type DeviceRequest struct {
    // Name can be used to reference this request in a pod.spec.containers[].resources.claims
    // entry and in a constraint of the claim.
    //
    // References using the name in the DeviceRequest will uniquely
    // identify a request when the Exactly field is set. When the
    // FirstAvailable field is set, a reference to the name of the
    // DeviceRequest will match whatever subrequest is chosen by the
    // scheduler.
    //
    // Must be a DNS label.
    //
    // +required
    Name string

    // Exactly specifies the details for a single request that must
    // be met exactly for the request to be satisfied.
    //
    // One of Exactly or FirstAvailable must be set.
    //
    // +optional
    // +oneOf=deviceRequestType
    Exactly *SpecificDeviceRequest

    // FirstAvailable contains subrequests, of which exactly one will be 
    // satisfied by the scheduler to satisfy this request. It tries to
    // satisfy them in the order in which they are listed here. So if
    // there are two entries in the list, the schduler will only check
    // the second one if it determines that the first one can not be used.
    //
    // DRA does not yet implement scoring, so the scheduler will
    // select the first set of devices that satisfies all the
    // requests in the claim. And if the requirements can
    // be satisfied on more than one node, other scheduling features
    // will determine which node is chosen. This means that the set of
    // devices allocated to a claim might not be the optimal set
    // available to the cluster. Scoring will be implemented later.
    //
    // +optional
    // +oneOf=deviceRequestType
    // +listType=atomic
    // +featureGate=DRAPrioritizedList
    FirstAvailable []DeviceSubRequest
}

// SpecificDeviceRequest is a request for one or more identical devices.
type SpecificDeviceRequest struct {
    // DeviceClassName references a specific DeviceClass, which can define
    // additional configuration and selectors to be inherited by this
    // request.
    //
    // A DeviceClassName is required. Clients must check that it is
    // indeed set. It's absence indicates that something changed in a way that
    // is not supported by the client yet, in which case it must refuse to
    // handle the request.
    //
    // Administrators may use this to restrict which devices may get
    // requested by only installing classes with selectors for permitted
    // devices. If users are free to request anything without restrictions,
    // then administrators can create an empty DeviceClass for users
    // to reference.
    //
    // +required
    DeviceClassName string

    // Selectors define criteria which must be satisfied by a specific
    // device in order for that device to be considered for this
    // request. All selectors must be satisfied for a device to be
    // considered.
    //
    // +optional
    // +listType=atomic
    Selectors []DeviceSelector

    // AllocationMode and its related fields define how devices are allocated
    // to satisfy this request. Supported values are:
    //
    // - ExactCount: This request is for a specific number of devices.
    //   This is the default. The exact number is provided in the
    //   count field.
    //
    // - All: This request is for all of the matching devices in a pool.
    //   Allocation will fail if some devices are already allocated,
    //   unless adminAccess is requested.
    //
    // If AlloctionMode is not specified, the default mode is ExactCount. If
    // the mode is ExactCount and count is not specified, the default count is
    // one. Any other requests must specify this field.
    //
    // More modes may get added in the future. Clients must refuse to handle
    // requests with unknown modes.
    //
    // +optional
    AllocationMode DeviceAllocationMode

    // Count is used only when the count mode is "ExactCount". Must be greater than zero.
    // If AllocationMode is ExactCount and this field is not specified, the default is one.
    //
    // +optional
    // +oneOf=AllocationMode
    Count int64

    // AdminAccess indicates that this is a claim for administrative access
    // to the device(s). Claims with AdminAccess are expected to be used for
    // monitoring or other management services for a device.  They ignore
    // all ordinary claims to the device with respect to access modes and
    // any resource allocations.
    //
    // This is an alpha field and requires enabling the DRAAdminAccess
    // feature gate. Admin access is disabled if this field is unset or
    // set to false, otherwise it is enabled.
    //
    // +optional
    // +featureGate=DRAAdminAccess
    AdminAccess *bool
}

// DeviceSubRequest describes a request for device provided in the
// claim.spec.devices.requests[].firstAvailable array. Each
// is typically a request for a single resource like a device, but can
// also ask for several identical devices.
//
// DeviceSubRequest is similar to SpecificDeviceRequest, but doesn't expose the
// AdminAccess field as that one is only supported when requesting a
// specific device.
type DeviceSubRequest struct {
    // Name can be used to reference this subrequest in the list of constraints
    // or the list of configurations for the claim. References must use the
    // format <main request>/<subrequest>.
    //
    // Must be a DNS label.
    //
    // +required
    Name string

    // DeviceClassName references a specific DeviceClass, which can define
    // additional configuration and selectors to be inherited by this
    // subrequest.
    //
    // A class is required. Which classes are available depends on the cluster.
    //
    // Administrators may use this to restrict which devices may get
    // requested by only installing classes with selectors for permitted
    // devices. If users are free to request anything without restrictions,
    // then administrators can create an empty DeviceClass for users
    // to reference.
    //
    // +required
    DeviceClassName string

    // Selectors define criteria which must be satisfied by a specific
    // device in order for that device to be considered for this
    // subrequest. All selectors must be satisfied for a device to be
    // considered.
    //
    // +optional
    // +listType=atomic
    Selectors []DeviceSelector

    // AllocationMode and its related fields define how devices are allocated
    // to satisfy this subrequest. Supported values are:
    //
    // - ExactCount: This request is for a specific number of devices.
    //   This is the default. The exact number is provided in the
    //   count field.
    //
    // - All: This subrequest is for all of the matching devices in a pool.
    //   Allocation will fail if some devices are already allocated,
    //   unless adminAccess is requested.
    //
    // If AlloctionMode is not specified, the default mode is ExactCount. If
    // the mode is ExactCount and count is not specified, the default count is
    // one. Any other subrequests must specify this field.
    //
    // More modes may get added in the future. Clients must refuse to handle
    // requests with unknown modes.
    //
    // +optional
    AllocationMode DeviceAllocationMode

    // Count is used only when the count mode is "ExactCount". Must be greater than zero.
    // If AllocationMode is ExactCount and this field is not specified, the default is one.
    //
    // +optional
    // +oneOf=AllocationMode
    Count int64
}

const (
    DeviceSelectorsMaxSize             = 32
    FirstAvailableDeviceRequestMaxSize = 8
)
```

For the `v1beta1` API:

A `DeviceRequest` that populates the `FirstAvailable` field must *not*
populate the `DeviceClassName` field. The `required` validation on this field
will be relaxed. This allows existing clients to differentiate between claims
they understand (with `DeviceClassName`) and those they do not (without
`DeviceClassName` but with the new field). Clients written for 1.31, when
`DeviceClassName` was required, were requested to include this logic, and the
in-tree components have been built in this way.

```go
// DeviceRequest is a request for devices required for a claim.
// This is typically a request for a single resource like a device, but can
// also ask for several identical devices.
type DeviceRequest struct {
    // Name can be used to reference this request in a pod.spec.containers[].resources.claims
    // entry and in a constraint of the claim.
    //
    // Must be a DNS label.
    //
    // +required
    Name string

    // DeviceClassName references a specific DeviceClass, which can define
    // additional configuration and selectors to be inherited by this
    // request.
    //
    // Either a class or FirstAvailable requests are required in DeviceClaim.Requests.
    // When this request is part of the FirstAvailable list, a class is required. Nested
    // FirstAvailable requests are not allowed.
    //
    // Which classes are available depends on the cluster.
    //
    // Administrators may use this to restrict which devices may get
    // requested by only installing classes with selectors for permitted
    // devices. If users are free to request anything without restrictions,
    // then administrators can create an empty DeviceClass for users
    // to reference.
    //
    // +optional
    // +oneOf=deviceRequestType
    DeviceClassName string

    // FirstAvailable contains subrequests, of which exactly one will be 
    // satisfied by the scheduler to satisfy this request. It tries to
    // satisfy them in the order in which they are listed here. So if
    // there are two entries in the list, the schduler will only check
    // the second one if it determines that the first one can not be used.
    //
    // This field may only be set in the entries of DeviceClaim.Requests.
    //
    // DRA does not yet implement scoring, so the scheduler will
    // select the first set of devices that satisfies all the
    // requests in the claim. And if the requirements can
    // be satisfied on more than one node, other scheduling features
    // will determine which node is chosen. This means that the set of
    // devices allocated to a claim might not be the optimal set
    // available to the cluster. Scoring will be implemented later.
    //
    // +optional
    // +oneOf=deviceRequestType
    // +listType=atomic
    // +featureGate=DRAPrioritizedList
    FirstAvailable []DeviceSubRequest

    ...
}

// DeviceSubRequest describes a request for device provided in the
// claim.spec.devices.requests[].firstAvailable array. Each
// is typically a request for a single resource like a device, but can
// also ask for several identical devices.
//
// DeviceSubRequest is similar to Request, but doesn't expose the AdminAccess (not
// supported) or FirstAvailable (recursion not supported) fields, as those can
// only be set on the top-level request.
type DeviceSubRequest struct {
    // Name can be used to reference this subrequest in the list of constraints
    // or the list of configurations for the claim. References must use the
    // format <main request>/<subrequest>.
    //
    // Must be a DNS label.
    //
    // +required
    Name string

    // DeviceClassName references a specific DeviceClass, which can define
    // additional configuration and selectors to be inherited by this
    // subrequest.
    //
    // A class is required. Which classes are available depends on the cluster.
    //
    // Administrators may use this to restrict which devices may get
    // requested by only installing classes with selectors for permitted
    // devices. If users are free to request anything without restrictions,
    // then administrators can create an empty DeviceClass for users
    // to reference.
    //
    // +required
    DeviceClassName string

    // Selectors define criteria which must be satisfied by a specific
    // device in order for that device to be considered for this
    // subrequest. All selectors must be satisfied for a device to be
    // considered.
    //
    // +optional
    // +listType=atomic
    Selectors []DeviceSelector

    // AllocationMode and its related fields define how devices are allocated
    // to satisfy this subrequest. Supported values are:
    //
    // - ExactCount: This request is for a specific number of devices.
    //   This is the default. The exact number is provided in the
    //   count field.
    //
    // - All: This subrequest is for all of the matching devices in a pool.
    //   Allocation will fail if some devices are already allocated,
    //   unless adminAccess is requested.
    //
    // If AlloctionMode is not specified, the default mode is ExactCount. If
    // the mode is ExactCount and count is not specified, the default count is
    // one. Any other subrequests must specify this field.
    //
    // More modes may get added in the future. Clients must refuse to handle
    // requests with unknown modes.
    //
    // +optional
    AllocationMode DeviceAllocationMode

    // Count is used only when the count mode is "ExactCount". Must be greater than zero.
    // If AllocationMode is ExactCount and this field is not specified, the default is one.
    //
    // +optional
    // +oneOf=AllocationMode
    Count int64
}

const (
    DeviceSelectorsMaxSize             = 32
    FirstAvailableDeviceRequestMaxSize = 8
)
```

Let's take a look at a simple example using a single request with the
`v1beta2` API:

```yaml
apiVersion: resource.k8s.io/v1beta2
kind: ResourceClaim
metadata:
  name: device-consumer-claim
spec:
  devices:
    requests:
    - name: gpu
      exactly:
        deviceClassName: big-gpu
```

Another example shows how to use `FirstAvailable` with the
`v1beta2` API:

```yaml
apiVersion: resource.k8s.io/v1beta2
kind: ResourceClaim
metadata:
  name: device-consumer-claim
spec:
  devices:
    requests:
    - name: nic
      exactly:
        deviceClassName: rdma-nic
    - name: gpu
      firstAvailable:
      - name: big-gpu
        deviceClassName: big-gpu
      - name: mid-gpu
        deviceClassName: mid-gpu
      - name: small-gpu
        deviceClassName: small-gpu
        count: 2
    constraints:
    - requests: ["nic", gpu"]
      matchAttribute:
      - dra.k8s.io/pcieRoot
    config:
    - requests: ["gpu/small-gpu"]
      opaque:
        driver: gpu.acme.example.com
        parameters:
          apiVersion: gpu.acme.example.com/v1
          kind: GPUConfig
          mode: multipleGPUs
```

There are a few things to note here. First, the "nic" request is listed with 
the `exactly` object, because it has no alternative request types. The "gpu"
request could be met by several different types of GPU, in the listed order of
preference as specified in the `firstAvailable` list. Each of those is a
separate `DeviceSubRequest`, with both a `deviceClassName` and also its own name.
The fact that these subrequests also have their own names allows us to apply 
constraints or configuration to specific, individual subrequests, in the event
that it is the chosen alternative. When referencing a subrequest, the name of the
main request must also be included in the form `<main request>/<subrequest>`.
This avoids having to enforce unique subrequest names between requests in a
claim. In this example, the "small-gpu" choice requires a configuration
option that the other two choices do not need. Thus, if the resolution of the
"gpu" request is made using the "small-gpu" subrequest, then that configuration
will be attached to the allocation. Otherwise, it will not.

Similarly, for `Constraints`, the list of requests can include the main request
name ("gpu" in this case), in which case the constraint applies regardless of
which alternative is chosen. Or, it can include the subrequest name, in which
case that constraint only applies if that particular subrequest is chosen.

In the PodSpec, however, the subrequest names are not valid. Only the main
request name may be used.


Finally, lets look at what a request using `FirstAvailable` looks like
with the v1beta1 API:

```yaml
apiVersion: resource.k8s.io/v1beta1
kind: ResourceClaim
metadata:
  name: device-consumer-claim
spec:
  devices:
    requests:
    - name: nic
      deviceClassName: rdma-nic
    - name: gpu
      firstAvailable:
      - name: big-gpu
        deviceClassName: big-gpu
      - name: mid-gpu
        deviceClassName: mid-gpu
      - name: small-gpu
        deviceClassName: small-gpu
        count: 2
    constraints:
    - requests: ["nic", gpu"]
      matchAttribute:
      - dra.k8s.io/pcieRoot
    config:
    - requests: ["gpu/small-gpu"]
      opaque:
        driver: gpu.acme.example.com
        parameters:
          apiVersion: gpu.acme.example.com/v1
          kind: GPUConfig
          mode: multipleGPUs
```

It is worth noting here that `DeviceClassName` is set for the
`nic` request that doesn't have alternatives, while that is only
specified for the subrequests when `FirstAvailable` is set.

All these ResourceClaims can be used from a Pod referencing
the claim.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: device-consumer
spec:
  resourceClaims:
  - name: "gpu-and-nic"
    resourceClaimName: device-consumer-claim
  containers:
  - name: workload
    image: my-app
    command: ["/bin/program"]
    resources:
      requests:
        memory: "64Mi"
        cpu: "250m"
      limits:
        memory: "128Mi"
        cpu: "500m"
      claims:
      - name: "gpu-and-nic"
        request: "gpu" # the 'nic' request is pod-level, no need to attach to container
```

For both the `v1beta1` and `v1beta2` APIs, the `status.allocationResult.devices.results[].request`
field will reference the name of the main request for requests without
alternatives that set the `Exactly` field on the request, while it will
reference the selected subrequest on the format `<main request>/<subrequest>`
for requests that provide alternatives by setting the `FirstAvailable` field.

### Scheduler Implementation

Currently, the scheduler loops through each entry in `DeviceClaim.Requests` and
tries to satisfy each one. This would work essentially the same, except that if
the `FirstAvailable` object is set, it will attempt all alternatives for a 
request until one is found or the list is exhausted.

The current implementation in the v1beta1 API uses a missing `DeviceClassName` to make sure that
an old version of the scheduler doesn't incorrectly handle a request that
sets `FirstAvailable` which it doesn't see. If this happens it will result in an
[error](https://github.com/kubernetes/kubernetes/blob/03f134461462f86239067ec20ec17a0ba892db52/staging/src/k8s.io/dynamic-resource-allocation/structured/allocator.go#L164).
For the v1beta2 API, this will be handled by the scheduler being able to
detect claims that have neither the `Exactly` nor `FirstAvailable` set on a request.

The current implementation will navigate a depth-first search of the devices,
trying to satisfy all requests and contraints of all claims. The optionality
offered at the `DeviceRequest` level provides another index state to track in
the
[`requestIndices`](https://github.com/kubernetes/kubernetes/blob/03f134461462f86239067ec20ec17a0ba892db52/staging/src/k8s.io/dynamic-resource-allocation/structured/allocator.go#L362) and [`deviceIndices`](https://github.com/kubernetes/kubernetes/blob/03f134461462f86239067ec20ec17a0ba892db52/staging/src/k8s.io/dynamic-resource-allocation/structured/allocator.go#L368). In the case of the feature gate
disabled, this new index will always be 0.

Alternatively, we can refactor to make this code more defensible via a feature
gate.

DRA today works on a "first match" basis for a given node. That would not change
with this KEP; on any given node, devices will be tried in the priority order
listed in the main request, and the first fit will be returned. However, in
practice, nodes typically only have one type of device that would satisfy any of
the three requests. That means that individual nodes with any of the listed
devices will show as valid nodes for the workload. In order for the scheduler to
prefer a node that has the initial prioritized device request, those requests
would need a higher score, which currently is planned for beta of this feature.
For alpha, the scheduler may still pick a node with a less preferred device, if
there are nodes with each type of device available.

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

<!--
Generated with:
go test -cover ./pkg/scheduler/framework/plugins/dynamicresources/... ./pkg/controller/resourceclaim ./pkg/kubelet/cm/dra/... ./staging/src/k8s.io/dynamic-resource-allocation/cel ./staging/src/k8s.io/dynamic-resource-allocation/structured | sed -e 's/.*\(k8s.io[a-z/-]*\).*coverage: \(.*\) of statements/- `\1`: \2/' | sort
-->

Start of v1.32 development cycle (v1.32.0-alpha.1-178-gd9c46d8ecb1):

- `k8s.io/dynamic-resource-allocation/cel`: 88.8%
- `k8s.io/dynamic-resource-allocation/structured`: 82.7%
- `k8s.io/kubernetes/pkg/controller/resourceclaim`: 70.0%
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources`: 72.9%

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

The existing [integration tests for kube-scheduler which measure
performance](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler_perf#readme)
will be extended to cover the overheaad of running the additional logic to
support the features in this KEP. These also serve as [correctness
tests](https://github.com/kubernetes/kubernetes/commit/cecebe8ea2feee856bc7a62f4c16711ee8a5f5d9)
as part of the normal Kubernetes "integration" jobs which cover [the dynamic
resource
controller](https://github.com/kubernetes/kubernetes/blob/294bde0079a0d56099cf8b8cf558e3ae7230de12/test/integration/scheduler_perf/util.go#L135-L139).

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

End-to-end testing depends on a working resource driver and a container runtime
with CDI support. A [test
driver](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/dra/test-driver)
was developed as part of the overall DRA development effort. We will extend this
test driver to enable support for alternative device requests and add tests to
ensure they are handled by the scheduler as described in this KEP.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Implemented in the scheduler but not necessarily the cluster auto scaler
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback
- Implement node scoring
- Evaluate feasibilty of cluster auto scaler implementation
- Additional tests are in Testgrid and linked in KEP

#### GA

- 3 examples of real-world usage
- Allowing time for feedback

### Upgrade / Downgrade Strategy

Standard upgrade/downgrade strategies may be used, no special configuration
changes are needed. There are no kubelet or DRA-driver changes for this feature,
they are all local to the control plane.

### Version Skew Strategy

The proposed API change relaxes a `required` constraint on the
`DeviceRequest.DeviceClassName` field. The `DeviceRequest` thus becomes a one-of
that must have either the `DeviceClassName` or the `FirstAvailable` field
populated.

Older clients have been advised in the current implementation to check this
field, even though it is required, and fail to allocate a claim that does not
have the field set. This means that during rollout, if the API server has this
feature, but the scheduler does not, the scheduler will fail to schedule pods
that utilize the feature. The pod will be scheduled later according to the new
functionality after the scheduler is upgraded.

This feature affects the specific allocations that get made by the scheduler.
Those allocations are stored in the `ResourceClaim` status, and will be acted
upon by the kubelet and DRA-driver just as if the user had made the request
without this feature. Thus, there is no impact on the data plane version skew;
if the selected request could be satisfied by the data plane without this
feature, it will work exactly the same with this feature.

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

This is an add-on on top of the `DynamicResourceAllocation` feature gate, which
also must be enabled for this feature to work.

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DRAPrioritizedList
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler
    - kube-controller-manager
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes. No existing claims or running pods will be affected. This feature affects
only the allocation of devices during scheduling.

If a workload controller or Pod uses a `ResourceClaimTemplate` that includes
this feature, it could happen that a new Pod may be created and need to be
scheduled, even though the feature is disabled. In this case, the new Pod will
fail to schedule, as the corresponding `ResourceClaim` will not be able to be
created.

The recommendation is to remove any usage of this feature in both
`ResourceClaim`s and `ResourceClaimTemplate`s when disabling the feature, and
force the workloads to use a specific device request instead. This will ensure
that there are no unexpected failures later, if a Pod gets rescheduled to
another node or recreated for some reason.

###### What happens if we reenable the feature if it was previously rolled back?

The feature will begin working again for future scheduling choices that make use
of it. For `Deployments` or other users of `ResourceClaimTemplate`, previously
failing Pod creations or scheduling may begin to succeed.

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

Unit tests will be written to validate the enablement and disablement behavior,
as well as type conversions for the new field and relaxed validation.

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

Will consider in the beta timeframe.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

Will consider in the beta timeframe.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Will consider in the beta timeframe.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No, though we do relax validation on one field to make it no longer a required
field.

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

Will consider in the beta timeframe.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

Will consider in the beta timeframe.

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

Existing DRA and related SLOs continue to apply.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

Will consider in the beta timeframe.

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

We can consider a histogram metric showing how many allocations are made from
indices 0-7 of ResourceClaims that utilize this feature.

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

This feature depends on the DRA structured parameters feature being enabled, and
on DRA drivers being deployed. There are no requirements beyond those already
needed for DRA structured parameters.

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

No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No, just a new field on the `ResourceClaim.DeviceRequest` struct.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

Yes, when using this field, the user will add additional data in their
`ResourceClaim` and `ResourceClaimTemplate` objects. This is an incremental
increase on top of the existing structures. The number of alternate requests is
limited to 8 in order to minimize the potential object size.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

Scheduling a claim that uses this feature may take a bit longer, if it is
necessary to go deeper into the list of alternative options before finding a
suitable device. We can measure this impact in alpha.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No.

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

1.32 Enhancements Freeze - KEP merged, alpha implementation initiated

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

This adds complexity to the scheduler and to the cluster autoscaler, which will
simulate the satisfaction of claims with different node shapes.


## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### Higher Level Indirection

Rather than embedding a list of alternative request objects, we could use an
indirection at either the `ResourceClaim` level, or the `DeviceClaim` level.
For example, we could create a new resource claim type by adding a
`FirstOfDevices` list to the `ResourceClaimSpec`, and making it a one-of with
`Devices`.

Something like this:

```go
// ResourceClaimSpec defines what is being requested in a ResourceClaim and how to configure it.
type ResourceClaimSpec struct {
        // Devices defines how to request devices.
        //
        // oneOf: claimType
        // +optional
        Devices DeviceClaim

        // FirstOfDevices defines devices to claim in a
        //
        // oneOf: claimType
        // +optional
        FirstOfDevices []DeviceClaim

        ...
}
```

This is arguably simpler and allows them to be essentially complete, alternate
claims. It would be more difficult for the user, though, as it would require
duplication of other device requests. Additionally, if there were multiple
separate `FirstAvailable` requests in a claim, the user would have to specify
all the combinations of those in order to get the same flexibility.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
