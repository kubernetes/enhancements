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
# KEP-2249: Namespace Selector For Pod Affinity

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
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Performance](#performance)
    - [Security/Abuse](#securityabuse)
    - [External Dependencies](#external-dependencies)
- [Design Details](#design-details)
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
- [Alternatives](#alternatives)
- [Implementation History](#implementation-history)
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
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
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

By default, pod affinity/anti-affinity constraints are calculated against 
the pods in the same namespace. The Spec allows users to expand that via
the `PodAffinityTerm.Namespaces` list. 

This KEP proposes two related features: 
1. Adding NamespaceSelector to `PodAffinityTerm` so that users 
   can specify the set of namespace using a label selector.
1. Adding a new quota scope named `CrossNamespaceAffinity` that allows operators
   to limit which namespaces are allowed to have pods that use affinity/anti-affinity
   across namespaces. 

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

## Motivation

Pod affinity/anti-affinity API allows using this feature across namespaces using 
a static list, which works if the user knowns the namespace names in advance.

However, there are cases where the namespaces are not known beforehand, for which there
is no way to use pod affinity/anti-affinity. Allowing users to specify the set of namespaces
using a namespace selector addresses this problem.

Since NamespaceSelector doubles down on allowing cross-namespace pod affinity, giving
operators a knob to control that is important to limit the potential abuse of this feature
as described in the risks section.


<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals
- Allow users to dynamically select the set of namespaces considered when using pod
  affinity/anti-affinity
- Allow limiting which namespaces can have pods with cross namespace pod affinity

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

<!--
### Non-Goals

What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1
I am running a SaaS service, the workload for each customer is placed in a separate
namespace. The workloads requires 1:1 pod to node placement. I want to use pod 
anti-affinity across all customers namespaces to achieve that.

<!--

### Notes/Constraints/Caveats (Optional)

What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

#### Performance
Using namespace selector will make it easier for users to specify affinity 
constraints across a large number of namespaces. The initial implementation 
of pod affinity/anti-affinity suffered from performance challenges, however 
over the releases 1.17 - 1.20 we significantly improved that. 

We currently have integration and clusterloader benchmarks that evaluate the extreme
cases of all pods having affinity/anti-affinity constraints to/against each other.
Those benchmarks show that the scheduler is able to achieve maximum throughput 
(i.e., the api-server qps limit).

#### Security/Abuse
See previous discussion [here](https://github.com/kubernetes/kubernetes/issues/68827#issuecomment-468470598)
NamespaceSelector will allow users to select all namespaces. This may cause a security 
concern: a pod with anti-affinity constraint can block pods from all other namespaces 
from getting scheduled in a failure domain.

We will address this concern by introducing a new quota scope named `CrossNamespaceAffinity`
that operators can use to limit which namespaces are allowed to have pods with affinity terms
that set the existing `namespaces` field or the proposed one `namespaceSelector`. 

Using this new scope, operators can prevent certain namespaces (`foo-ns` in the example below) 
from having pods that use cross-namespace pod affinity by creating a resource quota object in
that namespace with `CrossNamespaceAffinity` scope and hard limit of 0:

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: disable-cross-namespace-affinity
  namespace: foo-ns
spec:
  hard:
    pods: "0"
  scopeSelector:
    matchExpressions:
    - scopeName: CrossNamespaceAffinity
```

If operators want to disallow using `namespaces` and `namespaceSelector` by default, and 
only allow it for specific namespaces,  they could configure `CrossNamespaceAffinity` 
as a limited resource by setting the kube-apiserver flag --admission-control-config-file
to the path of the following configuration file:

```yaml
apiVersion: apiserver.config.k8s.io/v1
kind: AdmissionConfiguration
plugins:
- name: "ResourceQuota"
  configuration:
    apiVersion: apiserver.config.k8s.io/v1
    kind: ResourceQuotaConfiguration
    limitedResources:
    - resource: pods
      matchScopes:
      - scopeName: CrossNamespaceAffinity
```

With the above configuration, pods can use `namespaces` and `namespaceSelector` only
if the namespace where they are created have a resource quota object with 
`CrossNamespaceAffinity` scope and a hard limit equal to the number of pods that are
allowed to.

Moreover, to prevent accidentally selecting a large number of namespaces, we will reject empty
selectors. For example, users can do the following:

```yaml
podAntiAffinity:
  requiredDuringSchedulingIgnoredDuringExecution:
  - namespaceSelector:
      matchExpressions:
        - key: workload
          operator: In
          values:
          - HPC
```

but can't do the following:

```yaml
podAntiAffinity:
  requiredDuringSchedulingIgnoredDuringExecution:
  - namespaceSelector: {}
```

For more protection, admission webhooks like gatekeeper can be used to further
restrict the use of this field.

#### External Dependencies
We are aware of two k8s projects that could be impacted by this change, the descheduler 
and cluster autoscaler. Cluster autoscaler should automatically consume the change
since it imports the scheduler code, the descheduler however doesn't and needs to 
be changed to take this feature into account. We will open an issue to inform the project
about this update.


<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

 Add `NamespaceSelector` field to `PodAffinityTerm`:

```go
type PodAffinityTerm struct {
    // A label query over the set of namespaces that the term applies to.
    // The term is applied to the union of the namespaces selected by this field
    // and the ones listed in the namespaces field.
    // nil selector and empty namespaces list means "this pod's namespace"
    // An empty selector ({}) is not valid.
    NamespaceSelector *metav1.LabelSelector
}
```

As indicated in the comment, the scheduler will consider the union of the namespaces
specified in the existing `Namespaces` field and the ones selected by the new 
`NamespaceSelector` field. `NamespaceSelector`  is ignored when set to nil.

We will do two precomputations at PreFilter/PreScore:

- The names of the namespaces selected by the `NamespaceSelector` is computed. This set
  will be used by Fliter/Score to match against existing pods namespaces.
- A snapshot of the labels of the namespace of the incoming pod. This will be used when 
  to match against the anti-affinity constraints of existing pods. 

The precomputations are necessary for:

- Performance.
- Ensures a consistent behavior if namespace labels are added/removed during 
  the scheduling cycle of a pod.

Finally, the feature will be guarded by a new feature flag. If the feature is 
disabled, the field `NamespaceSelector` is preserved if it was already set in
the persisted Pod ojbect, otherwise it is silently dropped; moreover kube-scheduler 
will ignore the field and continue to behave as before.


With regards to adding the `CrossNamespaceAffinity` quota scope, the one design aspect
worth noting is that it will be rolled out in multiple releases similar to the `NamespaceSelector`:
when the feature is disabled, the new value will be tolerated in updates of objects
already containing the new value, but will not be allowed to be added on create or update.

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Test Plan

- Unit and integration tests covering:
  - core changes
  - correctness for namespace addition/removal, specifically label updates 
    should be taken into account, but not in the middle of a scheduling cycle
  - feature gate enabled/disabled
- Benchmark Tests: 
  - evaluate performance for the case where the selector matches a large number of pods 
    in large number of namespaces

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

#### Alpha -> Beta Graduation

- Benchmark tests showing no performance problems 
- No user complaints regarding performance/correctness.

#### Beta -> GA Graduation
    
- Still no complaints regarding performance.
- Allowing time for feedback

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

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include 
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
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

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: PodAffinityNamespaceSelector
    - Components depending on the feature gate: kube-scheduler, kube-api-server
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  No.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes. One caveat is that pods that used the feature will continue to have
  the NamespaceSelector field set even after disabling, however kube-scheduler
  will not take the field into account.

* **What happens if we reenable the feature if it was previously rolled back?**
 It should continue to work as expected.

* **Are there any tests for feature enablement/disablement?**
  No, but we can do manual testing.


### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  It shouldn't impact already running workloads. This is an opt-in feature since
  users need to explicitly set the NamespaceSelector parameter in the pod spec,
  if the feature is disabled the field is preserved if it was already set in the
  presisted pod object, otherwise it is silently dropped.

* **What specific metrics should inform a rollback?**
  - A spike on metric `schedule_attempts_total{result="error|unschedulable"}`
    when pods using this feature are added.
  - A spike on `plugin_execution_duration_seconds{plugin="InterPodAffinity"}`.


* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
No, will be manually tested.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  No.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  The operator can query pods with the NamespaceSelector field set in pod affinity terms.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [x] Metrics
    - Component exposing the metric: kube-scheduler
      - Metric name: `pod_scheduling_duration_seconds`
      - Metric name: `plugin_execution_duration_seconds{plugin="InterPodAffinity"}`
      - Metric name: `schedule_attempts_total{result="error|unschedulable"}`
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  - 99% of pod scheduling latency is within x minutes
  - 99% of `InterPodAffinity` plugin executions are within x milliseconds
  - x% of `schedule_attempts_total` are successful

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
No.

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
No


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
No. 

* **Will enabling / using this feature result in introducing new API types?**
No.

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
No.

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
Yes, if users set the NamespaceSelector field.

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
May impact scheduling latency if the feature was used.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
If NamespaceSelector field is set, then the scheduler will have to process that
which will result in some increase in CPU usage.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

Running workloads will not be impacted, but pods that are not scheduled yet will
not get assigned nodes.

* **What are other known failure modes?**
N/A

* **What steps should be taken if SLOs are not being met to determine the problem?**
N/A

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos


## Alternatives
Another alternative is to limit the api to "all namespaces" using a dedicated flag
or a special token in Namespacces lis, like "*" (see [here](https://github.com/kubernetes/kubernetes/issues/43203#issuecomment-287237992) for previous discussion).

While this limits the api surface, it makes the api slightly messy and limits 
use cases where only a select set of namespaces needs to be considered. Moreover,
a label selector is consistent with how pods are selected in the same api.

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Implementation History
 - 2021-01-11: Initial KEP sent for review

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

<!--

## Drawbacks

Why should this KEP _not_ be implemented?
-->




<!--

## Infrastructure Needed (Optional)

Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
