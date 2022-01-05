
# KEP-2802: Identify Pod's OS during API Server admission


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
  - [Changes to kube-apiserver](#changes-to-kube-apiserver)
  - [Changes to PodSecurity Standards](#changes-to-podsecurity-standards)
  - [Changes to Kubelet](#changes-to-kubelet)
  - [Potential future changes to Scheduler](#potential-future-changes-to-scheduler)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

The primary goal of this enhancement is to define the OperatingSystem scheduling requirements for a Pod in a manner which is common in the Kubelet as well as the KubernetesControlPlane.
on which a pod would like to run.


## Motivation

Identifying the OS of the pods during the API Server admission time is crucial to apply
appropriate security constraints to the pod. In the absence of having information about a target OS for a pod, some admission
plugins may apply unnecessary security constraints to the pods(which may prevent them from running properly) or in the
worst case, don't apply security constraints at all.


### Goals

Admission plugins and/or Admission Webhooks identifying the Operating System on which
pod intends to run.

### Non-Goals

- Interaction with container runtimes.
- Interaction with scheduler (this may change in future).

## Proposal

We propose to add a new field to the pod spec called 
os to identify the OS of the containers specified in the pod. There is no default
value for this `OS` field or the `Name` field in `OS` struct.

```go
type PodSpec struct {
  // If specified, indicates the type of OS on which pod intends
  // to run.  This field is alpha-level and is only honored by servers that 
  // enable the IdentifyPodOS feature. This is used to help identifying the
  // OS authoritatively during the API server admission.
  // +optional
  OS *OS
}

// OS has information on the type of OS. 
// We're making this a struct for possible future expansion.
type OS struct {
  // Name of the OS. Current supported values are linux and windows. 
  Name string
}

```

With the above change, all the admission plugins which validate or mutate the pod can 
identify the pod's OS authoritatively and can act accordingly. As of now, PodSecurityAdmission
plugin is the only admission plugin can leverage this field on the pod spec. In addition, API validators like ValidatePod and ValidatePodUpdate should be modified. In future, we can
have a validating admission plugin for Windows pods as Linux and Windows host capabilities are
different and not all Kubernetes features are supported on Windows host.


### User Stories (Optional)


#### Story 1

As a Kubernetes cluster administrator, I want appropriate security contexts to be 
applied to my Windows Pods along with Linux pods

#### Story 2

As a Kubernetes cluster administrator, I want to use my own admission webhook for pods
based on the OS it intends to run on.

### Notes/Constraints/Caveats (Optional)

Since the OS field is optional in the pod spec, we can expect the users to omit this field
when submitting pod to API server. In this scenario, we expect admission plugins to treat the pod
the way it is being treated now.

### Risks and Mitigations

The primary risk is a bug in the implementation of the admission plugins that validate or mutate
based on the OS field in the pod spec. The best mitigation for that scenario is unit testing 
when this featuregate is enabled and disabled.
Additionally, there may be some end-user confusion on the functional consequences of setting the new OS field, given that it is optional.

## Design Details

### Changes to kube-apiserver

- Pod spec API validation will be adjusted to ensure values are not set for OS specific field that are irrelevant to the Pod's OS
- Unit tests will be added to new fields in the pod spec are classified as OS specific or not (and which OSes they are allowed for)
- E2e test that demonstrates only required OS specific fields are applied to pods during API admission time.

### Changes to PodSecurity Standards

- Pod Security Standards will be reviewed and updated to indicate which Pod OSes they apply to
- The restricted Pod Security Standard will be reviewed to see if there are OS-specific requirements that should be added
- The PodSecurity admission implementation will be updated to skip checks which do not apply to the Pod's OS
- Unit and E2e tests which demostrate the PodSecurity admission plugin is behaving correctly with the new OS field

Pod Security Standards are to be changed in 1.25 timeframe to accommodate the supported kubelet and kube-apiserver skew.


### Changes to Kubelet

Apart from the above API changes, we intend to make the following changes to Kubelet:
- Kubelet should reject admitting pod if the kubelet cannot honor the pod.Spec.OS.Name. For instance, if the OS.Name does not match the host os.


### Potential future changes to Scheduler

We let the users to explicitly specify nodeSelectors/nodeAffinities+tolerations or runtimeclasses to express their intention
to run an particular OS. However, in future, once the OS struct expands, we can see if we can leverage those fields to
express scheduling constraints. During the alpha, we assume there are no scheduling implications.



### Test Plan

- Unit tests covering API server defaulting to various fields within pod with and without this feature
- Unit tests covering admission plugins which validate/mutate pod spec based on this feature
- Unit and E2E tests for Kubelet changes.
- Updates to the `sig-windows` tagged tests to utilize to direct windows scheduling for all pods .
### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Basic units and e2e tests completed and enabled

#### Alpha -> Beta Graduation
- Expand the e2e tests with more scenarios
- Gather feedback from end users
- Tests are in Testgrid and linked in the KEP

#### Beta -> GA Graduation
- 2 examples of end users using this field

### Upgrade / Downgrade Strategy

- Upgrades:
  When upgrading from a release without this feature, to a release with `IdentifyPodOS` feature enabled, there is no change to existing behavior unless user specifies
  OS field in the pod spec.
- Downgrades:
  When downgrading from a release with this feature to a release without `IdentifyPodOS`, the current behavior will continue.
### Version Skew Strategy

If the feature gate is enabled there are some kubelet implications as the code to strip security constraints based on OS can be removed and we need to add
admission/denying in the kubelet logic which was mentioned above. Older Kubelets without this change will continue stripping the unnecessary fields in the pod spec which is the current behavior.

## Production Readiness Review Questionnaire


### Feature Enablement and Rollback



###### How can this feature be enabled / disabled in a live cluster?


- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: IdentifyPodOS
  - Components depending on the feature gate:
    - kubelet
    - kube-apiserver

###### Does enabling the feature change any default behavior?
A Kubelet with a misscheduled pod (i.e. trying to run a windows pod on a linux node) will fail *before* trying to run a container (i.e. it will never actually invoke the underlying CRI), as opposed to after.

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Using the featuregate is the only way to enable/disable this feature. We'd have unit tests which exercise the update validation code which changes as a result of this feature. The change to update validation comes from the fact that we will allow certain fields to be empty or invalidated when this OS field in the pod spec is set.

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

The admission plugins/Kubelet can act based on the OS field in pod spec if set by the end-user

###### Are there any tests for feature enablement/disablement?
Yes, unit and e2es tests for feature enabled, disabled


### Rollout, Upgrade and Rollback Planning


###### How can a rollout or rollback fail? Can it impact already running workloads?
It shouldn't impact already running workloads. This is an opt-in feature since users need to explicitly set the OS parameter in the Pod spec i.e .spec.os field. 
If the feature is disabled the field is preserved if it was already set in the presisted pod object, otherwise it is silently dropped.


###### What specific metrics should inform a rollback?
N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?
Not yet tested.


###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?
None. This feature will be additive


### Monitoring Requirements


###### How can an operator determine if the feature is in use by workloads?

By looking at the pod's spec. Some fields like Security Constraint Contexts may not
be applied to the pod spec if the proper OS has been set.

###### How can someone using this feature know that it is working for their instance?

Windows Pods and Linux Pods with proper OS field set in the pod spec would cause pods
to run properly on desired OS


- [x] Events
  - Event Reason: Corresponding admission plugins(PodSecurityAdmission plugin) will send pod denied event.
  - Event Reason: Kubelet will send pod admitted/denied event.
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?
All the pods which have OS field set in the pod spec should have OS specific constraints/defaults applied.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?


- [x] Metrics
  - Metric name: `kube_pod_status_phase`
  - Metric name: `apiserver_request_total`
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [] Other (treat as last resort)

If the pod is rejected by admission plugins, we'd get a 400 series error. The increase in 400 series errors
during pod creation/updation would give us an indication of the health. This can be measured via metric
`apiserver_request_total`

If the pod gets admitted at the kube-apiserver and gets rejected by kubelet, the metric `kube_pod_status_phase`
would give us an indication of where the failure is happening.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies


###### Does this feature depend on any specific services running in the cluster?
kube-apiserver and kubelet


### Scalability



###### Will enabling / using this feature result in any new API calls?
No

###### Will enabling / using this feature result in introducing new API types?
Yes


###### Will enabling / using this feature result in any new calls to the cloud provider?
No


###### Will enabling / using this feature result in increasing size or count of the existing API objects?
It increases the size of Pod object since a new string field is added.


###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?
No


###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?
No

### Troubleshooting


###### How does this feature react if the API server and/or etcd is unavailable?


###### What are other known failure modes?



###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History



## Drawbacks

## Alternatives

Identifying the pods targeting Windows nodes during the scheduling phase can be done
in the following ways:
- Based on [nodeSelector and tolerations](https://kubernetes.io/docs/setup/production-environment/windows/user-guide-windows-containers/#ensuring-os-specific-workloads-land-on-the-appropriate-container-host) in the pod spec with 
  Windows node specific labels.
- Based on [runtimeclasses](https://kubernetes.io/docs/setup/production-environment/windows/user-guide-windows-containers/#ensuring-os-specific-workloads-land-on-the-appropriate-container-host) in the pod spec

The runtimeclass is a higher level abstraction which gets translated again to nodeSelectors+tolerations. While the 
nodeSelector with reserved OS label is good enough, it has following shortcomings:

- Piggybacking on nodeSelectors+tolerations to definitively identify Pod OS may not be ideal experience for end-users as they can convey scheduling constraints using the same abstractions.
- We can use this field when the hostOS is not not always equal to Container OS. For example, Linux Containers on Windows(using WSL).

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
