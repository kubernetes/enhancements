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
# KEP-3762: PersistentVolume last phase transition time

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
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [X] e2e Tests for all Beta API Operations (endpoints)
  - [X] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [X] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

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

We want to add a new PersistentVolumeStatus field, which would hold a timestamp of when a PersistentVolume last
transitioned to a different phase.

## Motivation

Some users have experienced data loss when using `Delete` retain policy and reverted to a safer `Retain` policy.
With `Retain` policy all volumes that are retained and left unclaimed have their phase is set to `Released`.
As the released volumes pile up over time admins want to perform manual cleanup based on the time when the volume was
last used, which is when the volume transitioned to `Released` phase.

We can approach the solution in a more generic way and record a timestamp of when the volume transitioned to any phase,
not just to `Released` phase. This allows anyone, including our perf tests, to measure time e.g. between a PV `Pending`
and `Bound`. This can be also useful for providing metrics and SLOs.

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals

1) Introduce a new status field in PersistentVolumes.
2) Update the new field with a timestamp every time a volume transitions to a different phase (`pv.Status.Phase`).

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

### Non-Goals

1) Implement any form of volume health monitoring.
2) Kubernetes will take any new actions based on the added timestamps in PersistentVolume.

<!--
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

We need to update API server to support the newly proposed field and set a value of the new timestamp field when a volume
transitions to a different phase. The timestamp field must be set to current time also for newly created volumes.

The value of the field is not intended for use by any other Kubernetes components at this point and should be used only
as a convenience feature for cluster admins. Cluster admins should be able to list and sort PersistentVolumes based on
a timestamp which indicates when the volume transitioned to a different state. 

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a cluster admin I want to use `Retain` policy for released volumes, which is safer than `Delete`, and implement a
reliable policy to delete volumes that are `Released` for more than X days.

#### Story 2

As a cluster admin I want to be able to reason about volume deletion, or produce alerts, based on a volume being in
`Pending` phase for more than X hours.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

The caveat of this proposal is that admins might not see the effect immediately after enabling/disabling the feature gate.
This is due to how and when the new `LastPhaseTransitionTime` field needs to be added/removed.

Adding the field to a PV is reasonable only when the PV actually transitions its phase - only at that point we can
capture meaningful timestamp. Trying to do this at any other step than phase transition would capture a timestamp
that would semantically incorrect and misleading.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

The new field is purely informative and should not introduce any risk.

## Design Details

Changes required for this KEP:

