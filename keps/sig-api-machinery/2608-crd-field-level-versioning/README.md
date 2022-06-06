# KEP-2608: Support Field Level Versioning In CRDs.

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
    - [Scenario 1](#scenario-1)
    - [Scenario 2](#scenario-2)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Implementation](#implementation)
    - [The <code>generation</code> Field](#the--field)
    - [Server Side Warnings](#server-side-warnings)
  - [Additional Validation](#additional-validation)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (1.25)](#alpha-125)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Upgrade](#upgrade)
    - [Downgrade](#downgrade)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

This KEP proposes adding support for field-level versioning for Custom Resources (CRs).
When a new field is added to an existing, built-in API type, it is typically gated by
the addition of a feature gate in [`kube_features.go`](https://github.com/kubernetes/kubernetes/blob/master/pkg/features/kube_features.go).
This gate effectively helps control the lifecycle of the feature that is implemented as
a result of adding this additional field. Currently it is not possible to version CRs in
a similar manner. This KEP aims to provide a mechanism that allows feature-gate-like
functionality for Custom Resources as well.

## Motivation

Currently, it is not possible to have fields of a CR exist in different stages such
as alpha, beta, stable or deprecated, the CR itself can be in a version that is defined
by its Custom Resource Definition (CRD). If a field of a CR were to be marked as deprecated,
the author of the CRD would have to create a new version of the CRD, mark the current one as
deprecated, optionally issue a deprecation warning, introduce a new CRD version with the
deprecated field not present and add conversion logic to convert between the now deprecated
version and the new version. Please note that these steps are done at the level of an entire
version, if the deprecated field is not updated/touched after being created, users of this CRD
would still get deprecation warnings asking them to switch to a new version (provided it is made 
available).  

Field level versioning for CRs will introduce the ability to have individual fields of a
CR in different stages and aims to try and simplify the above mentioned process and provide
a UX similar to built-in types.

### Goals

- Introduce custom feature gates that can be embedded as part of the CRD spec.
- Define the lifecycle of these custom feature gates and specify what the behaviour
  of the gate would be at different stages in its lifecycle.
- Allow issuing custom deprecation warnings for deprecated fields.  
- Extend the CRD validation chain to include checks for proper usage of these
  custom feature gates.

### Non-Goals

- Provide a generic mechanism for dynamic feature gate functionality.

## Proposal

This KEP proposes to extend the CRD spec to include feature gate information
for fields of Custom Resources (CRs):

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: crontabs.stable.example.com
spec:
  group: stable.example.com
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                cronSpec:
                  type: string
                image:
                  type: string
                replicas:
                  type: integer
  scope: Namespaced
  names:
    ...
  customFeatureGates:
    component: istio # optional
    featureGates:
      - name: ReplicasFeatureGate
        enabled: true  # optional
        default: false # optional
        preRelease: alpha
        fieldPaths:
          - .spec.replicas
```
This KEP, on a high level, proposes the following:
- Multiple feature gates can be defined and each feature gate can then gate
  multiple `fieldPaths` - each of which is a JSON path for a field in a Custom
  Resource (CR) of this CRD.
- Each feature gate defines the state that it is in, namely `preRelease`.
  - The `preRelease` field can be in one of the following states:
    - `alpha`
    - `beta`
    - `stable`
    - `deprecated`
      - Featuer gates that have `preRelease` as `deprecated` can optionally
        specify a deprecation warning that the server will send out when this
        field is used.
- Additionally, a feature gate can also optionally specify if it is `enabled` or not
  as well as optionally specify what the `default` state should be for the gate if
  `enabled` is not specified.
- Feature gates are evaluated and "applied" only on the storage versions of CRs whose
  CRDs define feature gates, i.e. they are evaluated after conversion(s) to the storage
  version take place. This behaviour is consistent with that of built-in types.
- Please note that these feature gates are for *compatible* changes *within* an API version
  (more specifically, the storage version).
- The behaviour of these feature gates can be illustrated by the following example:
  - Consider a CR for the above mentioned CRD:
    ```yaml
    apiVersion: "stable.example.com/v1"
    kind: CronTab
    metadata:
      name: my-new-cron-object
    spec:
      cronSpec: "* * * * */5"
      image: my-awesome-cron-image
      replicas: 3
    ```
  - If this CR is `Create`d, and the CRD defines `ReplicasFeatureGate` that gates the
    path `.spec.replicas`, the behaviour can be summarised as follows:
    | Feature Gate state | Action on `.spec.replicas` |
    | ----------- | ------------ |
    | disabled    | Drop Field   |
    | enabled     | Proceed as usual |
  - If a CR such as the above one already exists, and it is now `Update`d by applying
    the following configuration:
    ```yaml
    apiVersion: "stable.example.com/v1"
    kind: CronTab
    metadata:
      name: my-new-cron-object
    spec:
      cronSpec: "* * * * */5"
      image: my-awesome-cron-image
      replicas: 5 # <--- updated
    ```
    Provided the  CRD defines `ReplicasFeatureGate` that gates the path `.spec.replicas`,
    the behaviour for an `Update` can be summarised as follows:
    | Existing persisted object has field (`.spec.replicas`) | Feature Gate state | Action on `.spec.replicas` |
    | -------- | ----------- | ------------ |
    | no       | disabled    | Drop Field   |
    | no       | enabled     | Set field    |
    | yes      | disabled    | Do nothing - don't drop and don't update the field |
    | yes      | enabled     | Update field |
  - For more details on the behaviour, please see [Design Details](#design-details).

### User Stories (Optional)

Please refer to the CRD defined in the previous section for the examples used in the user stories.

#### Story 1
*I am a cluster admin and I want to enable/disable a particular feature gate after the CRD is created.*

If the user is the CRD owner, they can make the change to the CRD manifest and `apply` the changed manifest file.

If the user is *not* the CRD owner, they can `patch` the CRD object to enable/disable a feature gate. For example:

```
kubectl patch customresourcedefinitions.apiextensions.k8s.io crontabs.stable.example.com \
        --type 'json' \
        --patch '[{"op": "replace", "path": "/spec/customFeatureGates/featureGates/0/enabled", "value": true}]'
``` 

#### Story 2
*I am a CRD author and I would like to add a new field to the schema defined by the storage version.*

Provided that the addition of the new field is a compatible API change as per the [API Changes guide](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md), a new field can be added and a corresponding gate can be introduced gating this field provided it does not violate any [constraints](#NotesConstraintsCaveats-Optional).

A field is typically introduced at the `alpha` stage, and the feature gate for this field can be either be explicitly enabled or implicitly disabled similar to feature gates for built-in types.

Once certain criteria are met ("criteria" defined by CRD authors), the field can be promoted to `beta` and then to `stable`. 

If the field is in the `stable` stage for some amount of time (determined by the CRD authors), the feature gate for this field can now be removed.


### Notes/Constraints/Caveats (Optional)

A potential constraint with the current approach is when two feature gates gate 
field paths that have a common path. For example:
```yaml
customFeatureGates:
  component: my-component # optional
  featureGates:
    - name: FeatureGate1
      enabled: true  # optional
      default: false # optional
      preRelease: alpha
      fieldPaths:
        - .spec.foo
    - name: FeatureGate2
      enabled: true  # optional
      default: false # optional
      preRelease: alpha
      fieldPaths:
        - .spec.foo.bar
```
In the above case, `FeatureGate2`'s state will only be considered iff `FeatureGate1` is 
in the enabled state. Please see the [Design Details](#design-details) section to see how
the enabled state of a feature gate is computed.  

The reason for doing things this way is the intuition that if a field itself is considered
"disabled" (the feature gate gating that field is in the disabled state), then any and all
fields that are sub-fields of this field would also be "disabled", which translates to: if 
a feature is disabled, then all features that depend on this feature should also then be 
considered as disabled.

### Risks and Mitigations

#### Scenario 1

*How do we ensure that feature gates that have been enabled by a user don't get clobbered by a definition update from the CRD owner?*

To mitigate against cases such as this, we recommend using [Server Side Apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/) (SSA) to prevent unintentional changes to fields that were previously created/modified by another entity (user, controller, etc).

#### Scenario 2

*I am a controller author but I don't nescessarily own the CRD, do I need to know anything about feature gates defined as part of the CRD?*

As a controller author, it is helpful to be aware of the fact that a feature gate can be disabled on a particular field. Because of this, the following situation may arise:

- Assume the controller tries to reconcile field `f` to value `v` and `f` has a feature gate defined on it called `g`
- Initially, `g` was enabled and the field `f` could be modified by the controller. If now `g` is disabled while the field `f` had a value `v'`, where `v != v'`, updates to `f` will not take effect (please see the Implementation section for more details).
- If no updates are done to the object, the controller can get a false sense that the field `f` is reconciled, when in fact it is not.
- If, now a non-gated field `f'` is updated, the controller gets an event (assuming it is using the usual watch and informer mechanism).
- The controller sees that `f` has value `v'`, it tries to reconcile it to value `v`, but the update has no effect as `g` is disabled.

To mitigate this scenario, we recommend making use of the `generation` field in the object's `metadata` to check if an update was actually made or not. Please see the Implementation section for more details on this.

Subsequently, if you are using a client such as `kubectl` in the above scenario, if an update does not actually happen, an additional server side warning goes out conveying this and can be displayed back to the user.

## Design Details

### API Changes
The API changes include extending the CRD spec by adding an optional field:
```go
// CustomResourceDefinitionSpec describes how a user wants their resource to appear
type CustomResourceDefinitionSpec struct {
    ...
    // customFeatureGates defines feature gates for fields of
    // custom resources of this CRD.
    // +optional
    CustomFeatureGates *CustomResourceDefinitionFeatureGates
}
```
Additionally, a few types are introduced:
```go
// CustomResourceDefinitionFeatureGates defines feature gates for specific
// fields to enable field level versioning for custom resource definitions.
type CustomResourceDefinitionFeatureGates struct {
    // featureGates is a list of gates defined for fields of a custom
    // resource definition. No two feature gates may guard the same field.
    FeatureGates []CustomResourceDefinitionFeatureGate
    // component is meant to indicate the entity that is responsible for
    // the creation of these feature gates when the feature gate information
    // is published to OpenAPI.
    // +optional
    Component *string
}
```
```go
// CustomResourceDefinitionFeatureGate describes the information
// conveyed by a feature gate.
type CustomResourceDefinitionFeatureGate struct {
    // name is the name of the feature gate being defined.
    Name string
    // enabled signifies whether the feature gatea is enabled or not.
    // +optional
    Enabled *bool
    // default is the default enablement state for the feature.
    // +optional
    Default *bool
    // preRelease indicates the stage of development that the
    // field/fields it is guarding is in. Acceptable values are
    // alpha, beta, stable, deprecated.
    PreRelease string
    // fieldDeprecationWarning overrides the default warning returned
    // to API clients when a field marked as deprecated is used. This
    // may only be set when `deprecated` is true. The default warning
    // indicates the field path of this field along with messaging that
    // it is deprecated.
    // +optional
    FieldDeprecationWarning *string
    // fieldPaths defines the fields in JSON path format that this
    // feature gate helps gate.
    FieldPaths []string
}
```
A feature gate can be "enabled" or "disabled". However, as seen from
`CustomResourceDefinitionFeatureGate`, the fields `enabled` and `default`
are _optional_ but the `preRelease` field is _required_. Based on the
values of these fields, the state of the feature gate can be determined,
namely whether it is enabled or not using the following set of rules:
1. If the `preRelease` of the feature gate is `stable`, then it is enabled.
2. If the feature gate has an `enabled` value specified, then this will
   determine whether the feature gate is enabled or not.
3. If the feature gate does not have an `enabled` value specified but has
   a `default` value specified, then this will determine whether the gate
   is enabled or not.
4. If the `preRelease` of the feature gate is `beta`, and neither of `enabled`
   nor `default` values are specified, then the feature gate is enabled.
6. If neither `enabled` nor `default` values are specified, the enabled state
   of the feature gate takes after the default values of the `alpha` `preRelease`
   stage, which is of state disabled.  

```go
func isFeatureGateEnabled(featureGate apiextensionsv1.CustomResourceDefinitionFeatureGate) bool {
    switch {
    case featureGate.PreRelease == "stable":
        return true
    case featureGate.Enabled != nil:
        return *featureGate.Enabled
    case featureGate.Default != nil:
        return *featureGate.Default
    case featureGate.PreRelease == "beta":
        return true
    }

    return false
}
```
### Implementation
Feature gates are laregely evaluated for a CR as part of the `strategy`, specifically 
the `PrepareForCreate()` and `PrepareForUpdate()` stages, this also means that feature 
gates are evaluated on CRs that have already been converted to their storage versions. 
Due to this, the feature gates specified in the CRD spec are implicitly specified for 
the storage version of the CRD.  

Additionally, as specified in the constraints section, one possible constraint is that
of feature gates gating field paths that may have common paths. To recall:
```yaml
customFeatureGates:
  component: my-component # optional
  featureGates:
    - name: FeatureGate1
      enabled: true  # optional
      default: false # optional
      preRelease: alpha
      fieldPaths:
        - .spec.foo
    - name: FeatureGate2
      enabled: true  # optional
      default: false # optional
      preRelease: alpha
      fieldPaths:
        - .spec.foo.bar
```
In the above case, `FeatureGate2`'s state will only be considered iff `FeatureGate1` is 
in the enabled state. The complete behaviour by keeping this point in mind is as follows:  

Consider that we are applying a config:

```
.spec
  .foo # <- controlled by FooFeatureGate
    .baz = 2
    .qux = 3 # <- controlled by QuxFeatureGate
```

and if the object was persisted, the persisted object was (for cases 5-8)

```
.spec
  .foo # <- controlled by FooFeatureGate
    .qux = 1 # <- controlled by QuxFeatureGate
```

Sr. No | Existing persisted object has field (.spec.foo.qux) | Feature Gate (FooFeatureGate) | Feature Gate (QuxFeatureGate) | Action | Object | 
| --------| -------- | ----------- | ------------ | --------- | --------- |
| 1 | no       | disabled    | disabled     | Drop foo & children | `.spec` |
| 2 | no       | disabled    | enabled      | Drop foo & children | `.spec` |
| 3 | no       | enabled     | disabled     | Update foo, drop qux | `.spec.foo.baz = 2` |
| 4 | no       | enabled     | enabled      | Update both foo and qux | `.spec.foo.baz = 2`, `.spec.foo.qux = 3` |
| 5 | yes      | disabled    | disabled     | Do nothing | `.spec.foo.qux = 1` |
| 6 | yes      | disabled    | enabled      | Do nothing | `.spec.foo.qux = 1` |
| 7 | yes      | enabled     | disabled     | Update foo, don't update qux | `.spec.foo.baz = 2`, `.spec.foo.qux = 1` |
| 8 | yes      | enabled     | enabled      | Update both foo and qux | `.spec.foo.baz = 2`, `.spec.foo.qux = 3` |
  
#### The `generation` Field

Please note that the check to see if the `generation` of the CR should be incremented or not will happen *after* the above logic has executed. This means that if an update to a field does not take effect on account of a feature gate being disabled, the generation of the CR will not be incremented as opposed to it being incremented if the update *did* take place. This helps clients, mainly controllers, know if an update actually happened or not. Please see Scenario 2 under Risks and Mitigations for more details.

#### Server Side Warnings

We propose sending out server side warnings in two situations:

1. The proposal also allows the user to specify an optional deprecation warning message for feature gates marked as deprecated. These warnings are surfaced through the `WarningsOnCreate()` and `WarningsOnUpdate()` methods. Warnings are sent out iff the fields under the feature gate marked as deprecated are "used". A field is said to be used if it is mutated as part of an update or is present as part of a create operation.
2. In cases such as the one described in Scenario 2 under Risks and Mitigations, a server side warning is surfaced to let the user know that a field or fields were not actually updated due to a feature gate(s) being disabled. This warning is surfaced for clients like `kubectl` to display to users.

### Additional Validation
Additional validation will be added to enforce the following:

- Any given `fieldPath` can be under atmost one feature gate.
  - In other words, intersection of `fieldPaths` of any two
    feature gates should be empty.
- Each `fieldPath` mentioned *must* be a JSON path.
- A deprecation warning can be specified iff the `preRelease`
  value is `deprecated`.
- Provided that `default` is specified:
  - If `preRelease` is `alpha` or `beta`, `default` = `false`
  - If `preRelease` is `stable`, `default` = `true`
- If `preRelease` is `deprecated`, then `default` _must_ be specified.

### Test Plan

- [x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

Unit tests would be added with any new code introduced.

Existing coverage before implementation:

- `k8s.io/apiextensions-apiserver/pkg/apis`: `14th June 2022` - 56.2
- `k8s.io/apiextensions-apiserver/pkg/apis/v1`: `14th June 2022` - 35.9
- `k8s.io/apiextensions-apiserver/pkg/apis/v1beta1`: `14th June 2022` - 29.1
- `k8s.io/apiextensions-apiserver/pkg/registry/customresource`: `14th June 2022` - 56.7

##### Integration tests

The following integration tests would be added:

- To ensure that the state of a feature gate is inferred according to the rules described in the implementation section.
- To ensure that disabling a feature gate will drop the field on a create operation.
- To ensure that on update operations, the feature behaves in the way defined in the matrix laid out in the Implementation section.
- To ensure that on usage of a deprecated field, a server side warning goes out (either default or user provided).
- To ensure that on disabling a feature gate on an update operation, the generation is not incremented.
- To ensure that on disabling a feature gate on an update operation, a server side warning is sent out.

##### e2e tests

As of now, we think e2e tests won't be nescessary as integration and unit tests cover all scenarios.

### Graduation Criteria

#### Alpha (1.25)

- Implement field level versioning for CRDs.
- Introduce a feature gate for this feature (switched off by default).
- Unit and integration tests to be added.

#### Beta

- Gather feedback from controller authors, CRD authors and end users.
- Address and incorporate feedback if needed.
- Feature gate turned on by default.
- Tests are present in testgrid and linked in the KEP.

#### GA

- Feature gate removed and fetaure generally available.

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

#### Upgrade

If the `kube-apiserver` is upgraded to a version with this feature, and this feature is enabled, we recommended updating any controller implementations by keeping in mind Scenario 2 under the Risks and Mitigations section and then upgrading the `kube-apiserver` component.

#### Downgrade

If the `kube-apiserver` is downgraded to a version without this feature, no change to existing code or config would be required.

### Version Skew Strategy

NA

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

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: CRDFieldLevelVersioning
  - Components depending on the feature gate: `kube-apiserver`

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

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

No

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
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


### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Numbers obtained using `unsafe.SizeOf()`

Enablding this will increase the size of the `CustomResourceDefinitionSpec` object:

Before enabling this feature:

|Object | Size |
| -------- | -------- |
| `CustomResourceDefinitionSpec`     | 184     |

After enabling this feature:

|Object | Size |
| -------- | -------- |
| `CustomResourceDefinitionSpec`     | 192     |

New intermediate Go objects introduced:

|Object | Size |
| -------- | -------- |
| `CustomResourceDefinitionFeatureGates`     | 32     |
| `CustomResourceDefinitionFeatureGate`  | 72 |

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

There will be additional work done in the final stage of the API request (i.e. in the strategy). However, we don't forsee considerable increase in time per request. To ensure this, benchmarks will be added for the additional logic in strategy.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

There will be additional work done in the final stage of the API request (i.e. in the strategy). However, we don't forsee considerable increase in resource usage (CPU and memory). To ensure this, benchmarks will be added for the additional logic in strategy.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- [x] Provisional KEP introduced
- [ ] KEP Accepted as implementable
- [ ] Implementation started

## Drawbacks

## Alternatives

One alternative that was considered was creating a new API type called `FeatureGateDefinition`
similar to `CustomResourceDefinition`:
```yaml
apiVersion: apiextensions.k8s.io/v1
kind: FeatureGateDefinition
metadata:
  name: MyFGD
spec:
  featureGates:
    - name: FeatureGate1
      default: false
      prerelease: alpha
    - name: FeatureGate2
      default: false
      prerelease: beta
```
The reason this approach was not pursued was mainly the additional complexity it introduces:
- We would have to make API calls to aggregate feature gate information every time a CR is
  created/updated.
    - This becomes more complex when we want to support cases such the one mentioned in the
      constraints section because this could potentially result in an unbounded number of
      API calls per CR create/update operation. 

## Infrastructure Needed (Optional)
