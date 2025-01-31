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
# KEP-5051: Server Side Apply: Unsetting fields

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Apply (force=true) to unset an owned listType=map element](#apply-forcetrue-to-unset-an-owned-listtypemap-element)
    - [Apply (force=true) to unset an owned granular map value](#apply-forcetrue-to-unset-an-owned-granular-map-value)
    - [Apply (force=false) to an owned field](#apply-forcefalse-to-an-owned-field)
    - [Apply (force=false) to a unset field that is owned](#apply-forcefalse-to-a-unset-field-that-is-owned)
    - [Apply (force=false) to unset a field that is owned but already unset](#apply-forcefalse-to-unset-a-field-that-is-owned-but-already-unset)
    - [Apply to a field with a default value](#apply-to-a-field-with-a-default-value)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Defaulting considerations](#defaulting-considerations)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Risk: The implementation of this KEP breaks existing apply logic](#risk-the-implementation-of-this-kep-breaks-existing-apply-logic)
  - [Risk: The additional processing of apply requests negatively impacts performance](#risk-the-additional-processing-of-apply-requests-negatively-impacts-performance)
- [Design Details](#design-details)
    - [Admission Control](#admission-control)
  - [Type-safe apply configuration bindings](#type-safe-apply-configuration-bindings)
  - [Unset marker escaping](#unset-marker-escaping)
  - [High level implementation plan](#high-level-implementation-plan)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

This KEP proposes an improvement to Server Side Apply to allow apply configurations to declare the
intent that a field should not be set.

## Motivation

As of Kubernetes 1.32, Server side apply lacks a direct way to unset fields.

If a field manager exclusively owns a field, the field manage may unset the field by applying a
configuring that omits the field. But if the field manager does not exclusively own the field, there
is no way for the field manager to unset the field using an apply request.

Additionally, there is no way for a field manager to own a field that is unset.  This prevents a
field manager from expressing the opinion that a field should be unset, and to force a conflict if
another field manager attempts to set the field.

### Goals

- Introduce a way to express that a field should be unset in a Server Side Apply configuration.
- Introduce field ownership of unset fields into the managedFields of Server Side Apply.

### Non-Goals

_none_

## Proposal

Introduce a "marker" value that may be used in apply configurations to indicate that a value should
be unset.

The proposed marker value is: `{k8s_io__value: unset}`

For example, given an object with a field owned by field manager "mgr1":

```yaml
apiVersion: stable.example.com/v1
kind: Example
metadata:
  name: example1
  managedFields:
  - manager: mgr1
    operation: Apply
    …
    fieldsV1:
      f:spec:
        f:field {}
spec:
  field: "xyz"
```

Field manager "mgr2" may unset the field by force applying a apply configuration like:

```yaml
apiVersion: stable.example.com/v1
kind: Example
metadata:
  name: example1
spec:
  field: {k8s_io__value: unset}
```

After the configuration is applied field will be unset and will be owned by "mgr2".

```yaml
apiVersion: stable.example.com/v1
kind: Example
metadata:
  name: example1
  managedFields:
  - manager: mgr2
    operation: Apply
    …
    fieldsV1:
      f:spec:
        f:field {} # mgr2 owns this field even though it is unset
spec:
  # empty
```

### User Stories

#### Apply (force=true) to unset an owned listType=map element

Existing value:

```yaml
field: [
  {name: "a", value: 1}, 
  {name: "b", value: 2}
]
```

Existing field management:

```yaml
fieldManager1: 
  spec.field[name=a]
  spec.field[name=b]
```

Force apply configuration:

```yaml
field: [
  {name: "b", k8s_io__value: unset}
]
```

Result value:

```yaml
field: [
  {name: "a", value: 1}
]
```

Result field management:

```yaml
fieldManager1: 
  spec.field[name=a]

fieldManager2: 
  spec.field[name=b]
```

#### Apply (force=true) to unset an owned granular map value

Existing value:

```yaml
field: {"a": 1, "b": 2}
```

Existing field management:

```yaml
fieldManager1: 
  spec.field
```

Force apply configuration:

```yaml
field: {"b": {k8s_io__value: unset}}
```

Result value:

```yaml
field: {"a": 1}
```

Result field management:

```yaml
fieldManager2:
  field
```

#### Apply (force=false) to an owned field

Existing value:

```yaml
spec:
  field: 1
```

Existing field management:

```yaml
fieldManager1: 
  spec.field
```

Apply configuration:

```yaml
spec:
  field: {k8s_io__value: unset}
```

Apply conflicts:

```yaml
field owned by fieldManager1
```

#### Apply (force=false) to a unset field that is owned

Existing value:

```yaml
spec:
  # empty
```

Existing field management:

```yaml
fieldManager1: 
  spec.field
```

Apply configuration:

```yaml
spec:
  field: "xyz"
```

Apply conflicts:

```yaml
field owned by fieldManager1
```

#### Apply (force=false) to unset a field that is owned but already unset

Existing value:

```yaml
spec:
```

Existing field management:

```yaml
fieldManager1: 
  spec.field
```

Apply configuration:

```yaml
spec:
  field: {k8s_io__value: unset}
```

Result value:

```yaml
spec:
```

Result field management:

```yaml
Shared ownership.

fieldManager1: 
  spec.field

fieldManager2: 
  spec.field
```

#### Apply to a field with a default value

Existing value:

```yaml
spec:
  field: defaultValue
```

Existing field management:

```yaml
fieldManager1:
  # no owned fields
```

Force apply configuration:

```yaml
spec:
  field: {k8s_io__value: unset}
```

Result value:

```yaml
spec:
  field: defaultValue
```

Result field management:

```yaml
fieldManager2:
  spec.field
```

### Notes/Constraints/Caveats (Optional)

#### Defaulting considerations

1. A field that is not explicitly set, but is defaulted, is unowned (existing behavior)
1. If a field is unset using a marker, defaulting still applies
   - Consequence: A field manager that owns a unset value ends up owning the defaulted value
   - Consequence: If two field managers unset the same value, they share ownership of the field
   - Caveat: Non-declarative defaulting, such as defaulting that is performed by the strategy, or
    admission control, is not detectable by server side apply, and will result in conflicts between
	field managers even for cases where the defaulting SHOULD result in shared field ownership.
	This is a pre-existing problem between defaulting and server side apply but is more likely
	to occur with this enhancement.
1. listType=map key fields that are defaulted MUST be respected when extracting unset value markers from
   apply configurations.  That is, a unset marker such as `{keyField1: "x", k8s_io__value: unset}`
   will be treated as `{keyField1: "x", defaultedKeyField: "defaultValue", k8s_io__value: unset}`
   to ensure that the field paths are tracked correctly for field management purposes.

### Risks and Mitigations

#### Risk: The implementation of this KEP breaks existing apply logic

Mitigations: 

- Extensive testing. See Test Plan section.
- The new functionality will be isolated and gated to ensure the pre-existing behavior isolated
  preserved when the feature is off.

### Risk: The additional processing of apply requests negatively impacts performance

Mitigations:

- This change will be benchmarked.
- Apply requests that do not unset fields will only require the read-only check to search for unset
  field markers. All other new code will be skipped if no unset field markers are found.
- This change will leverage the optimizations used in structured-merge-diff to minimize allocations
  when traversing managedFields.

## Design Details

#### Admission Control

MutatingAdmissionPolicy will support the use of this feature.

For example, an ApplyConfiguration mutation in a MutatingAdmissionPolicy may unset a field:

```cel
Object{
  spec: Object.spec{
    template: Object.spec.template{
      spec: Object.spec.template.spec{
        volumes: [Object.spec.template.spec.volumes{
          name: "y",
          k8s_io__value: "unset",
        }]
      }
    }
}
```

This mutation unsets a field from the request.

This mutation **DOES NOT** result in the `managedFields` tracking ownership of the unset field.

At the time of KEP creation, MutatingAdmissionPolicy was an Alpha API; you can use a MutatingAdmissionPolicy to unset fields unconditionally (even
if the ServerSideApplyUnsettingFields feature gate is off). MutatingAdmissionPolicy will not
graduate to Beta before the this KEP and the `ServerSideApplyUnsettingFields` feature gate; we are phasing things
that way to ensure we do not limit our ability
to respond to community feedback for this enhancement.

### Type-safe apply configuration bindings

Before graduation to beta, we will add unsetting field support to `applyconfiguration-gen`.

Our goal is to add `Unset<FieldName>()` functions to the generated types. This may require
the introduction of custom marshalling to the apply configuration types.

### Unset marker escaping

We will **NOT** support escaping of the marker.

We considered this and started to prototype an implementation, but:

- Adding a performant unescaping pass to structured-merge-diff is complex. It effectively doubles
  the implementation effort of this KEP.
- It's hard to imagine where escaping this key would actually be needed. If we really need the ability
  to "apply to an apply configuration" in the future, look into options, but building this without
  a plausible use case does not seem necessary.

### High level implementation plan

In kubernetes-sigs/structured-merge-diff:

- An option will be added to apply configuration validation to allow unset field markers to pass
  validation.
- A new `TypedValue.ExtractMarkers()` function will be added that separates the markers from an
  apply configuration and returns them as a field set.
- An option will be added to `Updater.Apply` to allow unset fields.

In kubernetes/kubernetes:

- A feature gate will be added and used to enable support for unset field markers on
  `Updater.Apply`.
- MutatingAdmissionPolicy (Alpha feature) will be modified to always allow unset field markers.
  The use of unset field markers will **NOT** be feature gated in this alpha feature.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

- `kubernetes-sigs/structured-merge-diff/v4/typed`:
  - scalar (numeric, string, bool), list (atomic/set/map) and map (atomic/granular), and struct
    (atomic/granular) and null types
  - key formats (single key, multiple key, keys fields with defaults)
  - force=true and force=false apply requests (applying where conflicts are expected, applying where
    conflicts should not happen, applying where field ownership becomes shared)
  - applying to already managed fields (both fields that are owned and set and that are owned but
    unset), applying to unmanaged fields
  - applying to defaulted fields
  - a mix of apply and update requests from the same field manager
  - applying after another manage has used create, update, and all forms of patch requests
  - Invalid values of the unset marker key will be detected and produce clear error messages.
  - The unset marker key/value may only have other key fields as sibling fields, if any other map
    entries are present apply will fail with a clear error message.

##### Integration tests

Testing in kubernetes/kubernetes will include:

- Feature gate enablement/disablement
- Unset field markers are detected and used
- Unset fields are tracked correctly in managedFields across multiple requests
- applyconfiguration-gen generates typesafe bindings to unset fields that work as expected

##### e2e tests

TODO: For beta

- <test>: <link to test coverage>

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- All new functionality has been fully implemented and tested in kubernetes-sigs/structured-merge-diff
- All integration test shave been implemented
- MutatingAdmissionPolicy supports unsetting fields via ApplyConfiguration mutations

#### Beta

- applyconfiguration-gen generates typesafe bindings for unsetting fields
- We decide if the marker values should be allowed in CREATE/UPDATE manifests, and stripped out, or
  if we will not allow marker values in manifests.
- Kubectl allows `{k8s_io__value: unset}` when validating apply configurations.
- e2e tests are completed

#### GA

- 3 examples of real-world usage in the community
- Allow sufficient time for feedback, gather and respond to all actionable feedback

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

### Upgrade / Downgrade Strategy

This feature can be upgraded and downgraded without taking any special action because the new behavior
is limited to how requests are processed.

### Version Skew Strategy

Clients may only send requests with unset field markers to apiservers that support the feature.

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
  - Feature gate name: ServerSideApplyUnsettingFields
  - Components depending on the feature gate: kube-apiserver

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

Yes.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The only possible problem scenario is:
- A client unsets a field when this feature is enabled.
- The field ownership of the unset field is then tracked in managedFields.
- The ownership of the field conflicts with other expectations of the workload.

I believe this is acceptable because:
- The handling of managedFields is not changed by this enhancement, so the rollback does not
  make anything worse than it was before rollback
- the ownership of the field can be removed if needed (before or after rollback)

###### What specific metrics should inform a rollback?

- apiserver request failures

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be tested before graduation to beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

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

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

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

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

We considered requiring users modify the `managed.managedFields` data to add the fields that are unset
but that they wish to own. While this is possible, it goes against a goal established at the beginning
of the server side apply project to make all operations possible without client modifiecations to
`managed.managedFields`. We have documented that modifying `managed.managedFields` is discouraged and risky, 
and they were not designed to be modified by clients. Note that we may ALLOW users to include managedFields
in server side apply requests to indicate ownership of an unset field (and typedsafe generated bindings might
use this the type-incongruent marker values may be problematic), but we're not going to ever require users
modify `managed.managedFields` directly.

We considered using `null` (or other zero values).  But `null` and zero-values already may be used in apply
configurations, and do not indicate the intent to own a unset field. Modifying this semantic would be
breaking to clients that use `null` or zero-values today.

We considered using a different marker symbol to represent a unset field. We first considered a simple
string value such as "__UNSET__".  This can be made to work for unsetting fields of objects and values
of maps, but in order to be able to unset entries in keyed lists (`listType=map`) we need to be able to
both identify the entry to be unset by the entries keys, and then also indicate that the entry should be
unset.  So if we use "__UNSET__" for fields, then we need to introduce a special field name for keyed lists,
for example `__VALUE__: __UNSET__`. This observation led us to favor using a key/value representation
for all markers, which is slightly more verbose in the case of unsetting fields, but only requires
developers learn a single representation for the marker to be able to use it in for all possible cases.