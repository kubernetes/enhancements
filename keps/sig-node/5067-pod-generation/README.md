<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

* [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
* [x] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
* [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
`NNNN-short-descriptive-title` , where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
* [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig", 
  "Status", and date-related fields.
* [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
* [x] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
* [x] **Merge early and iterate.**
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

**Note:** Any PRs to move a KEP to `implementable` , or significant changes once
it is marked `implementable` , must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->

# KEP-5067: Pod Generation

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
  <code>&lt; !-- toc --&rt; &lt; !-- /toc --&rt; </code>
tags, and then generate with `hack/update-toc.sh` .
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Current behavior](#current-behavior)
  - [API Changes](#api-changes)
    - [Generation](#generation)
    - [ObservedGeneration](#observedgeneration)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Custom-set <code>metadata.generation</code>](#custom-set-metadatageneration)
    - [Infinite loop caused by misbehaving mutating webhooks](#infinite-loop-caused-by-misbehaving-mutating-webhooks)
- [Design Details](#design-details)
    - [API server and generation](#api-server-and-generation)
      - [Client requests to update generation](#client-requests-to-update-generation)
    - [Kubelet and observedGeneration](#kubelet-and-observedgeneration)
      - [Mutable Fields Analysis](#mutable-fields-analysis)
    - [Other writers of pod status](#other-writers-of-pod-status)
    - [Client requests to update observedGeneration](#client-requests-to-update-observedgeneration)
    - [Mirror pods](#mirror-pods)
    - [Future enhancements](#future-enhancements)
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

* [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
* [x] (R) KEP approvers have approved the KEP status as `implementable`
* [x] (R) Design details are appropriately documented
* [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  + [x] e2e Tests for all Beta API Operations (endpoints)
  + [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  + [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
* [x] (R) Graduation criteria is in place
  + [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
* [x] (R) Production readiness review completed
* [x] (R) Production readiness review approved
* [x] "Implementation History" section is up-to-date for milestone
* [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
* [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

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

This proposal aims to allow the pod status to express which pod updates are 
currently being reflected in the pod status. The idea is to leverage the 
existing `metadata.Generation` field and add a new `status.observedGeneration` field to 
the pod status. 

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

One of the motivations for this KEP comes from the existing ResizeStatus field. 
In its original implementation, the ResizeStatus field was written to by both 
the API server and the Kubelet, creating a race condition on updates. Removing 
the Proposed state makes the Kubelet the only writer to that field, but leaves a
gap in knowing whether the latest resize has been acknowledged by the Kubelet. 
The changes proposed in this KEP resolve this gap.

The ResizeStatus is used as an example here, but in practice this issue can be 
generally found in any type of pod update.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

* Provide a general solution for the pod status to express which pod update is 
currently being reflected.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

* Expand the set of mutable fields.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### Current behavior

The pod [ `metadata.generation` ](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata) field does exist today, and is 
documented as "a sequence number representing a specific generation of the 
desired state. Set by the system and monotonically increasing, per-resource." 

Its current behavior in pods is:

* Pod `metadata.generation` is not populated by default by the system.
* The client can custom-set the `metadata.generation` on pod create.
* `metadata.generation` cannot be updated by the client.
* `metadata.generation` does not get incremented by the system when the podspec is updated.
* `metadata.generation` does get incremented by the system when `DeletionTimestamp` is set.

### API Changes

#### Generation

The `metadata.generation` field is currently unused on pods, but we can start 
leveraging it to help track which pod state is currently being reflected in the 
pod status. For consistency, pod 
`metadata.generation` will be incremented whenever the pod has changes that the kubelet
needs to actuate. 

#### ObservedGeneration

A new optional field `status.observedGeneration` field will be added to the pod status.
Kubelet will set this to communicate which pod state is being expressed in the 
current pod status. This is analogous to the `status.observedGeneration` that exists
in other resources' statuses such as [StatefulSets](https://github.com/kubernetes/api/blob/9e7d345b161c12a3056efa88e285b3ef68450c54/apps/v1/types.go#L274).

The `status.observedGeneration` may not necessarily be a reflection of every single
field in the pod status. Instead, it reports the latest generation that the kubelet
has seen. This means that `status.observedGeneration` captures the kubelet's
decision to admit a change, and acknowledge that it has seen the pod's new spec.
There are cases when `status.observedGeneration` may be behind - other status values 
already reflect a next generation, but the next update from kubelet SHOULD bring 
the `status.observedGeneration` to the current value.
It also will not necessarily be able to reflect that the kubelet has completed actuation
of certain fields. There is a field-by-field analysis written up in [this doc](https://docs.google.com/document/d/1-4CR2NmDJotCM13YQyr8OEeGIczlc3WFmV1BhdUrWcw/edit?tab=t.0). We will
have to carefully document the nuanced meaning of `status.observedGeneration` to avoid confusion. 

Likewise, a new optional `observedGeneration` field will be added to the pod's 
`status.condition` struct. This is to keep parity with the 
[metav1. Condition struct](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L1620-L1625). 

The net result will be a new `status.observedGeneration` field for the kubelet
to express which generation the top-level status relates to, and a new `status.conditions[i].observedGeneration` field for the writer of that condition to express which generation that condition
relates to.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

Because there is only one singular `status.observedGeneration` in the pod status, only
one writer can set it. This is consistent with other object types that have
`status.observedGeneration` , and the expectation is that the primary controller for
the object sets the field. For a pod, that primary controller is the kubelet, so
once a pod is bound to a node, we expect that the kubelet on that node is the
sole writer of `status.observedGeneration`. 

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

#### Custom-set `metadata.generation`

Today, it is possible for a client to custom-set a pod's `metadata.generation` on creation, but once
set, the `metadata.generation` cannot be updated. That means that there is a
possibility for an external client to be setting `metadata.generation` on pod 
create and depend on that fixed value somehow in its own reconciliation logic.

That said, the `metadata.generation` field is described as ["set by the system and monotonically increasing"](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata), 
and thus it should be clear enough that `metadata.generation` was not intended to be
used in this way.

#### Infinite loop caused by misbehaving mutating webhooks

It is possible that today there exists mutating webhooks that overwrite a pod's
status. These older webhooks would not know about `status.observedGeneration` and
could be clearing it. This would cause an infinite loop of the kubelet
attempting to update a pod's `status.observedGeneration` and the webhook
clearing it.

Status-mutating webhooks could break more pod features than just what is
proposed in this KEP, so we will not attempt to solve this here. This risk can 
be mitigated by improving the documentation on webhooks and how they can be 
written to avoid these kinds of scenarios.

Symptoms of this scenario that users can look out for include:
- Unexpected sudden spikes in pod status update API calls that occur right after
  either upgrading the kubelet or creating a new status-mutating webhook.
- `status.observedGeneration` remains unchanged after a pod sync loop even
  when `metadata.generation` is changing. This occurs because the API server
  would be preserving any existing value of `status.observedGeneration` whenever
  the webhook attempts to clear it.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

#### API server and generation

For a newly created pod, the API server will set `metadata.generation` to 1. For any updates
to the [PodSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#podspec-v1-core), the API server will increment `metadata.generation` by 1. 

As described in the [field-by-field analysis doc](https://docs.google.com/document/d/1-4CR2NmDJotCM13YQyr8OEeGIczlc3WFmV1BhdUrWcw/edit?tab=t.0) above, the PodSpec 
mutable fields today are:
- Resources
- Ephemeral Containers
- Container image
- ActiveDeadlineSeconds
- TerminationGracePeriodSeconds
- Tolerations

If any new mutable fields are added to the PodSpec in the future, they will also
cause the API server to increment `metadata.generation`. 

The pod `metadata.generation` will also continue to be incremented on graceful delete or 
deferred delete, just as the API server currently does for pods and other 
objects today.

Pod updates that would not result in `metadata.generation` being incremented include:
* Changes to metadata (with the exception of `DeletionTimestamp`). This means 
that if a Pod uses the downward API to make pod metadata available to containers, 
Pod behavior can change without the generation being incremented. We will consider 
this working as intended.
* Changes to status.

The logic to set new pods' `metadata.generation` to 1 and to increment `metadata.generation` 
on update will run after all mutating webhooks have finished. 

##### Client requests to update generation

Any attempts by clients to set or modify the `metadata.generation` field themselves will be
ignored and overridden by the API Server. This is consistent with existing 
behavior of the `metadata.generation` field in all other objects. 

#### Kubelet and observedGeneration

When the Kubelet updates the pod status as part of the [pod sync loop](https://github.com/kubernetes/kubernetes/blob/45d0fddaf1f24f7b559eb936308ce2aeb9871850/pkg/kubelet/pod_workers.go#L1214), 
it will set the `status.observedGeneration` in the pod to reflect the pod `metadata.generation` corresponding 
to the snapshot of the pod currently being synced. That means if the 
pod spec gets updated concurrently while the kubelet is performing a pod sync loop 
on a previous update, the `status.observedGeneration` will be behind `metadata.generation`. 

Outside of the pod sync loop, another place where the kubelet sets the pod status is
when a [pod is rejected (during `HandlePodAdditions`)](https://github.com/kubernetes/kubernetes/blob/8770bd58d04555303a3a15b30c245a58723d0f4a/pkg/kubelet/kubelet.go#L2335). This
code will also be modified to populate `status.observedGeneration` to express which
`metadata.generation` was rejected. In this case, the kubelet will not be updating the pod
through the sync loop (due to the rejection).

The only other place where the pod status is updated is the readiness and probe
updates, but we will leave `status.observedGeneration` unchanged here as the probe that
updated the status would be the one synced in the last pod sync loop.

##### Mutable Fields Analysis

The [field-by-field analysis doc](https://docs.google.com/document/d/1-4CR2NmDJotCM13YQyr8OEeGIczlc3WFmV1BhdUrWcw/edit?tab=t.0) referenced above
goes into further detail about what `status.observedGeneration` means in relation
to the other fieds in the pod status. Here is a summary of the conclusions:
- For some fields, the allocated spec is reflected directly in the pod status, so
  their associated generation is reflected directly by `status.observedGeneration`.
  This is the case for allocated resources, resize status, ephemeral containers.
- For other fields, the status is an indirect result of actuating the PodSpec,
  and the associated generation for those fields are from the generation _before_
  what is reflected by `status.observedGeneration`. This is the case for actual
  resources, container image, activeDeadlineSeconds, and terminationGracePeriodSeconds.

To keep things simple and avoid having to add a new field to track the latter, the
kubelet will output the PodSpec `metadata.generation` that was observed at the time
of the current sync loop, even though the kubelet has not actuated the change yet.
We will document this very clearly and explicitly to avoid confusion.

#### Other writers of pod status

There are other writers of pod status besides the Kubelet including the scheduler
and the node lifecycle controller. These should also populate `status.observedGeneration` whenever 
they make a status update (excluding any updates to pod conditions, which
have their own dedicated `observedGeneration` field). Once the pod is bound to a node, 
however, the expectation is that only the kubelet will be writing to `status.observedGeneration`. 

The scheduler and node lifecycle controller also write pod conditions. Whenever 
they set a pod condition, they should just populate the `condition.observedGeneration` field 
with the relevant generation.

#### Client requests to update observedGeneration

During status update, if the incoming update clears `status.observedGeneration` back
to 0, the API server will preserve the previously existing value. All other updates to `status.observedGeneration` will be permitted by the API validation, including
regressions back to decreasing values.

#### Mirror pods

For this KEP, we will not treat mirror pods in any special way. Due to the way they are currently implemented in the
kubelet and apiserver, this means:

1. If a mirror pod's spec is modified manually by a client via the apiserver, its `metadata.generation` will be bumped accordingly.
1. If a static pod's manifest is updated, the kubelet treats this as a pod deletion followed by a pod creation,
which will reset the `metadata.generation` of the corresponding mirror pod to 1.
1. The kubelet does not currently propagate the mirror pod's `metadata.generation` to the place where
the pod status is updated today, so the `observedGeneration` fields of mirror pods will remain
unpopulated.

#### Future enhancements

We may at some future point reconsider mirror pods and potentially populate
`metadata.generation` and `status.observedGeneration` on them.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
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
* <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

Unit tests will be implemented to cover code changes that implement the feature, 
in the API server code and the kubelet code. 

Core packages touched:
* `pkg/registry/core/pod/strategy.go`: `2025-06-16` - `71.1`
* `pkg/registry/core/pod/util.go`: `2025-06-16` - `74`
* `pkg/apis/core/validation/validation.go`: `2025-06-16` - `84.6`
* `pkg/kubelet`: `2025-06-16` - `71`
* `pkg/kubelet/status`: `2025-06-16` - `86.8`

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

Unit and E2E tests provide sufficient coverage for the feature. Integration tests may be added to cover any gaps that are discovered in the future. 

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

E2E tests will be implemented to cover the following cases:
* Verify that newly created pods have a `metadata.generation` set to 1.
* Verify that PodSpec updates (such as tolerations or container images), resize requests, adding ephemeral containers, and binding requests cause the `metadata.generation` to be incremented by 1 for each update.
* Verify that deletion of a pod causes the `metadata.generation` to be incremented by 1.
* Issue ~500 pod updates (1 every 100ms) and verify that `metadata.generation` and `status.observedGeneration` converge to the final expected value.
* Verify that various conditions each have `observedGeneration` populated. 
* Verify that static pods have `metadata.generation` and `observedGeneration` fields set to 1, and that
they never change.

Added tests:
`pod generation should start at 1 and increment per update`: SIG Node, https://storage.googleapis.com/k8s-triage/index.html?test=Pod%20Generation
`custom-set generation on new pods and graceful delete`: SIG Node, https://storage.googleapis.com/k8s-triage/index.html?test=Pod%20Generation
`issue 500 podspec updates and verify generation and observedGeneration eventually converge`: SIG Node, https://storage.googleapis.com/k8s-triage/index.html?test=Pod%20Generation
`pod rejected by kubelet should have updated generation and observedGeneration`: SIG Node, https://storage.googleapis.com/k8s-triage/index.html?test=Pod%20Generation
`pod observedGeneration field set in pod conditions`: SIG Node, https://storage.googleapis.com/k8s-triage/index.html?test=Pod%20Generation
`pod-resize-scheduler-tests`: SIG Node, https://storage.googleapis.com/k8s-triage/index.html?test=pod-resize-scheduler-tests
`mirror pod updates`: SIG Node, https://storage.googleapis.com/k8s-triage/index.html?test=mirror%20pod%20updates


### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
* [Maturity levels (`alpha`,      `beta`,      `stable`)][maturity-levels]
* [Feature gate][feature gate] lifecycle
* [Deprecation policy][deprecation-policy]

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

* Feature implemented behind a feature flag
* Initial e2e tests completed and enabled

#### Beta

* Gather feedback from developers and surveys
* Complete features A, B, C
* Additional tests are in Testgrid and linked in KEP

#### GA

* N examples of real-world usage
* N installs
* More rigorous forms of testing—e.g., downgrade tests and scalability tests
* Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports, 
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

* Announce deprecation and support policy of the existing flag
* Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
* Address feedback on usage/changed behavior, provided on GitHub issues
* Deprecate the flag
-->

#### Alpha

* Initial e2e tests completed and enabled
* `metadata.generation` functionality implemented
* `status.observedGeneration` functionality implemented behind feature flag
* `status.conditions[i].observedGeneration` field added to the API
* `status.conditions[i].observedGeneration` functionality implemented behind feature flag

#### Beta

* `metadata.generation`,  `status.observedGeneration`,  `status.conditions[i].observedGeneration` functionality have been implemented and running as alpha for at least one release

#### GA

* No major bugs reported for three months. 
* No negative user feedback.
* Promote the [primary e2e tests](https://github.com/kubernetes/kubernetes/blob/08ee8bde594a42bc1a222c9fd25726352a1e6049/test/e2e/node/pods.go#L422-L719) to Conformance.

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
* What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
* What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

API server should be upgraded before Kubelets. Kubelets should be downgraded 
before the API server. 

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
* Does this enhancement involve coordinating behavior in the control plane and nodes?
* How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
* How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
* Will any other components on the node change? For example, changes to CSI, 
  CRI or CNI may require updating that component before the kubelet.
-->

Previous versions of clients unaware of `metadata.generation` functionality would either 
not set the pod `metadata.generation` field (having the effective value of 0) or set it to 
some custom value, though the latter is unlikely. In either case, the API server
will ignore whatever value of `metadata.generation` is set by the client and will manage 
`metadata.generation` itself (setting it to 1 for newly created pods, or incrementing it 
for pod updates).

Already running pods will likewise either not have the pod `metadata.generation` set (and
thus have a default value of 0), or will have a custom value if `metadata.generation` was
explicitly set by a client. On the first update after the API server is upgraded, 
the API server will increment the value of `metadata.generation` by 1 from whatever it
was set to previously. This means that the first update to a pod that did not
yet have a `metadata.generation` will now have a `metadata.generation` of 1.

If a pod that has a `metadata.generation` set or incremented via the new API server is
later updated by an older API server, the older API server will not modify the
`metadata.generation` field and it will stay fixed at its current value. That means that 
if there is version skew between multiple apiservers, the `metadata.generation` may or may
not be incremented. To address this, we will not feature-gate the logic in the
apiserver that increments `metadata.generation`. That means that by the time 
ObservedGeneration goes to beta, there will be 2 versions of apiservers updating
the `metadata.generation` field, removing the issue.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml` . This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[ `prod-readiness-approvers` ](http://git.k8s.io/enhancements/OWNERS_ALIASES)
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

* [x] Feature gate (also fill in values in `kep.yaml`)
  + Feature gate name: PodObservedGenerationTracking
  + Components depending on the feature gate: kubelet, kube-controller-manager, kube-scheduler
* [x] Other
  + Describe the mechanism: 
    - Writers to `status.observedGeneration` will propagate the pod's `metadata.generation`
      to `status.observedGeneration` if the feature gate is enabled OR
      if `status.observedGeneration` is already set. We will not attempt to
      clear `status.observedGeneration` if set in order to avoid an infinite loop
      between attempting to clear the field and the API server preserving the
      existing value when an incoming update attempts to clear it.
  + Will enabling / disabling the feature require downtime of the control
    plane?
    - No. 
  + Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
    - No.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

The pod's `metadata.generation` field is currently unset by default and
both new `observedGeneration` fields will be new fields in the pod status, so the feature
will not introduce any breaking changes of default behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml` .
-->

The `status.observedGeneration` feature can be disabled by setting the flag to 'false' 
and restarting the kubelet. Disabling the feature in the kubelet means that the kubelet will not propagate 
`metadata.generation` to `status.observedGeneration` for new pods. For existing
pods, if `status.observedGeneration` is already set, the kubelet will continue
to propagate `metadata.generation` to `status.observedGeneration`. The kubelet will not attempt to clear
`status.observedGeneration` if set in order to avoid an infinite loop between
the kubelet attempting to clear the field and the API server preserving the
existing value when an incoming update attempts to clear it.

Likewise, the `conditions[i].observedGeneration` feature can be disabled by setting
the flag to 'false' in the kubelet, node lifecycle controller, and scheduler. When
the feature flag is disabled, the condition's `observedGeneration` will no longer be populated.

The `metadata.generation` functionality will intentionally not be behind a feature gate so cannot be
disabled except by downgrading the API server.

###### What happens if we reenable the feature if it was previously rolled back?

The API server will start incrementing `metadata.generation` , the kubelet will start
setting `status.observedGeneration` , and writers of pod conditions will start
incrementing those conditions' `observedGeneration` s. 

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

Unit tests will be added to cover the code that implements the feature, and will
cover the cases of the feature gate being both enabled and disabled.

The following unit test covers what happens if I disable a feature gate after having
objects written with the new field (in this case, the field should persist).

* https://github.com/kubernetes/kubernetes/blob/74210dd399c14582754e933de83a9e44b1d69c69/pkg/api/pod/util_test.go#L1228

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

A rollout or rollback won't have significant impact on any components, even
if they restart mid-rollout. Already running workloads likewise won't be
significantly impacted.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

If users see the `metadata.generation` and `status.observedGeneration` fields
are not being updated or are significantly misaligned, that indicates that
the feature is not working as expected.

Some metrics to look at that could indicate a problem include:
- `kubelet_pod_start_total_duration_seconds`
- `kubelet_pod_status_sync_duration_seconds`
- `kubelet_pod_worker_duration_seconds`

You could also check the [Pod Startup Latency SLI](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/pod_startup_latency.md).

Any of these being significantly elevated could indicate an issue with the feature.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Testing steps:

1. Create test pod with old version of API server and node; expected outcome: `generation` and `observedGeneration` fields are not populated
1. Upgrade API server
1. Send an update request to the running pod; expected outcome: `generation` is set to 1 and `observedGeneration` fields are not populated
1. Create a new pod; expected outcome: `generation` is set to 1 and `observedGeneration` fields are not populated
1. Create upgraded node
1. Create second test pod on the upgraded node; expected outcome: `generation` and `observedGeneration` fields are set to 1
1. Restart the upgraded node with the feature disabled
1. Send an update request to the second pod; expected outcome: `generation` and `observedGeneration` continue to be updated so are set to 2
1. Restart the upgraded node with the feature enabled
1. Send an update request to the second pod; expected outcome: `generation` and `observedGeneration` are set to 3

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

They can check if `metadata.generation` is set on the pod and that `observedGeneration`
is being updated.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

* [x] API .status
  + Other field: `metadata.generation`, `status.observedGeneration`, `status.conditions[].observedGeneration`

Each pod should have its `metadata.generation` set, starting at 1 and incremented by 1 for each update.

Each pod's `status.observedGeneration` should be populated to reflect the `metadata.generation` that was last
observed by the kubelet.

Each pod's `status.conditions[].observedGeneration` should be populated to reflect the `metadata.generation`
that was last observed by the component owning the corresponding condition.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  + per-day percentage of API calls finishing with 5XX errors <= 1%
  + 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%

  + 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

We can reuse the [Pod Startup Latency SLI/SLO](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/pod_startup_latency.md) here.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

We can reuse the [Pod Startup Latency SLI/SLO](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/pod_startup_latency.md) here.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost, 
implementation difficulties, etc.).
-->

N/A

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
  + [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

No.

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
  + API call type (e.g. PATCH pods)
  + estimated throughput
  + originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  + components listing and/or watching resources they didn't before
  + API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)

  + periodic API calls to reconcile state (e.g. periodic fetching state, 
    heartbeats, leader election, etc.)

-->

Yes, enabling this feature could result in additional API calls. If the pod
sync loop results in a new status where `status.observedGeneration` is the only
status field changed, there will be a new status update call. This can occur
when a pod generation is updated in the middle of a pod sync loop, but the
next sync loop does not have any other status changes.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  + API type
  + Supported number of objects per cluster
  + Supported number of objects per namespace (for namespace-scoped objects)
-->

No, this feature does not introduce any new API types.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  + Which API(s):
  + Estimated increase:
-->

No, there will not be any new calls to the cloud provider.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  + API type(s):
  + Estimated increase in size: (e.g., new annotation of size 32B)
  + Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

Enabling this feature would negligibly increase the size of pods, since
they will have new fields `metadata.generation` , `status.observedGeneration` , 
and `conditions[i].observedGeneration` populated. 

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No, this feature will not result in any noticeable performance change.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No, this feature will not increase resource usage.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No, this feature will not result in resource exhaustion. 

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

The feature depends on the API server. If the API server is unavailable, the 
new fields will not be updated. 

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  + [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?

    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?

    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.

    - Testing: Are there any tests for failure mode? If not, describe why.
-->

Other failure modes are described under Risks and Mitigations.

Detection and mitigation of the infinite status-update loop by a badly-behaving
admission webhook is covered in these docs: https://kubernetes.io/docs/concepts/cluster-administration/admission-webhooks-good-practices/#why-good-webhook-design-matters.

###### What steps should be taken if SLOs are not being met to determine the problem?

One could disable the feature gate and restart the API server. Additionally,
one could investigate the apiserver and/or kubelet logs errors.

Detection and mitigation of the infinite status-update loop by a badly-behaving
admission webhook is covered in [these docs](https://kubernetes.io/docs/concepts/cluster-administration/admission-webhooks-good-practices/#why-good-webhook-design-matters). Specifically,
the section about [detecting loops caused by competing controllers](https://kubernetes.io/docs/concepts/cluster-administration/admission-webhooks-good-practices/#prevent-loops-competing-controllers)
can be helpful.

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
* the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
* the `Proposal` section being merged, signaling agreement on a proposed design
* the date implementation started
* the first Kubernetes release where an initial version of the KEP was available
* the version of Kubernetes where the KEP graduated to general availability
* when the KEP was retired or superseded
-->

2025-01-21: initial KEP draft created
2025-02-12: PR feedback addressed, KEP moved to "implementable" and merged
2025-06-05: proposed promotion to beta
2025-09-23: proposed promotion to stable

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

We are not currently aware of any drawbacks.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

We could fully reflect the version of the spec that the Kubelet is operating on, 
such as by adding a "desired resources" field to the status or "observed spec"
field to the status where we copy the whole podspec in. 

ObservedGeneration is preferable over these alternatives because it expresses
the same amount of inforrmation while being significantly more concise, and 
because it is consistent with what other resources such as
[StatefulSet](https://github.com/kubernetes/api/blob/9e7d345b161c12a3056efa88e285b3ef68450c54/apps/v1/types.go#L274) are already doing.
