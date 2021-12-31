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

# KEP-3101: 1:1 pod to node assignment

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
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Possible Mitigations](#possible-mitigations)
- [Design Details](#design-details)
  - [API](#api)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
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
  - [Define a NodeSelector/NodeAffinity Constraint](#define-a-nodeselectornodeaffinity-constraint)
  - [Use resource requirments](#use-resource-requirments)
  - [Use pod anti affinity](#use-pod-anti-affinity)
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

Items marked with (R) are required *prior to targeting to a milestone /
release*.

-   [ ](R) Enhancement issue in release milestone, which links to KEP dir in
    [kubernetes/enhancements](not the initial KEP PR)
-   [ ](R) KEP approvers have approved the KEP status as `implementable`
-   [ ](R) Design details are appropriately documented
-   [ ](R) Test plan is in place, giving consideration to SIG Architecture and
    SIG Testing input (including test refactors)
    -   [ ] e2e Tests for all Beta API Operations (endpoints)
    -   [ ](R) Ensure GA e2e tests for meet requirements for
        [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
    -   [ ](R) Minimum Two Week Window for GA e2e tests to prove flake free
-   [ ](R) Graduation criteria is in place
    -   [ ](R)
        [all GA Endpoints](https://github.com/kubernetes/community/pull/1806)
        must be hit by
        [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
-   [ ](R) Production readiness review completed
-   [ ](R) Production readiness review approved
-   [ ] "Implementation History" section is up-to-date for milestone
-   [ ] User-facing documentation has been created in [kubernetes/website], for
    publication to [kubernetes.io]
-   [ ] Supporting documentation—e.g., additional design documents, links to
    mailing list discussions/SIG meetings, relevant PRs/issues, release notes

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

This Kep proposes adding a feature (`CoexistPolicy`) that provides users with an
explicit API and scheduler support for 1:1 pod to node assignment.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Customers trying to deploy batch workloads on Kubernetes have filed multiple
requests (See [1](https://github.com/kubernetes/kubernetes/issues/68827),
[2](https://github.com/kubernetes/kubernetes/issues/105287) for details) for 1:1
pod-to-node assignment. The most common use case for enforcing 1:1 pod to node
assignment is HPC workloads (MPI and Spark).

Currently there are two approaches to achieve this:

-   Set resource requests to something high enough to prevent scheduling any
    other pods but daemonset and static pods. This is tricky to do and error
    prone, especially for smaller nodes. This will not be possible if node size
    is not known in advance.
-   Use pod anti affinity. Users will need to define the same set of labels on
    all pods that needs to be repelled, whether or not they actually request 1:1
    pod to node assignment, it is also expensive to calculate especially for
    cluster autoscaler.

A simple and native way to force 1:1 pod-to-node assignment is therefore needed.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

-   Allow users to force 1:1 pod-to-node assignment for a given pod.
    -   This is enforced by both scheduler and kubelet.
    -   This feature is implemented as a filter, and so features associated with
        filters applies to it like preemption and autoscaling.

### Non-Goals

-   Find the best fit node (a.k.a `Score`) from the nodes that satisfy
    `pod.spec.CoexistPolicy`
-   Change node ranking when selecting a candidate for preemption.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

Add a new pod-level API that allows users to enforce how a pod can coexist with
other pods when scheduled on a node. Available policies are:

-   `DaemonsetAndStaticPods`: pod only tolerates daemonset and static pods to be
    running on the node.
-   `Any`: pod tolerates any pod type to be running on the node (default)

The API is implemented by the scheduler as a filter plugin and kubelet as an
admission predicate.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

A user wants a Pod to be scheduled alone on a node to avoid them from contending
on resources (e.g. claiming all CPU cores available on a node). In other words,
resource intensive workloads are needed to be scheduled on separate nodes.

Example: As a user, I want to deploy a Spark or MPI job such that only a single
worker pod is running on each node.

#### Story 2

A DBaaS running natively on Kubernetes which uses 1:1 Pod to Node assignment
where Nodes are sized appropriately for said pod's resource requirements. A
combination of `nodeSelector`/`node affinity` with 1:1 pod to node assignment
can be used to achieve this goal.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

Users might be able to circumvent 1:1 pod to node assignments by creating a
daemonset with node affinity to a specific node.

#### Possible Mitigations

-   Cluster admins can use one of the following to prevent users from creating
    daemonsets
    -   [ResourceQuota](https://kubernetes.io/docs/concepts/policy/resource-quotas/)
        -   Object count quota (`count/daemonsets`) can be used for this
            purpose.
    -   kubernetes
        [RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) to
        restrict daemonsets creation to admins/specific users.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

The idea is to create a scheduler filter plugin such that given a pod and a
node, a node is filtered out (as in considered no fit for the pod) if any of the
following is true (assume `CoexistPolicy` is set to `DaemonsetAndStaticPods` in
the incoming Pod's podspec):

-   The incoming pod is not a daemonset pod or static pod (by checking
    OwnerReference) and there is a pod with
    `CoexistPolicy`=`DaemonsetAndStaticPods` already assigned to the node,
-   If the incoming pod has `CoexistPolicy`=`DaemonsetAndStaticPods` and the
    node hosts at least one regular pod (neither daemonset pod nor static pod).
-   The above logic would also be added to the set of predicates that kubelet
    tests on pod admission. This will prevent users from working around the
    scheduler to avoid coexist policy enforcement.
-   If no Node was found that satisfies the filter for the incoming pod,
    [Pod Preemption](https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/#preemption)
    may evict lower priority pods to satisfy the coexist policy of the incoming
    pod.

### API

A new API field `pod.spec.coexistPolicy` is introduced to be used by the new
filter. The value for the newly introduced field is an enum that has one of the
following values: `DaemonsetAndStaticPods`, `Any` (Default) where `Any` is the
same as the current behaviour and `DaemonsetAndStaticPods` would allow placing
only a single pod on a node excluding daemonset and static pods, effectively
enforcing 1:1 workload pod to node mapping.

```yaml
...
spec:
  coexistPolicy: <string>
```

This API will look as follows in Golang

```go
type PodSpec struct {
  // CoexistPolicy specifies which other pods this pod can "coexist" with. It is similar to pod anti-affinity but
  // limited to node topology and doesn't operate on pod labels, and so more performant but has narrow use
  // cases defined by the hardcoded policies.
  CoexistPolicy CoexistPolicy
}

// +enum
type CoexistPolicy string

const (
    // Pod tolerates co-existing with other pods.
    CoexistPolicyAny CoexistPolicy = "Any"
    // Pod tolerates coexisting with daemonset and static pods only.
    CoexistPolicyDaemonsetAndStaticPods CoexistPolicy = "DaemonsetAndStaticPods"
)
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

Unit and integration tests verifying the following

-   a single pod is assigned to a node while allowing daemonsets and static
    pods.
-   scheduler doesn't schedule new pods to a node that already have a pod with
    `DaemonsetAndStaticPods` policy.
-   Daemonset and static pods can be scheduled to a node that has a pod with
    `DaemonsetAndStaticPods` policy.
-   kubelet rejects a pod if it was assigned directly to a node already running
    a pod with `DaemonsetAndStaticPods` policy
-   the API is no-op if feature gate is turned off

Benchmark Tests

-   Verify that performance is better than using inter-pod anti-affinity
-   Feature shouldn't impose performance overhead. We will verify by designing
    some benchmark tests.

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

#### Alpha -> Beta Graduation

-   Unit and integration tests passing.
-   Feature implemented behind a feature flag.
-   Benchmark tests showing no performance problems.
-   No user complaints regarding performance/correctness.

#### Beta -> GA Graduation

-   Still no complaints regarding performance.
-   Ensure feature documentation is clear and complete.

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

In the event of a downgrade, the coexistPolicy will be ignored even if it was
set to `DaemonsetAndStaticPods`.

More specifically

-   While the feature flag is disabled, the field will be ignored by the
    api-server and will not be stored in etcd.
-   Not setting the new field will maintain the old behavior on both upgrade and
    downgrade of the control plane.
-   Setting the field after upgrading and while the feature flag is enabled will
    allow users to make use of the feature.
-   Downgrading after upgrade or disabling the feature flag will
    -   maintain previously set values in the field on existing pods, but they
        will have no effect.
    -   disable the scheduler filter, and so the cluster may end up in a state
        where there are pods with `DaemonsetAndStaticPods` previously assigned
        to nodes, those nodes after the downgrade will now be allowed to fit
        other pods that wouldn't have fit while the feature is enabled. This
        situation will not be fixed after a subsequent upgrade, node drain is
        required to fix that.

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

If Kubelet is running a version older than the control plane, and the version
has the feature disabled, the predicate will not be enforced on pod admission.

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

-   [x] Feature gate (also fill in values in `kep.yaml`)
    -   Feature gate name: `CoexistPolicy`
    -   Components depending on the feature gate: kube-scheduler,
        kube-api-server
-   [ ] Other
    -   Describe the mechanism:
    -   Will enabling / disabling the feature require downtime of the control
        plane?
    -   Will enabling / disabling the feature require downtime or reprovisioning
        of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior? No.

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)? Yes.

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back? It should continue to work as expected.

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

It should not impact already running workloads. This is an opt-in feature since
users need to explicitly set the API feature in pod spec. If the feature is
disabled, this feature will be silently ignored even if it's already set.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

-   A spike on metric schedule_attempts_total{result="error|unschedulable"} when
    pods using this feature are added.
-   Metric plugin_execution_duration_seconds{plugin="CoexistPolicy"} larger than
    100ms on 90-percentile.
-   A spike on failure events with keyword "failed CoexistPolicy" in scheduler
    log.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No. This will be done upon beta graduation.

<!-- Describe manual testing that was done and the outcomes. Longer term, we may want to require automated upgrade/rollback tests, but we are missing a bunch of machinery and tooling and can't do that now. -->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

Operator can query `pod.spec.coexistPolicy` field and identify if this is being
set to non-default values. Also non-zero value of metric
plugin_execution_duration_seconds{plugin="CoexistPolicy"} is a sign indicating
this feature is in use.
<!-- Ideally, this should be a metric. Operations against the Kubernetes API (e.g., checking if there are objects with field X set) may be a last resort. Avoid logs or events for this purpose. -->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

For pods with `CoexistPolicy` field set to `DaemonsetAndStaticPods`, check that
it is the only pod running on the nodes other than daemonset and static pods.

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

-   Metric plugin_execution_duration_seconds{plugin="CoexistPolicy"} <= 100ms on
    90-percentile.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

-   [x] Metrics
    -   Metric name: `plugin_execution_duration_seconds{plugin="CoexistPolicy"}`
    -   [Optional] Aggregation method:
    -   Components exposing the metric:
-   [ ] Other (treat as last resort)
    -   Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster? No.

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

No. <!-- Describe them, providing: - Which API(s): - Estimated increase: -->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.
<!-- Describe them, providing: - API type(s): - Estimated increase in size: (e.g., new annotation of size 32B) - Estimated amount of new objects: (e.g., new Object X for every existing Pod) -->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. <!-- Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc. This
through this both in small and large cases, again with respect to the [supported
limits].

\[supported limits]:
https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

No new pods will schedule whether or not the feature is used.

###### What are other known failure modes?

None.
<!-- For each of them, fill in the following information by copying the below template: - [Failure mode brief description] - Detection: How can it be detected via metrics? Stated another way: how can an operator troubleshoot without logging into a master or worker node? - Mitigations: What can be done to stop the bleeding, especially for already running user workloads? - Diagnostics: What are the useful log messages and their required logging levels that could help debug the issue? Not required until feature graduated to beta. - Testing: Are there any tests for failure mode? If not, describe why. -->

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

2022-01-17: Initial KEP.

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

### Define a NodeSelector/NodeAffinity Constraint

A nodeSelector/nodeAffinity constraint can be used to achieve this 1:1 mapping,
something similar to:

```yaml
spec:
  nodeSelector:
    pod-per-node: true
```

Or

```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
      - matchExpressions:
        - key: pod-per-node
          operator: Exists
```

The new filter will look for this selector/affinity rule and filter out nodes
that has pods other than daemonset and static pods. However, `NodeAffinity`
filter will still evaluate this selector, and it will be looking for the labels
on the nodes as well.

This option is obscure with edge cases and can cause confusion to users

-   NodeAffinity filter might need to be changed to ignore the node labels for
    this particular case (nodes will not be required to have a `pod-per-node`
    label)

    -   otherwise all nodes will need to have a `pod-per-node` label in order to
        pass the affinity validation.

-   If this is set in nodeAffiniy, OR/AND logic of affinity constraints will
    need to be supported/handled?

### Use resource requirments

Set resource requests to something high enough to prevent scheduling any other
pods but daemonset pods.

-   This is tricky to do and error prone, especially for smaller nodes.
-   This will not be possible if node size is not known in advance.

### Use pod anti affinity

Inter-pod anti-affinity can be used to enforce mapping a single pod to force 1:1
pod to node mapping.

-   users will need to define the same set of labels on all pods that needs to
    be repelled, whether or not they actually request 1:1 pod to node
    assignment.
-   it is expensive to calculate especially for cluster autoscaler.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
