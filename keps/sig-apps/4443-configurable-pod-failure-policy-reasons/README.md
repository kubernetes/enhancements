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
# KEP-4443: More granular Job failure reasons for PodFailurePolicyRule

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
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
    - [Defaulting](#defaulting)
    - [Validation](#validation)
    - [Business logic](#business-logic)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes to extend the Job API by adding an optional `Name` field to `PodFailurePolicyRule`. If unset, it would default to the index of
the rule in the `podFailurePolicy.rules` slice. 

The purpose of giving the rule a name is to expose more detailed failure information inside the
`JobFailed` condition reason. When a pod failure policy rule triggers a Job failure, the rule name would be appended as a suffix to the `JobFailed` condition reason, in the
format: `PodFailurePolicy_{ruleName}`. This will allow users to set multiple pod failure policy rules
and distinguish which one (if any) triggered a Job failure.

## Motivation

Higher level K8s APIs are built via a composition of features, using primitive K8S APIs as building blocks to implement more advanced features.
These higher level APIs using the Job API as a building block need to be able to distinguish between different types of Job failures in order to
make informed decisions about how to react to these failures.

Currently, no mechanism exists in the Job API to propagate granular failure reason information (e.g., container exit codes) up to be 
programmatically consumed by higher level software managing Jobs. A `PodFailurePolicy` can be configured to add a condition reason of `PodFailurePolicy`
to the `JobFailed` condition added to the Job when it fails, but different pod failure policies targeting different container exit codes all use the
same condition reason of `PodFailurePolicy`. This prevents higher level APIs like JobSet from distinguishing them and being able to take different
actions depending on the type of Job failure that occurred.

For a concrete use case, see the JobSet [Configurable Failury Policy KEP](https://github.com/kubernetes-sigs/jobset/pull/381) which illuminated the need for more granular pod failure policy reasons.

### Goals

For pod failure policies to be able communicate different failure types to higher level APIs.

### Non-Goals

- Modifying PodFailurePolicy behavior
- The Job controller using this new field for any purpose not explicitly defined in the proposal.

## Proposal

The proposal is to add an optional `Name` field to the `PodFailurePolicyRule`. If unset, it will default to the index of the  `PodFailurePolicyRule` in the `PodFailurePolicy.Rules` slice.

When a `PodFailurePolicyRule` matches a pod failure and the `Action` is `FailJob`, the Job
controller will append the name of the pod failure policy rule which triggered the failure
to the JobFailed [condition](https://github.com/kubernetes/kubernetes/blob/6a4e93e776a35d14a61244185c848c3b5832621c/pkg/controller/job/job_controller.go#L816)
reason. The exact format of the JobFailed condition reason will be `PodFailurePolicy_{ruleName}`.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1
As a user, I am using a JobSet to manage a group of jobs, and I want to be able to decide whether to fail the
JobSet or not, based on the exact container exit code that caused a child job failure.

**Example JobSet for this use case**:

```yaml
apiVersion: jobset.x-k8s.io/v1alpha2
kind: JobSet
metadata:
  name: fail-jobset-example
spec:
  failurePolicy:
    rules:
    # If Job fails due to a pod failing with exit code 2, fail the JobSet immediately, without attempting any restarts.
    - action: FailJobSet
      targetReplicatedJobs:
      - workers
      onJobFailureReasons:
      - PodFailurePolicy_ExitCode2 # Job failure reason format: PodFailurePolicy_{ruleName}
    maxRestarts: 10
  replicatedJobs:
  - name: workers
    replicas: 10
    template:
      spec:
        parallelism: 1
        completions: 1
        backoffLimit: 0
        # If a pod fails with exit code 2, fail the job with the user-defined reason.
        podFailurePolicy:
          rules:
          - name: "ExitCode2"  # Will be added as a suffix to the reason "PodFailurePolicy" condition reason.
            action: FailJob
            onExitCodes:
              containerName: main
              operator: In
              values: [2]
        template:
          spec:
            restartPolicy: Never
            containers:
            - name: main
              image: python:3.10
              command: ["..."]
```

#### Story 2

As a user, I am using a JobSet to manage a group of jobs, each running a HPC simulation.
Each job runs a simulation with different random initial parameters. When a simulation ends, the
application will exit with one of two exit codes:

- Exit code 2, which indicates the simulation produced an invalid result due to bad starting parameters, and should
not be retried.
- Exit code 3, which indicates the simulation produced an invalid result but the initial parameters were reasonable,
so the simulation should be restarted.

When a Job fails due to a pod failing with exit code 2, I want my job management software to leave the Job in
a failed state.

When a Job fails due to a pod failing with exit code 3, I want my job management software to to restart the Job.

**Example JobSet for this use case**:

```yaml
apiVersion: jobset.x-k8s.io/v1alpha2
kind: JobSet
metadata:
  name: restart-job-example
  annotations:
    alpha.jobset.sigs.k8s.io/exclusive-topology: {{topologyDomain}} # 1:1 job replica to topology domain assignment
spec:
  failurePolicy:
    rules:
    # If Job fails due to a pod failing with exit code 2, leave it in a failed state.
    - action: FailJob
      targetReplicatedJobs:
      - simulations
      onJobFailureReasons:
      - PodFailurePolicy_ExitCode2  # Job failure reason format: PodFailurePolicy_{ruleName}
    # If Job fails due to a pod failing with exit code 3, restart that Job.
    - action: RestartJob
      targetReplicatedJobs:
      - simulations
      onJobFailureReasons:
      - PodFailurePolicy_ExitCode3  # Job failure reason format: PodFailurePolicy_{ruleName}
    maxRestarts: 10
  replicatedJobs:
  - name: simulations
    replicas: 10
    template:
      spec:
        parallelism: 1
        completions: 1
        backoffLimit: 0
        # Pod failure policy rules, the names of which are referenced in the JobSet failure policy.
        podFailurePolicy:
          rules:
          - name: ExitCode2
            action: FailJob
            onExitCodes:
              containerName: main
              operator: In
              values: [2]
          - name: ExitCode3
            action: FailJob
            onExitCodes:
              containerName: main
              operator: In
              values: [3]
        template:
          spec:
            restartPolicy: Never
            containers:
            - name: main
              image: python:3.10
              command: ["..."]
```

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->
There is a risk to making a field that was previously exclusively managed by the controller,
to now being configurable by the user. However, as described in the validation section below,
we are validating against malformed/invalid inputs.


## Design Details

#### Defaulting
If unset, the `Name` field will default to the index of the pod failure policy rule in the `Rules` slice.

#### Validation
- Validate all pod failure policy rule names are unique.
- We will validate the Job failure conditionr reason that would be generated from a given
PodFailurePolicy rule name (i.e., `PodFailurePolicy_{ruleName}`) will be a valid reason (CamelCase, max length of 128 characters, and matches the regex defined [here](https://github.com/kubernetes/kubernetes/blob/dd301d0f23a63acc2501a13049c74b38d7ebc04d/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L1559)).
- We will also validate the pod failure policy rule does not conflict with any [K8s internal reasons used by the Job controller](https://github.com/kubernetes/kubernetes/blob/862ff187baad9373d59d19e5d736dcda1e25e90d/staging/src/k8s.io/api/batch/v1/types.go#L542-L553). 

#### Business logic
When a `PodFailurePolicyRule` matches a pod failure and the `Action` is `FailJob`, the Job controller will
set the JobFailed condition reason deterministically in the format `PodFailurePolicy_{ruleName}`. This 
suffix will be added to the condition [condition](https://github.com/kubernetes/kubernetes/blob/6a4e93e776a35d14a61244185c848c3b5832621c/pkg/controller/job/job_controller.go#L816) here.

Note: If the PodFailurePolicy feature gate is disabled, but the `PodFailurePolicyName` feature gate is enabled, there will be 
no adverse effect and neither feature will actually be used, since the only place the proposed new field `Name` will be
used is inside of a [code block](https://github.com/kubernetes/kubernetes/blob/6a4e93e776a35d14a61244185c848c3b5832621c/pkg/controller/job/job_controller.go#L816) protected by the PodFailurePolicy feature gate.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
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

- `k8s.io/kubernetes/pkg/controller/job`: `02/05/2024` - `91.5%`

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

<!-- - <test>: <link to test coverage> -->
- Test that when the feature flag is enabled and a Job's PodFailurePolicy triggers a Job failure, due to a matching PodFailurePolicyRule, check that the `JobFailed` condition has a reason of `PodFailurePolicy_{Name}`.

- Test that when the feature flag is off, but when it was previously enabled, there is an existing Job
which already had the `JobFailed` condition reason set with the new suffix (i.e., `PodFailurePolicy_{Name}`), that the Job controller does not overwrite the reason to `PodFailurePolicy`, and that it remains set to the existing value.

- Add test cases for both onPodConditions and onExitCodes to ensure the `Name` is properly added
as a suffix to the JobFailed reason upon Job failure.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

<!-- - <test>: <link to test coverage> -->
We will a test case similar to the integration test case:

- When the feature flag is enabled and a Job's PodFailurePolicy triggers a Job failure, due to a
matching PodFailurePolicyRule, check that the `JobFailed` condition has the `PodFailurePolicy_{Reason}` reason set correctly.

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
- Initial unit and integration tests are implemented

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
After a user upgrades their cluster to a k8s version which supports this feature, 
the user can use this feature by simply specifying the new field in their podFailurePolicy
config.

When a user downgrades from a k8s version that supports this field to one that does
not support this field:
- for existing Jobs, this new field will be ignored by the Job controller,
resulting in the condition reason being set to the previous default of `PodFailurePolicy`
for any Job failures triggered by a pod failure policy.
- for new Jobs, the kube-apiserver would remove this field when the Job is submitted.

### Version Skew Strategy

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

N/A. This feature doesn't require coordination between control plane components,
the changes to the Job controller are self-contained. 

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
- Upgrade to k8s version 1.31+
- Enable feature flag `PodFailurePolicyName`

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `PodFailurePolicyName`
  - Components depending on the feature gate:
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
No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
Yes, by disabling the feature flag `PodFailurePolicyName`.

###### What happens if we reenable the feature if it was previously rolled back?

For new Jobs, the apiserver will stop wiping out the new field.
For existing Jobs, the Job controller will stop ignoring the new field, and begin
using it as described in previous sections.

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
We can add unit tests for:
- feature enabled and field set
- feature disabled and field set

### Rollout, Upgrade and Rollback Planning


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
No

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
If the optional `name` field is specified, the podFailurePolicy object size will increase by 1 byte per
character in the `name` string. The name field will be no longer than 128 characters, thus the max size
increase of the PodFailurePolicy object will be 128 bytes.
Otherwise, if unset, it will default to the index of the rule, and thus increase by 1 byte per digit in
the index number.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->
No

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
- KEP Published: 02/05/2024

## Drawbacks

It is a less intuitive user experience to have the PodFailurePolicy rule name be appended
as a suffix to the Job failure reason, rather than 

## Alternatives

We discussed the idea of having a new optional `PodFailurePolicyRule` field `SetConditionReason`, which will enable
the user to explicitly set the condition reason they want on the JobFailed condition set on the Job when that pod failure
policy rule triggers a Job failure. However, ultimately it was decided we didn't want to open up the reason field to be
explicitly set by the user to any arbitrary value, as this would be tricky to validate, and would diverge from the current
paradigm of having only machine set reasons which are determined programatically.


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