* kube-apiserver
  * extend [PersistentVolumeStatus](https://github.com/kubernetes/api/blob/a26a16a095cab454e928a95c533b8cf1b80aa2ec/core/v1/types.go#L402) type with `LastPhaseTransitionTime` field:
  ```
    type PersistentVolumeStatus struct {
    ...
    // lastPhaseTransitionTime represents a point in time as a timestamp of when a volume last transitioned its phase.
    // +optional
    LastPhaseTransitionTime string `json:"lastPhaseTransitionTime,omitempty" protobuf:"bytes,4,opt,name=lastPhaseTransitionTime"`
    ...
    }
    ```
  * update the timestamp whenever PV transitions to a different phase (`pv.Status.Phase`)
  * allow `LastPhaseTransitionTime` to be updated by users if needed
  * reset the timestamp in `LastPhaseTransitionTime` to `nil` only when feature gate is disabled and `LastPhaseTransitionTime` is not initialized (time is zero)

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

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

Current e2e test coverage is sufficient: [test/e2e/storage/persistent_volumes.go](https://github.com/kubernetes/kubernetes/blob/ccfac6d3200f63656575819e7b5976c12c3019a6/test/e2e/storage/persistent_volumes.go)

New e2e tests will be added for the new timestamp feature.

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

Changes will be implemented in packages with sufficient unit test coverage.

For any new or changed code we will add new unit tests.

- `pkg/apis/core/validation/`: `2023-01-25` - `82%`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

This feature could be covered with integration tests only, however e2e testing provides more value and might help
catch more bugs. Because these two kinds of tests would be almost identical, integration testing is not needed.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

We plan to add new e2e tests which should not interfere with any other tests, and so they could run in parallel.

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
- Unit tests completed and enabled
- Add unit tests covering feature enablement/disablement.
- Initial e2e tests completed and enabled

#### Beta

- Allowing time for feedback (at least 2 releases between beta and GA).
- Manually test upgrade->downgrade->upgrade path.

#### GA

- No users complaining about the new behavior.

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

No change in cluster upgrade / downgrade process.

When upgrading, the new `LastPhaseTransitionTime` field and its value will be added to PVs when transitioning phase - 
this means that enabling and disabling feature gate might not have an immediate effect. 

See "Notes/Constraints/Caveats" section for more details.

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

Version skew is not applicable, KCM was not changed in scope of this enhancement.

| API server  | Behavior              |
|-------------|-----------------------|
| off | Existing Kubernetes behavior.|
| on | New behavior.                 |

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

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: PersistentVolumeLastPhaseTransitionTime
  - Components depending on the feature gate: kube-apiserver
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

Yes. All PVs will start to contain the new `LastPhaseTransitionTime` field.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, for PVs not updated while feature was enabled. However once the `LastPhaseTransitionTime` value is set, disabling
feature gate will not remove the value.

More details in "Upgrade / Downgrade Strategy" section.

###### What happens if we reenable the feature if it was previously rolled back?

No issues expected. There are two cases that can occur for a PV:

  1. PV did not transition its phase when feature gate was enabled - the `LastPhaseTransitionTime` field was not added 
     to the PV object so this is the same case as enabling the feature gate for the first time.

  2. PV did transition its phase when feature gate was enabled - the `LastPhaseTransitionTime` field is already set,
     and it's timestamp value will be updated on next phase change.

See "Upgrade / Downgrade Strategy" and "Notes/Constraints/Caveats" sections for more details.

###### Are there any tests for feature enablement/disablement?

Unit tests for enabling and disabling feature gate are required for alpha - see "Graduation criteria" section.

The tests should focus on verifying correct handling of the new PV field in relation to feature gate state. Correct
handling means the values of the newly added field are added or updated when PV transitions its phase while feature gate
is enabled, and persisted if already set and feature gate is disabled.

Feature enablement tests:
https://github.com/kubernetes/kubernetes/blob/4eb6b3907a68514e1b2679b31d95d61f4559c181/pkg/registry/core/persistentvolume/strategy_test.go#L45

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

Rollout is unlikely to fail, unless API server fails and there should be no need for rollback as this enhancement only
adds a new field.

Rollback in terms of removal of this new field is not possible, once a PV is updated with the new field it will not be
removed by disabling this feature.

However, users can set any arbitrary timestamp value by patching PV status subresource:

```
$ kc patch --subresource=status pv/task-pv-volume -p '{"status":{"lastPhaseTransitionTime":"2023-01-01T00:00:00Z"}}'
```

Or remove it by setting zero timestamp:

```
$ kc patch --subresource=status pv/pv-1 -p '{"status":{"lastPhaseTransitionTime":"0001-01-01T00:00:00Z"}}'
$ kc get pv/pv-1 -o json | jq '.status.lastPhaseTransitionTime'
null
```

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

No metrics are required. Not having the new field set after enabling this feature is a sufficient signal to indicate
that there is a problem.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Manual upgrade->downgrade->upgrade test was performed to verify correct behavior of the new field. If a downgrade is performed
there are two scenarios that can occur for each PV:

1) Phase transitioned while feature was enabled - in this case the feature gets disabled and any updates to `LastPhaseTransitionTime` field must not be allowed.
2) Phase did not transition while feature was enabled - in this case the timestamp value must be persisted in `LastPhaseTransitionTime` field.

After upgrading again the behavior has to match behavior as if the feature was turned on for the first time.

The difference between feature enablement/disablement and downgrade/upgrade is that after downgrading to a version that
does not support `LastPhaseTransitionTime` field the data can not be accessed. Whereas only disabling the feature will
still show last the value that was set, if present.

**Upgrade->downgrade->upgrade test results:**

1) Perform pre-upgrade tests (1.27.5)

Create a PVC to provision a volume:
```
$ cat /tmp/pvc.yaml
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc-1
spec:
  accessModes:
    - ReadWriteOnce
  volumeMode: Filesystem
  resources:
    requests:
      storage: 1Gi
  storageClassName: csi-hostpath-sc
```

