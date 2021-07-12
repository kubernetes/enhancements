
# KEP-2802: Identify Windows Pods during API Server admission


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

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
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

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The main goal of this enhancement is to authoritatively identify Windows
pods during the API Server admission time. 


## Motivation

Identifying Windows pods during the API Server admission time is crucial to apply
appropriate security constraints to the pod. Without proper identification, some admission
plugins may apply unnecessary security constraints to the Windows pods or in the
worst case, don't apply security constraints at all.


### Goals

Admission plugins and/or Admission Webhooks identifying if the incoming pod
is Windows or not.


### Non-Goals


- Validating if the Windows pods have appropriate security constraints.
- Interaction with container runtimes.

## Proposal


As mentioned [earlier](#Motivation) identifying the Windows pods during the scheduling phase can be done
in the following ways:
- Based on [nodeSelector and tolerations](https://kubernetes.io/docs/setup/production-environment/windows/user-guide-windows-containers/#ensuring-os-specific-workloads-land-on-the-appropriate-container-host) in the pod spec with 
  Windows node specific labels.
- Based on [runtimeclasses](https://kubernetes.io/docs/setup/production-environment/windows/user-guide-windows-containers/#ensuring-os-specific-workloads-land-on-the-appropriate-container-host) in the pod spec

The problem with using nodeSelector and tolerations is that any unprivileged user/entity can apply the 
nodeSelector and/or tolerations in the pod spec making it unsecure whereas RuntimeClasses are [recommended](https://kubernetes.io/docs/concepts/containers/runtime-class/#setup) to be
created by cluster administrator making them authoritative enough to be used.

### User Stories (Optional)


#### Story 1

As a Kubernetes cluster administrator, I want appropriate security contexts to be 
applied to my Windows Pods along with Linux pods

#### Story 2

As a Kubernetes cluster administrator, I want to use my own admission webhook for
Windows pods.

### Notes/Constraints/Caveats (Optional)

While the `scheduling` field in the RuntimeClass is created to handle scheduling constraints, we have a reserved label
[`kubernetes.io/os`](https://kubernetes.io/docs/reference/labels-annotations-taints/#kubernetes-io-os) to identify
Windows nodes from their Linux counterparts. If the RuntimClass's scheduling field has a nodeSelector  `kubernetes.io/os: 'windows'`,
we can authoritatively say that the pod is indeed a Windows one during the api-server admission time. Querying the RuntimeClass
during admission time may be expensive operation as the admission plugin is not stateless anymore. 

### Risks and Mitigations



Existing users may be impacted as we're enforcing to use RuntimeClasses instead of plain nodeSelector and tolerations.


The downside of this approach is since we're making RuntimeClass as default choice for the users, existing users have to update their pod spec or workload pod template spec to ensure that their workloads are not broken during upgrades. 
This can be mitigated by 
- Having appropriate warnings in the releases before we beta and GA will mitigate the problem.
- Having an example out-of-tree admission webhook which can detect such workloads and update them.(Having an in-tree admission plugin which mutates the pod spec to have appropriate Windows specific RuntimeClass may be too hard to do generically and not transparent to the user)


## Design Details


We can piggyback on the existing RuntimeClass admission controller to query for the RuntimeClass and see if it has
`kubernetes.io/os: windows` in it's scheduling field. 
TODO: We need to get an ack from sig-auth to use it.

If we cannot piggyback on the RuntimeClass controller we need to build our own validating admission controller for Windows specific pods.

### Test Plan


Whatever path we choose, unit and e2e tests are a hard requirement to ensure smooth
transition to make RuntimeClasses default choice for the users.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled
- Users getting a warning to switch to RuntimeClasses in beta release

#### Alpha -> Beta Graduation
- Gather feedback from end users
- Tests are in Testgrid and linked in the KEP
- Deny admission to pods that have only `kubernetes.io/os: windows` in nodeSelector
  but don't have an associated RuntimeClass with `kubernetes.io/os: windows` in
  scheduling field.

#### Alpha -> Beta Graduation
- 2 examples of end users using this field

### Upgrade / Downgrade Strategy

- Upgrades:
  When upgrading from a release without this feature, to a release with `RuntimeClassesForWindows` enabled, we will honor both nodeSelector and RuntimeClasses
  but a warning will be sent to the user that `honoring explicit nodeSelector without Runtimeclasses will be stopped from next release` if the pod has just nodeSelector set. This ensures users have enough time to move without breaking their workloads.
- Downgrades:
  When downgrading from a release with this feature to a release without `RuntimeClassesForWindows`, the existing behavior will continue where both nodeSelector and RuntimeClasses are honoured without a warning message.

### Version Skew Strategy

If the feature gate is enabled, only admission controller(either RuntimeClass or Windows specific admission controller) should be impacted. This feature may have some kubelet implications as the code to strip security constraints based on OS can be removed.


## Production Readiness Review Questionnaire


### Feature Enablement and Rollback



###### How can this feature be enabled / disabled in a live cluster?


- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: RuntimeClassesForWindows
  - Components depending on the feature gate:
    - kubelet

###### Does enabling the feature change any default behavior?
In alpha, the users would get a warning 

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Using the featuregate is the only way to enable/disable this feature

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

The admission controller would start sending warning messages to users.

###### Are there any tests for feature enablement/disablement?
Yes, unit and integration tests for feature enabled, disabled


### Rollout, Upgrade and Rollback Planning


###### How can a rollout or rollback fail? Can it impact already running workloads?



###### What specific metrics should inform a rollback?


###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?



###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?



### Monitoring Requirements


###### How can an operator determine if the feature is in use by workloads?

###### How can someone using this feature know that it is working for their instance?


- [x] Events
  - Event Reason: No Windows specific RuntimeClass but just nodeSelector.
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?


###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?


- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?


### Dependencies


###### Does this feature depend on any specific services running in the cluster?



### Scalability



###### Will enabling / using this feature result in any new API calls?


###### Will enabling / using this feature result in introducing new API types?



###### Will enabling / using this feature result in any new calls to the cloud provider?



###### Will enabling / using this feature result in increasing size or count of the existing API objects?



###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?



###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?


### Troubleshooting


###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?



###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History



## Drawbacks

## Alternatives

Have a OS specific field in the pod spec which can be either `windows` or `linux` etc. The main advantage of this approach is this can be non-breaking to the existing users, however we already have a way to distinguish pods using the scheduling field in RuntimeClasses. 
Following are the reasons to not choose this approach:
- Having two different ways to do the same thing may be confusing from user experience standpoint
- With proper phasing from using just nodeSelector to RuntimeClasses, we can ensure that the users are not broken. We will also provide an example documentation of this needs to be
fixed for existing workload and an example admission webhook which can properly mutate the pod spec if a Windows specific RuntimeClass already exists.

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
