# KEP-4368: Support managedBy field for Jobs

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Prior work](#prior-work)
    - [Can the field be mutable?](#can-the-field-be-mutable)
    - [Use for MultiKueue](#use-for-multikueue)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Ecosystem fragmentation due to forks](#ecosystem-fragmentation-due-to-forks)
    - [Two controllers running when feature is disabled](#two-controllers-running-when-feature-is-disabled)
    - [Debuggability](#debuggability)
    - [Custom controllers not compatible with API assumptions by CronJob](#custom-controllers-not-compatible-with-api-assumptions-by-cronjob)
- [Design Details](#design-details)
    - [API](#api)
    - [Implementation overview](#implementation-overview)
    - [Job status validation](#job-status-validation)
    - [Terminating pods and terminal Job conditions](#terminating-pods-and-terminal-job-conditions)
    - [Mutability](#mutability)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
  - [Reserved controller name value](#reserved-controller-name-value)
  - [Defaulting of the for newly created jobs](#defaulting-of-the-for-newly-created-jobs)
  - [Alternative names for field](#alternative-names-for-field)
  - [Managed-by label](#managed-by-label)
  - [Alternative names for label (scopes)](#alternative-names-for-label-scopes)
    - [Generic kubernetes.io/managed-by](#generic-kubernetesiomanaged-by)
    - [Job-prefixed job.kubernetes.io/managed-by](#job-prefixed-jobkubernetesiomanaged-by)
  - [Alternative mechanisms to mirror the Job status](#alternative-mechanisms-to-mirror-the-job-status)
    - [mirrored-by label](#mirrored-by-label)
    - [Class-based approach](#class-based-approach)
    - [Annotation](#annotation)
  - [Custom wrapping CRD](#custom-wrapping-crd)
  - [Use the spec.suspend field](#use-the-specsuspend-field)
  - [Using field selectors](#using-field-selectors)
  - [Alternative ideas to improve debuggability](#alternative-ideas-to-improve-debuggability)
    - [Condition to indicate Job is skipped](#condition-to-indicate-job-is-skipped)
    - [Event indicating the Job is skipped](#event-indicating-the-job-is-skipped)
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

We support the "managedBy" field as a lightweight mechanism to delegate the Job
synchronization to an external controller.

## Motivation

As a part of [Kueue](https://github.com/kubernetes-sigs) (an effort done by
WG Batch, in cooperation with SIG Autoscaling, SIG Scheduling, SIG Apps and SIG Node) we
are working on a multi-cluster job dispatcher project, called
[MultiKueue](https://github.com/kubernetes-sigs/kueue/tree/main/keps/693-multikueue).

In the MultiKueue design, which follows manager-worker architecture, a user
creates a Job in the management cluster, but a mirror-copy of the Job is created
and executed in one of the worker clusters. The status updates of the mirror-Job
are reflected by the Kueue controller in the management cluster, in the status
of the Job created by the user.

In order to support this workflow we need a mechanism to disable the main Job
controller, and delegate the status synchronization to the Kueue controller.

### Goals

- support delegation of Job synchronization to an external controller

### Non-Goals

- passing custom parameters to the external controller

## Proposal

The proposal is to support the "managedBy" field in the Job spec
to indicate the only controller responsible for the Job object synchronization.

### User Stories (Optional)

#### Story 1

As a developer of Kueue I want to have Job API which allows me to implement the
MultiKueue design. For this reason I need a way to disable the main Job
controller on the management cluster.

The mechanism should be per-Job, because the management cluster might also be
one of the worker clusters, for two reasons:
1. Disabling the Job controller per cluster requires access to the `kube-controller-manager`
   manifest. Such access is generally discouraged by cloud providers.
2. The management cluster may also be a worker. Supporting this scenario is important
   for smooth transition of Kueue users from a single-cluster to multi-cluster.

Ideally, the mechanism should be lightweight so that it is easy to be adopted
by other Job CRDs supported by Kueue
(see [here](https://github.com/kubernetes-sigs/kueue/blob/6d428f3279a9ca0e204c083dc649dbbc6558db71/config/components/manager/controller_manager_config.yaml#L31-L42)):
MPIJob, RayJob, JobSet, multiple Kubeflow jobs.

It could be handy if the controller can be indicated by Kueue after the Job is
created, but before starting it. In the scenario of role sharing (where the
management cluster is also a worker), it would allow to avoid creation of a
mirror Job within the cluster.

### Notes/Constraints/Caveats (Optional)

#### Prior work

This approach of allowing another controller to mirror information between APIs
is already supported with the "managedBy" label used by
EndpointSlices ([`endpointslice.kubernetes.io/managed-by`](https://github.com/kubernetes/kubernetes/blob/5104e6566135e05b0b46eea1c068a07388c78044/staging/src/k8s.io/api/discovery/v1/well_known_labels.go#L27), see also in [KEP](https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/0752-endpointslices#endpointslice-api))
and IPAddresses ([`ipaddress.kubernetes.io/managed-by`](https://github.com/kubernetes/kubernetes/blob/5104e6566135e05b0b46eea1c068a07388c78044/staging/src/k8s.io/api/networking/v1alpha1/well_known_labels.go#L32)).

Note that, the reserved label values for the built-in controllers have the `k8s.io`
suffix, i.e.: `endpointslicemirroring-controller.k8s.io` and `ipallocator.k8s.io`,
for the EndpointSlices, and IPAddresses, respectively.

#### Can the field be mutable?

There is a potential risk of leaking pods, if the value is changed.
For example, assume there is a running Job, which is reconciled by the Job
controller, and has some pods created.
Then, if the value is switched to the mirroring Kueue controller (which by
itself does not manage pods). Then, the pods are leaking and remain running.

In order to avoid the risk of pods leaking between the controllers when changing
value, we make it immutable (allow to be added
on Job creation, but fail requests trying to update its value, see also
[mutability](#mutability)).

However, the question remains if we can make the field mutable when the job is
stopped, similarly, as we do with the `AllowMutableSchedulingDirectives` flag
which guards mutability of the Job's pod template labels.

It seems possible, and could be handy in [Story 1](#story-1), but it is also not
a blocker.

It would also complicate debuggability of the feature.

We decide to keep the field immutable, at least for [Alpha](#alpha), we will
re-evaluate the decision for [Beta](#beta).

#### Use for MultiKueue

The "managedBy" field is going to be added by a dedicated MultiKueue webhook
for Jobs created by users, and the Jobs remain suspended until ready to run.
Once the job is ready to run its mirror copy is created on a selected worker
cluster. Note that the mirror copy differs from the Job on the management cluster
as it does not have the "managedBy" field (removed), and will have different
UIDs.

When the job is running it is unsuspended (both on the management and the worker
clusters). Until the job completes, the MultiKueue controller mirrors back
the Job status, periodically,  from the worker onto the Job object in the
management cluster.

Note that for simplicity of the design, and thus debugging, there is one-to-one
relationship between the Job on the management cluster and the Job on the
worker cluster (it is [non-goal](https://github.com/kubernetes-sigs/kueue/tree/main/keps/693-multikueue#non-goals)
to distribute the Job among multiple clusters).

### Risks and Mitigations

#### Ecosystem fragmentation due to forks

The mechanism to disable the main Job controller opens the door for users to
substitute it with a fork. This may create more fragmentation in the community
as users may prefer to use their forked controllers rather than contribute
upstream.

First, this risk, to some extent, exists even today as admins with access to the
control plane can disable job controller by passing `--controllers=-job,*` in the manifest for
`kube-controller-manager` (see more info on the `--controllers` flag
[here](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager/)).

Second, we believe that users who had the need to fork the Job controller
already introduced dedicated Job CRDs for their needs.

#### Two controllers running when feature is disabled

It is possible that one creates jobs with "managedBy" field on kubernetes version
which enables the feature, then downgrades to the alpha version which disables
the feature gate. In that case the field remains present on the job and two
controllers (the built-in, and external controller) might start running and
compete with Job status updates at the same time.

Note that an analogous situation may happen when the version of Kubernetes
already supports the field, but the feature gate is disabled in `kube-controller-manager`.

To mitigate this risk we warn about it in Kueue documentation, to remove the
jobs using this field before downgrade or disablement of the feature gate.

Finally, this risk will fade away with time as the new versions of
Kubernetes support it.

#### Debuggability

With this mechanism new failure modes can occur. For example, a user may make
a typo in the field value, or the cluster administrator may not install the
custom controller (like MultiKueue) on that cluster.

In such cases the user may not observe any progress by the job for a long time
and my need to debug the Job.

In order to allow for debugging of situations like this the Job controller will
put a log line indicating the synchronization is delegated to another controller
(see [implementation overview](#implementation-overview)).

Additionally, re-evaluate extending the `kubectl` command-line tool
before [GA](#ga). We could extend the command to provide useful debugging
information with the following:
- new `MANAGED_BY` column for `kubectl get job -owide` (possibly also without `-owide`)
- a line in the `kubectl describe job` output, just before the list of events,
providing a user readable information if the Job is synchronized by a custom
controller.

Alternative ideas considered were
[a dedicated condition](#condition-to-indicated-job-is-skipped)
and [events](#event-indicating-the-job-is-skipped).

#### Custom controllers not compatible with API assumptions by CronJob

Currently, the validation of the Job status API is rather relaxed, allowing for
potential situations, when modified by custom controllers, which might be
problematic for the API consumers, like the built-in CronJob controller.

In particular, the CronJob relies on the assumption that the `CompletionTime`
is only set to successful jobs. Also, that a job does not flip its state from
Complete to Failed (or the other way round). Also, a finished job should not
flip back to non-finished.

In order to mitigate for this risk, we strengthen the validation of the Job
Status API to prevent faulty custom controllers to do unexpected changes to the
status (see [here](#job-status-validation)).

Additionally, we intend to strengthen the CronJob implementation to verify the
Job has the `Complete` condition before using `CompletionTime`
(see [here](https://github.com/kubernetes/kubernetes/blob/48b68582b89b0ae9ad4d435516b2dd5943f48cd3/pkg/controller/cronjob/cronjob_controllerv2.go#L452)).


## Design Details

#### API

```golang
type JobSpec struct {
  ...
	// ManagedBy field indicates the controller that manages a Job. The k8s Job
	// controller reconciles jobs which don't have this field at all or the field
	// value is the reserved string `kubernetes.io/job-controller`, but skips
	// reconciling Jobs with a custom value for this field.
	// The value must be a valid domain-prefixed path (e.g. acme.io/foo) -
	// all characters before the first "/" must be a valid subdomain as defined
	// by RFC 1123. All characters trailing the first "/" must be valid HTTP Path
	// characters as defined by RFC 3986. The value cannot exceed 64 characters.
	//
	// This field is alpha-level. The job controller accepts setting the field
	// when the feature gate JobManagedBy is enabled (disabled by default).
	// +optional
	ManagedBy *string
}
```

#### Implementation overview

We skip synchronization of the Jobs with the "managedBy" field, if it has any
different value than `kubernetes.io/job-controller`. When the synchronization is skipped,
the name of the controller managing the Job object is logged.

We leave the particular place at which the synchronization is skipped as
implementation detail which can be determined during the implementation phase,
however, two candidate places are:
1. inside `syncJob` function
2. inside `enqueueSyncJobInternal` function

Note that, if we skip inside `enqueueSyncJobInternal` we may save on some memory
needed to needlessly enqueue the Job keys.

There is no validation for the values of the field beyond that of standard
permitted field values.

#### Job status validation

We will add the following validation rules, which are currently satisfied by the
built-in Job controller, to mitigate the risk of [here](#custom-controllers-not-compatible-with-api-assumptions-by-cronjob):
- the `CompletionTime` field is only set for jobs with `Complete` condition (per
  [API comment](https://github.com/kubernetes/kubernetes/blob/48b68582b89b0ae9ad4d435516b2dd5943f48cd3/pkg/apis/batch/types.go#L436)) and cannot be changed once set.
- once the `Failed=True` or `Complete=True` conditions are added, their status
  cannot be changed. They are also mutually exclusive.

Additionally, we verify the following:
- the `completedIndexes` and `failedIndexes` fields are non-nil only when `.spec.completionMode: Indexed`
- the format of the `completedIndexes` and `failedIndexes` fields (but tolerate corrupted values if already present)

We may come up with more validation rules during the implementation phase.
The API comments to the Job status API fields will be updated to make the contract
clear.

We also plan to add a validation rule to make sure that the terminal conditions
(`Failed` and `Complete`) are only added to Job when all pods are terminated.
For that we plan to follow the approach described [below](#terminating-pods-and-terminal-job-conditions).

#### Terminating pods and terminal Job conditions

During the development process of Alpha in 1.30 we considered adding a
validation rule enforcing that the Job terminal conditions (`Failed` or `Complete`)
are only added when all pods are terminated (`status.terminating=0` and
`status.ready=0`). However, the rule turned out to be violated by the built-in
Job controller (see issue [#123775](https://github.com/kubernetes/kubernetes/issues/123775)).

We are going to solve this issue by delaying the addition of the Job terminal
conditions (Failed or Complete) until the pods are terminated
(`terminating=0` and `ready=0`).

One complication of this approach is that pod termination may take an
arbitrarily long if the non-standard pod graceful termination period
(`terminationGracePeriodSeconds`) is configured (30s by default). In order to
give the API clients flexibility to know the fate of a Job as soon as possible
we extend the scope for the following Job conditions:
- `FailureTarget` - introduced in [Pod Failure Policy KEP](https://github.com/kubernetes/enhancements/issues/3329)
- `SuccessCriteriaMet` - introduced in [Success Policy KEP](https://github.com/kubernetes/enhancements/issues/3998)

Note that, with the new conditions an API client can know the fate of a Job
earlier than currently, since adding the conditions will not wait for emptying
of the `uncountedTerminatedPods` structure, as shown in the
[experiment](https://github.com/kubernetes/kubernetes/issues/123775#issuecomment-2114710754).

Since the fix is needed for this KEP and the
[Pod Replacement Policy KEP](https://github.com/kubernetes/enhancements/issues/3939),
we intend to protect the fix with the OR of the `JobManagedBy` and the
`JobReplacementPolicy` feature gates.

Note also that the fix impacts `CronJob` when using the `Forbid` concurrency
policy (see [here](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#concurrency-policy)).
However, it can be argued that the current behavior of the CronJob / Job integration
does not match the expectations of the `Forbid` policy
(see [comment](https://github.com/kubernetes/kubernetes/issues/123775#issuecomment-2115200217)),
and can be handled as a bug.

The validation rule will be added in the +1 release (1.32) to ensure the
addition of the `Failed` and `Complete` conditions is delayed appropriately in
case of a skewed deployment when the kube-controller-manager is one version
ahead the kube-apiserver.

#### Mutability

We keep the field immutable. See also the discussion in
[Can the field be mutable?](#can-the-field-be-mutable).

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `pkg/controller/job`: `2023-12-20` - `91.5%`
- `pkg/registry/batch/job`: `2023-12-20` - `92.2%`
- `pkg/apis/batch/validation`: `2024-01-22` - `98.0%`
- `pkg/apis/batch/v1`: `2023-12-20` - `29.3%` (mostly generated code)

The following scenarios are covered:
- the Job controller reconciles jobs with the "managedBy" field equal to `kubernetes.io/job-controller` when the feature is enabled
- the Job controller reconciles jobs without the "managedBy" field when the feature is enabled
- the Job controller does not reconcile jobs with custom value of the "managedBy" field when the feature is enabled
- the Job controller reconciles jobs with custom "managedBy" field when the feature gate is disabled
- verify the field is immutable, both when the job is suspended or unsuspended; when the feature is enabled
- enablement / disablement of the feature after the Job (with custom "managedBy" field) is created
- verify the new Job Status API validation rules (see [here](#job-status-validation))

##### Integration tests

The following scenarios are covered:
- the Job controller reconciles jobs with the "managedBy" field equal to `kubernetes.io/job-controller`
- the Job controller reconciles jobs without the "managedBy" field
- the Job controller does not reconcile a job with any other value of the "managedBy" field. In particular:
  - it does not reset the status for a Job with `.spec.suspend=false`,
  - it does not add the Suspended condition for a Job with `.spec.suspend=true`.
- the Job controller reconciles jobs with custom "managedBy" field when the feature gate is disabled
- the `job_by_external_controller_total` metric is incremented when a new Job with custom "managedBy" is created
- the `job_by_external_controller_total` metric is not incremented for a new Job without "managedBy" or with default value
- the `job_by_external_controller_total` metric is not incremented for Job updates (regardless of the "managedBy")

During the implementation more scenarios might be covered.

##### e2e tests

The feature does not depend on kubelet, so the functionality can be fully
covered with unit & integration tests.

We propose a single e2e test for the following scenario:
- the Job controller does not reconcile a job with any other value of the "managedBy" field. In particular,
  it does not reset the status for an unsuspended Job.

### Graduation Criteria

#### Alpha

- skip synchronization of jobs when the "managedBy" field does not exist, or equals `kubernetes.io/job-controller`
- unit and integration
- implement the additional Job status validation (see [here](#job-status-validation)); also update the comments to the
  API fields affected by the new validation rules
- make CronJob more resilient by checking the Job condition is `Complete` when using `CompletionTime` (see [here](#custom-controllers-not-compatible-with-api-assumptions-by-cronjob))
- The feature flag disabled by default

#### Beta

- e2e tests
- implement the `job_by_external_controller_total` metric
- address all known inconsistencies between validation and the Job controller behavior,
  in particular: [#123775](https://github.com/kubernetes/kubernetes/issues/123775).
  The proposed approach is outlined in [here](#terminating-pods-and-terminal-job-conditions).
- verify the validation passes during e2e tests for open-source projects (like Kueue and JobSet)
- The feature flag enabled by default

Second Beta (1.32):
- Add validation rule that `Failed` and `Complete` conditions are added when
  `terminating=0`, and `ready=0`.

#### GA

- Address reviews and bug reports from Beta users
- Re-evaluate the ideas of improving debuggability (like [extended `kubectl`](#debuggability), [dedicated condition](#condition-to-indicated-job-is-skipped), or [events](#event-indicating-the-job-is-skipped))
- Re-evaluate the support for mutability of the field
- Asses the fragmentation of the ecosystem. Look for other implementations of a job controller and asses their conformance with k8s.
- Lock the feature gate

#### Deprecation

- Remove the feature-gate in GA+2.

### Upgrade / Downgrade Strategy

#### Upgrade

An upgrade to a version which supports this feature does not require any
additional configuration changes. This feature is opt-in at the Job-level, so
to use it users need to add the "managedBy" field to their Jobs.

#### Downgrade

A downgrade to a version which does not support this feature (1.29 and below)
does not require any additional configuration changes. All jobs, including these
that specified a custom value for "managedBy", will be handled in the default
way by the Job controller. However, this introduces the risk of
[two controllers running at the same time](#two-controllers-running-at-the-same-time-on-old-version).

In order to prepare the risk the admins may want to make sure the custom controllers
using the "managedBy" field are disabled before the downgrade.

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

### Version Skew Strategy

This feature is limited to control plane, so the version skew with kubelet does
not matter.

In case kube-apiserver is running in HA mode, and the versions are skewed, then
the old version of kube-apiserver may let the field get mutated, if the feature
is not supported on the old version.

In case the version of the kube-controller-manager leader is skewed (old), the
built-in Job controller would reconcile the Jobs with custom "managedBy" field,
running into the risk of
[two controllers running at the same time](#two-controllers-running-at-the-same-time-on-old-version).
It is recommended the users don't create jobs with custom "managedBy" field
during an ongoing upgrade.

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

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
  - Feature gate name: `JobManagedBy`
  - Components depending on the feature gate: `kube-apiserver`, `kube-controller-manager`
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

No.

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

However, when the feature is disabled and there are Jobs external controllers by
using "managedBy" field there is a risk of
[two controller running at the same time](#two-controllers-running-at-the-same-time-on-old-version).
Thus, it is recommended administrators make sure there are no Jobs using external
controllers before rollback.

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

The feature behaves as if it was enabled for the first time.

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

Yes, we will have unit tests for the feature enablement / disablement after the
Job is created (see [unit tests](#unit-tests)).

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

The rollout will not impact already running workloads, unless they set the
"managedBy" field to a custom value, but this would require a prior intentional
action.

###### What specific metrics should inform a rollback?

A substantial increase in the `apiserver_request_total[code=409, resource=job, group=batch]`,
while there are jobs with the custom "managedBy" field, can be indicative of
the built-in job controller stepping onto another controller, causing conflicts.
This can be further investigate per-job by checking the `.metadata.managedFields.manager`
being flipped between two owners.

The feature is opt-in so in case of such problems the custom "managedBy" field
should not be used.

Also, an admin could check if the value of the `job_by_external_controller_total`
matches the expectations. For example, if the value of the metric does not increase
when new jobs are being added with a custom "managedBy" field, it might be
indicative that the feature is not working correctly.

A substantial increase in `kube_cronjob_status_active` after upgrade may suggest
that the Jobs are not making progress. Additionally, if the non-progressing
Jobs use custom "managedBy" field, then rollback of the feature might be
justified to make the CronJobs run, by letting the built-in Job controller
handle the Jobs.

A substantial drop in the `job_sync_duration_seconds`, while the number of
jobs with the custom "managedBy" field is low, could be indicative of the
Job controller skipping reconciliation of jobs it should reconcile. This could
be further investigated per-job by looking at the timestamp of changes in
`.metadata.managedFields.time`, and owners in `.metadata.managedFields.manager`.

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
The Upgrade->downgrade->upgrade testing will be done manually prior to release
as Beta, with the following steps:

1. Start the cluster with the `JobManagedBy` enabled for api server and control-plane.

Then, create two-long running Jobs:
- `job-managed` with custom value of the "managedBy" field
- `job-regular` without the "managedBy" field

Then, verify that:
- the `job-managed` does not get status updates from built-in controller. Update the status manually and observe it is not reset by the built-in controller.
- the `job-regular` starts making progress (creates pods and updates the status accordingly by the built-in controller)

2. Simulate downgrade by disabling the feature for api server and control-plane.

Then, verify that:
- the `job-managed` starts to make progress, the status is reset, and updated to some new values
- the `job-regular` continues making progress

3. Simulate upgrade by re-enabling the feature for api server and control-plane.

Then, verify that:
- the `job-managed` stops getting status updates from the built-in controller. Update the status manually and observe it is not reset by the built-in controller.
- the `job-regular` continues making progress

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

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

Check the `job_by_external_controller_total` metric. If the value is non-zero
for a field, it means there were Jobs using the custom controller created, so
the feature is in use.

For a specific Job in question, check if the Job has the "managedBy" field.

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
- [x] API .spec
  - Condition name:
  - Other field:
    - `.spec.managedBy` for Jobs
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This feature does not propose SLOs. We don't expect any of the existing SLOs
to be impacted negatively by the proposal.

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

- [x] Metrics
  - Metric name:
    - `job_by_external_controller_total` (new), with the `controller_name` label
corresponding to the custom value of the "managedBy" field. The metric is
incremented by the built-in Job controller on each ADDED Job event,
corresponding to a Job with custom value of the "managedBy" field.
This metric can be helpful to determine the health of a job and its controller
in combination with already existing metrics (see below).
    - `apiserver_request_total[code=409, resource=job, group=batch]` (existing):
substantial increase of this metric, when additionally `job_by_external_controller_total>0`
may be indicative of two controllers stepping onto each-other causing
conflicts (see [here](#two-controllers-running-at-the-same-time-on-old-version)).
    - `kube_cronjob_status_active` (existing), substantial increase of this
metric, may suggest that there are accumulating non-progressing jobs controlled
by `CronJob`. If additionally `job_by_external_controller_total>0` it may suggest
that the Jobs are getting stuck due to not being synchronized by the custom
controller.
  - Components exposing the metric: kube-apiserver

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

No.

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

No.

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

No.

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No, unless a custom value of the "managedBy" field is set. In the worst case
scenario this is 9 bytes for the field name and 63 for the value.

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

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

No change from existing behavior of the Job controller.

###### What are other known failure modes?

None.


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

N/A.

## Implementation History

- 2023-12.20 - First version of the KEP
- 2024-03-05 - Merged implementation PR [Support for the Job managedBy field (alpha)](https://github.com/kubernetes/kubernetes/pull/123273)
- 2024-03-07 - Merged [Update Job conformance test for job status updates](https://github.com/kubernetes/kubernetes/pull/123751)
- 2024-03-08 - Merged [Follow up fix to the job status update test](https://github.com/kubernetes/kubernetes/pull/123815)
- 2024-03-11 - Merged [Adjust the Job field API comments and validation to the current state](https://github.com/kubernetes/kubernetes/pull/123792)
- 2024-05-16 - Merged [Fix the comment for the Job managedBy field](https://github.com/kubernetes/kubernetes/pull/124793)

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

### Reserved controller name value

We could also use just `job-controller` for the reserved value of the field
(without the k8s suffix).

**Reasons for discarding/deferring**

In the [prior work](#prior-work) the names end with `k8s.io` for the built-in
kubernetes controllers.

### Defaulting of the for newly created jobs

We could default the field in the `PrepareForCreate` function in `strategy.go`
for newly created jobs.

**Reasons for discarding/deferring**

We anyway need to support jobs without the field to be synchronized by the
Job controller for many releases before we can ensure that all the jobs have it.

An additional case for jobs without the field does not increase the
complexity significantly.

### Alternative names for field

Alternative names we considered:
- `controllerName`
- `controlledBy`

**Reasons for discarding/deferring**

The use of "controller" in the field name may be confused with the owning
controller (indicated by the OwnerReference). For a batch Job this might be
CronJob.

Choosing "managedBy" as the name we are also closer to the "managed-by" label
used in the [prior work](#prior-work).

### Managed-by label

We also considered the label `batch.kubernetes.io/managed-by`, which was planned
originally for this KEP.

**Reasons for discarding/deferring**

- no clear indication if supported, on old versions of k8s users would add the
  label, the external controller is likely to try to sync the Job, so is the
  built-in controller. With the field old k8s will reject the request.
- The risk of [two controllers running at the same time](#two-controllers-running-when-feature-is-disabled)
  is limited to disabling the feature or downgrade, requiring admin action.
  With label this was possible if one created the Job on an old k8s version.
- worse discoverability of this functionality would be worse, compared to the field.

Users don't know what the allowed values of the field are. The values are not
validated anyway.

### Alternative names for label (scopes)

#### Generic kubernetes.io/managed-by

The idea was suggested (see [thread](https://github.com/kubernetes/enhancements/pull/4370/files#r1433055461))
that the similar mechanism might be useful for other k8s controllers.

**Reasons for discarding/deferring**

There is currently no clear use case to support the label across the stack for
APIs like StatefulSets, Deployments, or DaemonSets.

A generic name without support across all k8s APIs might be confusing to the
users, and supporting it for all k8s APIs would be much bigger effort than
currently needed for the MultiKueue scenario use.

The "managed-by" label idea has significant risks, such as
[ecosystem fragmentation due to forks](#ecosystem-fragmentation-due-to-forks).
It makes sense to start with limited scope as a "pilot" and assess the impact.

#### Job-prefixed job.kubernetes.io/managed-by

The idea of job-prefixed label was suggested (see [here](https://github.com/kubernetes/enhancements/pull/4370/files#r1467153754)).

**Reasons for discarding/deferring**

We prefer the "batch" prefix for consistency with current labels used by the Job
controller (see [here](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/batch/types.go#L29-L46)).

The "batch" prefix fits well for the use case of supporting this label
in other batch Job CRDs supported by Kueue, such as JobSet, MPIJob, RayJob, etc.

### Alternative mechanisms to mirror the Job status

A couple of other approaches to allow mirroring of the Job status was considered.
They share the same risk as the managed-by label approach of substituting the
Job controller with a custom one implementing the Job API.

#### mirrored-by label

Similar idea as the managed-by trying to address the risk of replacing the
controller. To mitigate this risk we would document the label as used for the
purpose of mirroring only. No controllers with custom logic are supported.

**Reasons for discarding/deferring**

This is wishful thinking, the users would still be free to use other custom controllers for Job API.

#### Class-based approach

The idea is that there is an interim object which allows to specify also parameters
of the custom controllers.

**Reasons for discarding/deferring**

Also, the mechanism will be significantly harder to adopt by other Job CRD
projects with which Kueue integrates, so effectively we would need to have
multiple mechanisms in the ecosystem.

There is no need for the custom controllers in the job-mirroring use-case for
MultiKueue, so it adds unnecessary complexity.

#### Annotation

Annotations have more relaxed validation for values.

**Reasons for discarding/deferring**

This would not be consistent with the [prior work](#prior-work).

The ability to filter jobs by the label or field is likely useful by users to identify
jobs using custom controllers, for example by `kubectl get jobs -lbatch.kubernetes.io/managed-by=custom-controller`.

### Custom wrapping CRD

To avoid the [risk](#ecosystem-fragmentation-due-to-forks) we could introduce
a CRD that allows users to run and monitor the status of the k8s Jobs. In this
case a user creates, say `kueue.MulticlusterJob`. The instance of the `MulticlusterJob` embeds the
`JobSpec` and the `JobStatus`. Then, based on the MulticlusterJob, Kueue creates
the k8s Job on the selected cluster. Also, Kueue mirrors the status of the
running k8s Job as the status of the MulticlusterJob.

**Reasons for discarding/deferring**

Huge friction when transitioning from single cluster to multi cluster.
The in-house frameworks and pipelines need to be updated to use (create and monitor)
the MulticlusterJob. This requires all the pipelines and frameworks to be aware
of the multi-cluster. On the contrary, the proposed approach is transparent to
the ecosystem.

The approach isn't easily transferable for other Job CRDs. Creating a wrapping
Multicluster Job CRD per Job CRD type creates maintenance cost at the Kueue side.

Increases fragmentation in the ecosystem. We don't need yet another Job CRD and
uproot the k8s Job. We want to have less, more universal APIs.
We believe that the community driving the development of other Job CRDs is
likely to adopt the label-based mechanism for making their CRDs
multicluster-ready. So, the situation in which we go with the wrapping CRD for
the K8s job, but the label-based mechanism for other CRD Jobs may result in
decreased adoption of k8s Job, relative to the alternative Job CRDs, for the
batch-related tasks.

It would not be compatible with CronJob. Using CronJob with MultiKueue is a valid
use case we want to support.

### Use the spec.suspend field

This approach is to keep `spec.suspend=true` on the management cluster, while
allowing `spec.suspend=false` on the worker cluster and syncing the status.

**Reasons for discarding/deferring**

when `.spec.suspend=true` then the Job controller resets some of the status
fields (like `.status.active` or `.status.ready`), while not resetting others
(like status.Failed) so mirrored fields would be inconsistent.

Frameworks or users observing the main Job would get wrong information that it
is suspended, while some of its status fields would be updating.

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### Using field selectors

We consider using field selectors by the Job controller to identify the subset
of jobs it should watch. This could result in smaller memory usage.

**Reasons for discarding/deferring**

First, We use shared-informers (so that all core k8s controllers see all objects), then
we cannot make the memory saving this way.

Second, there is no "OR" logic in selectors, however, the built-in Job
controller needs to sync jobs in two cases:
1. old jobs without the field
2. new jobs with the field equal to `kubernetes.io/job-controller`

This means we would need to go via a difficult process of ensuring all jobs
have the field, or listen on events from two informers. In any case, the use of
field-selectors is significantly more complicated than the skip `if` inside the
`syncJob`, and does not allow for big memory gain.

### Alternative ideas to improve debuggability

#### Condition to indicate Job is skipped

In order to inform the user that a job is skipped from synchronization we
could add a dedicated condition, say `ManagedBy`, indicating that the job is
skipped by the built-in controller.

**Reasons for discarding/deferring**

- Since the Job field is immutable, then the usability of the condition is limited,
because the timestamp of the other fields will not bring extra debugging value.
- Conceptually, we want to give full ownership of the Job object to the other
job controller, objects mutated by two controllers could actually make debugging
more involving.
- The MultiKueue controller would have to non-trivially reconcile the Job Status.
If it just blindly mirrored the status from the worker cluster (as currently
planned), then it would remove the condition. Other controllers would need to be
careful not to remove the condition either.
- It requires extra request per job, and risks conflicts for the status Update
requests.

Additionally, notice that the analogous situation takes place when `spec.schedulerName`
does not match a custom scheduling profile. There is no condition indicating that.

#### Event indicating the Job is skipped

Job controller could emit event on the Job creation event indicating the Job
is synchronized by a custom controller. This would not run into the issue with
controllers conflicting on status updates.

**Reasons for discarding/deferring**

Events have expiration time, which is potentially cloud-provider dependent.
It makes them not that useful to debug situations when the Job didn't make
progress for long time. So, they would not give a reliable signal for debugging
based on playbooks.

Renewing the even on every Job update seems excessive from the performance
perspective.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
