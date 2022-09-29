<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] sig-cli
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
# KEP-3517: kubectl: Client-Side-Apply to Server-Side-Apply migration

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
  - [Example Usage](#example-usage)
  - [Transferring Field Ownership](#transferring-field-ownership)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Undermining SSA](#undermining-ssa)
    - [Operation reversion](#operation-reversion)
    - [last-applied-configuration annotation](#last-applied-configuration-annotation)
- [Design Details](#design-details)
  - [Approach](#approach)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Graduation Criteria](#graduation-criteria-1)
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
  - [Server-Side Solution](#server-side-solution)
  - [Separate Migration Command](#separate-migration-command)
  - [kubectl Plugin](#kubectl-plugin)
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

Adds a migration path to use server-side-apply from usage of client-side-apply.

## Motivation

The current documentation for Kubernetes Server-Side-Apply has [instructions for
migrating](https://kubernetes.io/docs/reference/using-api/server-side-apply/#upgrading-from-client-side-apply-to-server-side-apply) from usage of client-side-apply to server-side-apply:

> Client-side apply users who manage a resource with kubectl apply can start using server-side apply with the following flag.
>
>     kubectl apply --server-side [--dry-run=server]


Unfortunately it is not so simple. By following these instructions you can be
left in a situation where after migrating to server-side-apply, certain fields
become un-deletable from the object without a manual update repairing the managed fields. 
Here is a simple reproducing case:

Create a configmap in the cluster with client-side apply:
```shell
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key: value
  legacy: unused
EOF
```
```console
configmap/test created
```

Apply the same manifest with `--server-side` flag, as per server-side-apply migration instructions:
```shell
cat <<EOF | kubectl apply --server-side -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key: value
  legacy: unused
EOF
```
```console
configmap/test serverside-applied
```

Remove one of the values from the configmap, we choose `legacy` key, and apply
again using server-side apply. You would expect that the field is now removed
from the object, but it is not:
```shell
cat <<EOF | kubectl apply --server-side -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key: value
EOF
```
```console
configmap/test serverside-applied
```

Check the configmap, expecting the legacy value to be removed, and be surprised that it remains:
```shell
kubectl get configmap -o yaml test
```
```console
apiVersion: v1
kind: ConfigMap
data:
  key: value
  legacy: unused
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","data":{"key":"value"},"kind":"ConfigMap","metadata":{"name":"test","namespace":"default"}}
  name: test
  namespace: default
```

This KEP is intended to fix this problem by providing users with a path to completely
migrate their managed fields to be used with server-side-apply.

### Goals

This bug prevents users of server-side-apply, which is in GA, from deleting any fields of their objects until they manually update their managed fields.
It especially poses huge challenges to cluster operators seeking to upgrade a large collection of resources to server-side-apply. Without an automatic method of preparing objects for use with SSA,
adoption for server-side-apply is blocked for many users.

1. The primary goal of this KEP is to provide a pathway for users blocked by this issue to adopt SSA.
2. Provide an idempotent migration to convert large collections of resources to server-side apply, especially for automatic systems
3. Keep the number of requests to the apiserver limited on the nominal server-side apply path (no additional writes)


### Non-Goals

1. Provide a server-side fix to this problem (see alternatives considered)
2. Deprecate the `kubectl/last-applied-configuration` annotation

## Proposal

Provide a transparent migration path of client-side-apply field ownership to
being owned by server-side-apply.

```shell
kubectl apply -f <path/to/file> --server-side
```

Today when using server-side-apply, kubectl unconditionally does an initial fetch
of the object. This is an acknowledged anti-pattern of SSA, and will likely be changed
in the future. Since it is pre-existing behavior that does not do much harm, the SIG has
come to consensus that the behavior can remain for a few more releases until client-side-apply
is out of vogue, especially since SSA is now GA.

The initially fetched object will be tested to check if the migration should be
performed. This check is done by performing the managed fields migration, which is
idempotent. If the migration results in a no-op change, then SSA proceeds as normal.
If the migration is required, then a `UPDATE` or JSON PATCH is sent prior to the
normal SSA operation. This is a technical requirement: managed fields may not be
modified within an SSA operation.

### Example Usage

The usage should be transparent to the user, so they don't have to learn any new
interface. Users of server-side-apply will have their fields owned by client-side-apply
transparently migrated for them.

```shell
# Migrate fields owned by kubectl-client-side-apply to now be owned by `myssamanager`
kubectl apply -f <path/to/file> --server-side --fieldmanager=myssamanager
```

Before continuing with the apply operation, kubectl will give ownership to all
fields previously managed through client-side-apply, to server-side-apply.
And drop client-side-apply's claim on the field.

If client-side-apply does not own any fields, then the behavior is to no-op and
continue with the `apply` invocation.

### Transferring Field Ownership

The fields previously managed by kubectl-client-side-apply before this operation
will afterwoards be managed by whatever field manager was provided to the command
line via the `--fieldmanager` flag.

If a user wishes any specific fields to then be owned by other field managers piecemeal,
then the user may do so normally using documentation provided about transferring
field ownership.

### Risks and Mitigations

### Undermining SSA

It can be said that requiring the fetch of an object before applying the object
is an anti-pattern of using SSA. Since this is the behavior of kubectl today,
it is seen as relatively low risk. But the question arises, when would it be
safe to remove the initial fetch and migration step?

THe migration needs only to be applied while kubectl users use client-side-apply.
Once SSA becomes on-by-default, we can assume there are no more client-side-apply
users after that version leaves the kubectl support window. At that point this
migration and extra fetch should be removed from kubectl.

#### Operation reversion

If the user reverts to using client-side-apply again, this operation's effect
will be reverted. Because this transformation is applied before SSA is used
as a precondition, the bug which motivates this KEP will not resurface despite
reverting to client-side-apply.

Once all kubectl users have switched to server-side-apply, the migration will be
stable and it will not be fetch the object and check if it needs to be done
anymore.

#### last-applied-configuration annotation

There was a concern brought up that the last-applied-configuration would no longer
be updated although tools still depend upon it.

There should be a separate deprecation plan for this annotation, but this KEP does not
seek to remove this flag.

Since before SSA went GA, the kube-apiserver [will generate a
last-applied-configuration annotation](https://github.com/kubernetes/kubernetes/blob/2e771b8e745c4a3be0d5bae3a6dc94087284c73b/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/lastappliedupdater.go#L60-L69) from managedFields owned by `kubectl`. The annotation
is generated using the current object state before defaulting/mutating admissions.
Tools depending upon this annotation will remain working as before.

## Design Details

The bug this enhancement fixes is due to the fact that when `--server-side` is
used today, it causes all fields to be owned by **both** the client-side-apply manager
and the server-side-apply manager. When the SSA manager drops a field, the CSA
manager still owns it, and it is not deleted from the object:

ManagedFiields Before --server-side:
```shell
cat <<EOF | kubectl apply --f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key: value
  legacy: unused
EOF
kubectl get configmap test -o yaml --show-managed-fields
```

```console
apiVersion: v1
kind: ConfigMap
data:
  key: value
  legacy: unused
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","data":{"key":"value","legacy":"unused"},"kind":"ConfigMap","metadata":{"annotations":{},"name":"test","namespace":"default"}}
  managedFields:
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:data:
        .: {}
        f:key: {}
        f:legacy: {}
      f:metadata:
        f:annotations:
          .: {}
          f:kubectl.kubernetes.io/last-applied-configuration: {}
    manager: kubectl-client-side-apply
    operation: Update
  name: test
  namespace: default
```

After --server-side:
```shell
cat <<EOF | kubectl apply --server-side -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key: value
  legacy: unused
EOF
kubectl get configmap test -o yaml --show-managed-fields
```

```console
apiVersion: v1
kind: ConfigMap
data:
  key: value
  legacy: unused
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","data":{"key":"value","legacy":"unused"},"kind":"ConfigMap","metadata":{"name":"test","namespace":"default"}}
  managedFields:
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:data:
        f:key: {}
        f:legacy: {}
    manager: kubectl
    operation: Apply
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:data:
        .: {}
        f:key: {}
        f:legacy: {}
      f:metadata:
        f:annotations:
          .: {}
          f:kubectl.kubernetes.io/last-applied-configuration: {}
    manager: kubectl-client-side-apply
    operation: Update
  name: test
  namespace: default
```

Note that after the `--server-side` command there are now two owners of `key` and `legacy` fields.
This is what causes problems for users. When the user removes the
field `legacy` via server-side-apply, it surprisingly persists within the object:


After field is removed from SSA apply:
```shell
cat <<EOF | kubectl apply --server-side -f -                                                                                           csa-to-ssa*
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key: value
EOF
kubectl get configmap test -o yaml --show-managed-fields
```

```console
apiVersion: v1
kind: ConfigMap
data:
  key: value
  legacy: unused
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","data":{"key":"value"},"kind":"ConfigMap","metadata":{"name":"test","namespace":"default"}}
  managedFields:
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:data:
        f:key: {}
    manager: kubectl
    operation: Apply
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:data:
        .: {}
        f:key: {}
        f:legacy: {}
      f:metadata:
        f:annotations:
          .: {}
          f:kubectl.kubernetes.io/last-applied-configuration: {}
    manager: kubectl-client-side-apply
    operation: Update
  name: test
  namespace: default
```

Note that the `f:legacy: {}` entry was removed from the `kubectl` `Apply` manager,
but it persists on the `kubectl-client-side-apply` manager. Because the field is
still owned by at least one manager, it is not removed from the underlying object.

If the user was to now run `kubectl apply -f <path/to/file> --server-side`, the expected fields would
look like the following:

```console
apiVersion: v1
kind: ConfigMap
data:
  key: value
  legacy: unused
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","data":{"key":"value"},"kind":"ConfigMap","metadata":{"name":"test","namespace":"default"}}
  managedFields:
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:data:
        .: {}
        f:key: {}
        f:legacy: {}
      f:metadata:
        f:annotations:
          .: {}
          f:kubectl.kubernetes.io/last-applied-configuration: {}
    manager: kubectl
    operation: Apply
  name: test
  namespace: default
```

The fields of `kubectl` and `kubectl-client-side-apply` have been unioned and merged
into the single `Apply` entry of `kubectl`.

### Approach

Whenever the migration is to be performed, kubectl fetches the object and simply
calls the recently added client-go library function [`UpgradeManagedFields`](https://github.com/kubernetes/kubernetes/pull/111967/files#diff-4538195db8472f5237db69d0424dfd6fd7a7b0232f67f67dae52f57aea7b1af1R53) to
migrate the managed fields to their desired form. The changes can then be
sent as a PATCH or UPDATE to the apiserver to install the change.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
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

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
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

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Graduation Criteria

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

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
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


- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [x] Other
  - Describe the mechanism: The environment variable `KUBECTL_ENABLE_CSA_MIGRATION=true`.
  - Will enabling / disabling the feature require downtime of the control
    plane? No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled). No.

###### Does enabling the feature change any default behavior?

No this feature proposes a new operation to be added to kubectl, so any
functionality is strictly "opt-in".

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

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

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->


### Server-Side Solution

There is an argument to be made in favor of performing this migration on the
server-side upon the initial call to `kubectl apply --server-side` for a
previously csa'd object.

There are a few problems with this approach:

1.) **Existing users need a way out**: There are users today which have corrupted
managed fields which are owned by both client-side-apply and server-side-apply. These
users need a path out of this situation that does not require so much manual effort.

2.) **It can be solved clinet-side**: Since this problem can be solved client-side,
it is far less risk to implement the solution from kubectl if the solution ended up
being wrong.

### Separate Migration Command

It was proposed early in this KEP to provide a separate command such as:

```shell
kubectl migrate pod <name> --from <csamanager> --to <ssamanager>
```

This was decided against during SIG meetings for the following reasons:

1. Users must learn new command
2. Much more bureaucracy towards adding/deprecating new command
3. Should not add new top-level CLI for command which is only to be used temporarily

### kubectl Plugin

To avoid making direct changes to kubectl, we can instead offer a plugin with this proposed functionality for users migrating to SSA to install with krew.

Pros
1. Can be maintained by API-machinery
2. Easy to deprecate once client-side-apply is no longer used, and migration is not necessary (users just stop installing plugin)
3. No extra complexity in kubectl

Cons
1. Users must perform extra step to use functionality
2. Low discoverability for something a huge portion of users will need to do

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