```
kc create -f /tmp/pvc.yaml
```

2) Verify the PV does not have `lastPhaseTransitionTime` set:
```
$ kc get pv/$(kc get pvc/pvc-1 -o json | jq '.spec.volumeName' | tr -d "\"")  -o json | jq  '.status.lastPhaseTransitionTime'
null
```

**Upgrade cluster (1.27.5 -> 1.28.1)**

1) Check available versions:
```bash
$ dnf search kubeadm --showduplicates --quiet | grep 1.28
kubeadm-1.28.0-0.x86_64 : Command-line utility for administering a Kubernetes cluster.
kubeadm-1.28.1-0.x86_64 : Command-line utility for administering a Kubernetes cluster.
```

2) Upgrade kubeadm:
```bash
$ sudo dnf install -y kubeadm-1.28.1-0
```

3) Prepare config file that enables FeatureGate:
```
$ cat /tmp/config.yaml
---
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
apiServer:
  extraArgs:
    feature-gates: PersistentVolumeLastPhaseTransitionTime=true
controllerManager:
  extraArgs:
    cluster-cidr: 10.244.0.0/16
    feature-gates: PersistentVolumeLastPhaseTransitionTime=true
```

4) Perform kubeadm upgrade:
```bash
$ sudo kubeadm upgrade plan --config /tmp/config.yaml
$ sudo kubeadm upgrade apply --config /tmp/config.yaml v1.28.1
```

5) Perform kubelet upgrade:
```bash
$ sudo dnf install -y kubelet-1.28.1-0
$ sudo systemctl daemon-reload 
$ sudo systemctl restart kubelet
```

**Perform post-upgrade tests**

1) Create a second PVC to provision a volume:
```
$ cat /tmp/pvc2.yaml
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc-2
spec:
  accessModes:
    - ReadWriteOnce
  volumeMode: Filesystem
  resources:
    requests:
      storage: 1Gi
  storageClassName: csi-hostpath-sc
```

```
kc create -f /tmp/pvc2.yaml
```

2) Verify it has `lastPhaseTransitionTime` set:
```
$ kc get pv/$(kc get pvc/pvc-2 -o json | jq '.spec.volumeName' | tr -d "\"")  -o json | jq  '.status.lastPhaseTransitionTime'
"2023-09-12T08:53:09Z"
```

3) Change retain policy on the first PV to `Retain`:
```
$ kc get pv/pvc-0c9ea251-b156-4786-ac82-8713b76bb312
NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM           STORAGECLASS      REASON   AGE
pvc-0c9ea251-b156-4786-ac82-8713b76bb312   1Gi        RWO            Retain           Bound    default/pvc-1   csi-hostpath-sc            52m
```

4) Delete PVC for the first volume to release the PV:
```
kc delete pvc/pvc-1
```

5) Verify the first (pre-upgrade) PVC transitioned phase and transition timestamp is now set:
```
$ kc get pv/pvc-f2eee26c-bca3-448b-9198-d4948f54dce3 -o json | jq '.status.phase'
"Released"

$ kc get pv/pvc-f2eee26c-bca3-448b-9198-d4948f54dce3 -o json | jq '.status.lastPhaseTransitionTime'
"2023-09-12T08:58:01Z"
```

**Downgrade cluster (1.28.1 -> 1.27.5)**

```
$ kc version -o json | jq '.serverVersion.gitVersion'
"v1.27.5"
```

**Perform post-rollback tests**

1) Create another PVC and volume:
```
$ cat /tmp/pvc3.yaml
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc-3
spec:
  accessModes:
    - ReadWriteOnce
  volumeMode: Filesystem
  resources:
    requests:
      storage: 1Gi
  storageClassName: csi-hostpath-sc
```

```
kc create -f /tmp/pvc3.yaml
```

2) Verify new PV does not have `lastPhaseTransitionTime` set:
```
$ kc get pv/$(kc get pvc/pvc-3 -o json | jq '.spec.volumeName' | tr -d "\"")  -o json | jq  '.status.lastPhaseTransitionTime'
null
```

3) Verify `lastPhaseTransitionTime` of previous PVs can not be accessed anymore:
```
$ kc get pv/$(kc get pvc/pvc-2 -o json | jq '.spec.volumeName' | tr -d "\"")  -o json | jq  '.status.lastPhaseTransitionTime'
null
```

