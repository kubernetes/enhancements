# KEP-2855: Automated CRD status.storedVersions Management

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [CA Requirements](#ca-requirements)
    - [Alternative Approaches](#alternative-approaches)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Proposed Admission Webhook](#proposed-admission-webhook)
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

The [kube-storage-version-migrator] supports migrating existing resources to a new storage version but it does not modify a CRD's `status.storedVersions` field after a successful migration, instead requiring that the cluster administrator updates this field manually. As adoption for CRDs continues to increase and new versions of those CRDs are released this design decision will place a non-trivial amount of work on cluster administrators.

Historically, the [kube-storage-version-migrator] has not managed modifying the `status.storedVersion` field because the CRD's storedVersion could change mid-migration unbeknownst to the migrator due to slow/failed watches. This KEP proposes a solution that would allow the [kube-storage-version-migrator] to safely modify a CRD's `status.storedVersions` after a successful migration, reducing the burden on cluster administrators.

[kube-storage-version-migrator]: https://github.com/kubernetes-sigs/kube-storage-version-migrator

## Motivation

It is desirable for the [kube-storage-version-migrator] to modify a CRD's `status.storedVersions` array after a successful migration rather than relying on a cluster admin to manually perform the same steps.

### Goals

- Allow the [kube-storage-version-migrator] to safely modify a CRD's `status.storedVersions` after a successful migration.
- Stop other actors from modifying the storage version while a migration is ongoing.

## Proposal

In order for the [kube-storage-version-migrator] to safely modify a CRD's `status.storedVersions` field after a successful migration, this KEP proposes that the [kube-storage-version-migrator] introduces a [Validating Admission Webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/) that prevents changes to a CRD's `spec.versions[*].storage` fields during a migration using a [fail](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#failure-policy) failure policy.

The validating admission webhook would need to:

- Intercept all Update operations against CRDs.
- Prevent changes to a CRD's `spec.versions[*].storage` fields if the CRD includes the `ksvm.sigs.k8s.io/migrating=true` annotation.

A summary of the new migration workflow can be found below:

1. A `StorageVersionMigration` CR for the `foo` CRD has been created, possibly by the trigger-controller as [described here](https://github.com/kubernetes-sigs/kube-storage-version-migrator/blob/acdee30ced218b79e39c6a701985e8cd8bd33824/USER_GUIDE.md#storage-version-migrator-in-a-nutshell)). 
2. The [kube-storage-version-migrator] adds the `ksvm.sigs.k8s.io/migrating=true` annotation to the`foo` CRD.
3. The [kube-storage-version-migrator] begins migrating all `foo` resources as it does today.
4. When a migration is successful, the `foo` CRD's `status.storedVersion` is patched to only include the new stored version.
5. When a migration is finished, the [kube-storage-version-migrator] removes the `ksvm.sigs.k8s.io/migrating=true` annotation from the `foo` CRD.

### User Stories (Optional)

#### Story 1

As a cluster admin, I would like the [kube-storage-version-migrator] to automatically update a CRD's `status.storeVersions` after a successful a migration.

### Notes/Constraints/Caveats (Optional)

#### CA Requirements

The introduction of an admission webhook would require a valid CA.

These certificates may be manually created, but it's possible that the CA will be managed by [cert-manager](https://github.com/jetstack/cert-manager) or other cert-provisioners. If the webhook's certificates are managed by means of a CRD (as is the case with cert-manager) a cyclical dependency is introduced in which:

- The [kube-storage-version-migrator] manages the storage versions of CRD introduced by the cert provisioner.
- The [kube-storage-version-migrator] relies on the cert provisioner for cert management.

The above dependencies could cause an issue when:

- A migration for one of the CRDs owned by the cert provisioner is started.
- A new version of the cert provisioner is deployed that is not compatible with the CRD being migrated and introduces a new stored version.
- The cert provisioner upgrade fails because of the admission webhook and is no longer able to provision new certificates.
- The certificate for ksvm expires.

In the situation described above, the cluster administrator would need to manually revert the cert provision upgrade to proceed.

#### Alternative Approaches

- Instead of relying on an annotation to identify CRDs currently being migrated, the admission webhook could check for the existence of a non-completed `StorageVersionMigration` resource.

### Risks and Mitigations

- It is possible that the admission webhook introduced in this KEP may prevent changes to a CRD's `spec.versions[*].storage` fields if the migrating annotation is not successfully removed. Update operations against a CRD will also fail if the admission webhook is unavailable. Cluster Administrators can resolve this issue by deleting or modifying the validating admission webhook.

## Design Details

### Kube Storage Version Migrator Changes

- When reconciling a `StorageVersionMigration` CR, the [kube-storage-version-migrator] must start by adding the `ksvm.sigs.k8s.io/migrating=true` annotation to the CRD being migrated.
- When a migration is successful, the migrated CRD's `status.storedVersion` is patched to only include the new stored version.
- After a successful or failed migration, the [kube-storage-version-migrator] removes the `ksvm.sigs.k8s.io/migrating=true` annotation from the migrated CRD. 
- Should the `StorageVersionMigration` CR be deleted, the kube-storage-version-migrator] should remove the `ksvm.sigs.k8s.io/migrating=true` annotation from the CRD.

### Proposed Admission Webhook

As described earlier, the admission webhook would prevent update operations to any CRD's `spec.versions[*].storage` fields when the `ksvm.sigs.k8s.io/migrating=true` annotation is present. An example of the ValidationWebhookConfiguration can be found below:

```yaml=
apiVersion: admissionregistration.k8s.io/v1beta1
  kind: ValidatingWebhookConfiguration
  metadata:
    name: crd.kube-storage-version-migrator.io
  webhooks:
  - name: crd.kube-storage-version-migrator.io
    clientConfig:
      service:
        namespace: <deployed namespace>
        name: webhooks
       path: <TBD>
      caBundle: <KUBE_CA_HERE>
    rules:
    - operations:
      - Update
      apiGroups:
      - "apiextensions.k8s.io"
      apiVersions:
      - "v1" # Exclude v1beta1 as it is no longer served in K8s 1.22: https://kubernetes.io/docs/reference/using-api/deprecation-guide/#customresourcedefinition-v122
      resources:
      - customresourcedefinitions
    failurePolicy: fail
```

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

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
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: 
  - Components depending on the feature gate: 
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
The following bullets outline changes in default behavior on clusters using the [kube-storage-version-migrator]:

- Changes to a CRD are blocked during migration
- Cluster Admins are no longer responsible for manually updating a CRD's `status.storedVersions` array.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

Expected behavior.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

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

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Implementing this feature will result in:
- A new ValdiatingWebhookConfiguration resource that intercepts create/update/delete requests against CRDs
- A transiet annotation applied to CRDs undergoing migration.


###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

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

- 

## Alternatives

- We had considered adding this as a feature to the [Operator-Lifecycle-Manager](github.com/operator-framework/operator-lifecycle-manager/) project, but believed that this feature fulfills a generic need better delivered within the [kube-storage-version-migrator] project.

## Infrastructure Needed (Optional)

None.