4) Verify `lastPhaseTransitionTime` can not be set manually:
```
$ kc patch pvc/pvc-3 -p '{"status":{"lastPhaseTransitionTime":"2023-09-11T13:07:09Z"}}'
Warning: unknown field "status.lastPhaseTransitionTime"
persistentvolumeclaim/pvc-3 patched (no change)
```

**Upgrade cluster again (1.27.5 -> 1.28.1)**

1) Install/update kubeadm:
```bash
$ sudo dnf install -y kubeadm-1.28.1-0
```

2) Perform kubeadm upgrade:
```
$ sudo kubeadm upgrade plan --config /tmp/config.yaml
$ sudo kubeadm upgrade apply --config /tmp/config.yaml v1.28.1
```

**Perform post-upgrade tests again**

1) Verify timestamp is available again and unchanged on old PVs:
```
$ kc get pv/$(kc get pvc/pvc-2 -o json | jq '.spec.volumeName' | tr -d "\"")  -o json | jq  '.status.lastPhaseTransitionTime'
"2023-09-12T08:53:09Z"
```

```
$ kc get pv/pvc-f2eee26c-bca3-448b-9198-d4948f54dce3 -o json | jq '.status.lastPhaseTransitionTime'
"2023-09-12T08:58:01Z"
```

2) Change reclaim policy on exiting PV, release it and check `lastPhaseTransitionTime` is set correctly:
```
$ kc get pv/pvc-2e55f2fd-b0dc-4c95-b8d5-085d16ee6d27 -o json | jq '.spec.persistentVolumeReclaimPolicy'
"Delete"

$ kc patch pv/pvc-2e55f2fd-b0dc-4c95-b8d5-085d16ee6d27 -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'
persistentvolume/pvc-2e55f2fd-b0dc-4c95-b8d5-085d16ee6d27 patched

$ kc get pv/pvc-2e55f2fd-b0dc-4c95-b8d5-085d16ee6d27 -o json | jq '.spec.persistentVolumeReclaimPolicy'
"Retain"

$ kc get pv/pvc-2e55f2fd-b0dc-4c95-b8d5-085d16ee6d27 -o json | jq '.status.phase'
"Bound"

$ kc delete pvc/pvc-2
persistentvolumeclaim "pvc-2" deleted

$ kc get pv/pvc-2e55f2fd-b0dc-4c95-b8d5-085d16ee6d27 -o json | jq '.status.phase'
"Released"

$ kc get pv/pvc-2e55f2fd-b0dc-4c95-b8d5-085d16ee6d27 -o json | jq '.status.lastPhaseTransitionTime'
"2023-09-12T12:05:07Z"

$ date
Tue Sep 12 12:05:24 PM UTC 2023
```

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

PV objects can be inspected for `LastPhaseTransitionTime` field.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [X] API .status
  - Other field: `pv.Status.LastPhaseTransitionTime`

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

N/A - no SLI defined

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [X] Other (treat as last resort)
  - Details: To check correct functionality, inspect `LastPhaseTransitionTime` of a PV after binding it to a PVC.
  Or simply create a PVC and check dynamically provisioned PV if it has a `LastPhaseTransitionTime` set to current time.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

Due to the simple nature if this feature there's no need to add any metric.

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

No, the feature is implemented directly in API strategy for updating PVs.

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

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

Yes, all PV objects will have an entirely new status field to hold a timestamp called `LastPhaseTransitionTime`.

Estimated increase in size: < 50B

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?
Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No.

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

If API server or etcd is unavailable objects can not be updated. Since this feature relies on PVs being updated to
set `LastPhaseTransitionTime` field this feature is basically disabled in this case.

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

None, the feature is dependent only on API server and should not be affected by other failures.

###### What steps should be taken if SLOs are not being met to determine the problem?

Users should inspect API server logs for errors in case PV objects are not updated properly.

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

No drawbacks discovered, enhancement only adds a new informative field.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

Alternative solution is to update phase transition timestamp in PV controller/KCM. This would increase chances of having
a time skew between API audit logs and the timestamp. Updating phase transition timestamp in API strategy code is
therefore a better solution.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

None.
