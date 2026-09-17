# KEP-6369: Pod Assigned Resource Exposure via Downward API

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
  - [Use Cases](#use-cases)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Implementation](#implementation)
    - [Scale Down Delay in CPU Manager](#scale-down-delay-in-cpu-manager)
      - [Scale-Down Delay Timing](#scale-down-delay-timing)
      - [Consecutive Scaling](#consecutive-scaling)
    - [Resize Complete State](#resize-complete-state)
    - [Actual Resources Update](#actual-resources-update)
    - [Extend Downward API Volume to Expose CPU Manager Status](#extend-downward-api-volume-to-expose-cpu-manager-status)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha2](#alpha2)
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
  - [1. LIFO (Last-In, First-Out) CPU Release](#1-lifo-last-in-first-out-cpu-release)
  - [2. CPU Release Based on Real-Time Usage](#2-cpu-release-based-on-real-time-usage)
  - [3. Immediate Actuation (No Delay)](#3-immediate-actuation-no-delay)
  - [4. Handshake-Based Synchronization](#4-handshake-based-synchronization)
  - [5. Node Declared Features as Opt-Out Mechanism](#5-node-declared-features-as-opt-out-mechanism)
  - [6. Pod-Level Grace Period (Opt-In/Opt-Out Mechanism)](#6-pod-level-grace-period-opt-inopt-out-mechanism)
  - [7. Hook-Based Synchronization Approach](#7-hook-based-synchronization-approach)
  - [8. Generalizing Scale-Down Delay to Other Resource Types](#8-generalizing-scale-down-delay-to-other-resource-types)
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
  - [X] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.
-->

This proposal extends the downward API volume to expose the CPUs assigned and memory assigned to containers. This extension is controlled by the new `DownwardAPIAssignedResources` feature gate.

Together with KEP-6122, these two features allow latency-sensitive applications to obtain the assigned cpuset in advance via the downward API. This enables workloads to prepare for CPU removal triggered by scale-down within the guaranteed delay window `scale-delay-time`. As a result, performance degradation caused by sudden CPU loss is avoided.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Latency-sensitive applications often require exclusive CPUs to achieve predictable performance and resource isolation. These applications commonly use CPU affinity to minimize performance degradation caused by CPU migration.

When scaling down, guaranteed QoS pods need to know in advance which CPUs will be removed from their cpuset. This allows latency-sensitive applications to take preparatory actions — such as migrating workloads away from affected CPUs — and avoid performance degradation caused by CPU migration and core sharing during the removal of active CPUs.

**Note:** This KEP was split from [KEP-6122](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/6122-configurable-scaling-delay-with-pod-resource-exposure) because the two features (configurable scale-down delay and downward API exposure of assigned resources) are functionally independent. This separation improves document readability and reduces complexity, making each feature easier to understand and review.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

* Expose CPUSet assignments to containers via the downward API volume with `DownwardAPIAssignedResources` feature gate enabled:
   + `assigned.cpuset`: the desired exclusive cpuset (Linux cpuset format, e.g. `0-3,7,12-15`). Empty string when no exclusive CPUs are assigned.
* Expose Memory Manager assignments to containers via the downward API volume with `DownwardAPIAssignedResources` enabled, the exposed value is:
   + `assigned.memset`: Memory Blocks, memory size and NUMA affinity information  (e.g. `memory:2097152000,NUMA:[0]`). Empty string when no exclusive memory assigned.
### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

* Allow containers to specify which CPUs to remove during scale-down.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

KEP-6122 proposed a delay for guaranteed pod scale down. This allows latency-sensitive workloads to monitor and prepare for the upcoming CPUSet change before CPU(s) are removed from the container.

This proposal extends the Downward API volume to expose CPU Manager cpuset information through a new `assigned.cpuset` resource field and Memory Manager memset information through a new `assigned.memset` resource field. During the delay window, `assigned.cpuset` exposes the desired cpuset and `assigned.memset` exposes the desired memory assigned info so that workloads can prepare for the upcoming change. Exposition of new field is gated by the `DownwardAPIAssignedResources` feature gate.

### Use Cases

Refer to https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/6122-configurable-scaling-delay-with-pod-resource-exposure/README.md#use-cases.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

**Security Considerations:** Exposing `assigned.cpuset` and `assigned.memset` via Downward API does not introduce new security risks. The same CPU assignment information is already accessible to applications from inside the container via `/proc/self/cgroup` controllers and from the node level. The `assigned.cpuset` field only exposes CPU IDs (e.g., `0-3,7,12-15`), not NUMA topology information. This KEP simply exposes the assigned CPUset and assigned memory assigned information in advance (during the scale-down delay window) without creating new data recipients or adding new security risks.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Implementation

The `DownwardAPIAssignedResources` feature integrates with the existing Downward API framework to expose assigned CPU and Memory resources to containers. The feature relies on the `DownwardAPIAssignedResources` feature gate to determine whether the API server accepts pods using `assigned.cpuset` and `assigned.memset` fields.

A new field `assigned.cpuset` is added to the existing `ResourceFieldRef.Resource`:

ResourceFieldRef.Resource
* resource: limits.cpu
   + A container's CPU limit
* resource: requests.cpu
   + A container's CPU request
* resource: limits.memory
   + A container's memory limit
* resource: requests.memory
   + A container's memory request
* resource: limits.hugepages-*
   + A container's hugepages limit
* resource: requests.hugepages-*
   + A container's hugepages request
* resource: limits.ephemeral-storage
   + A container's ephemeral-storage limit
* resource: requests.ephemeral-storage
   + A container's ephemeral-storage request
* **resource: assigned.cpuset** *(NEW)*
   + **A container's CPU desired assignments**
* **resource: assigned.memset** *(NEW)*
   + **A container's Memory desired assignments**

The Volume manager gets the CPU state from CPU manager, and writes it to the Downward API volume file `assigned.cpuset`, which exposes the CPUSet to the container:
  - If the container has exclusive CPUs assigned, the value exposes the exclusive cpuset (If preAssignments exist (scale-down is pending), this exposes the preAssignments; otherwise, it exposes the assignments).
  - Otherwise, the value is empty ("").

The Volume manager gets the Memory state from Memory manager, and writes it to the Downward API volume file `assigned.memset`, which exposes the MemorySet to the container:
  - If the container has memory assigned, the value exposes the memory NUMA nodes (e.g., `0-1`).
  - Otherwise, the value is empty ("").

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

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

We plan on adding/modifying functions to the following files:
- `pkg/volume/downwardapi/downwardapi_test.go`
- `pkg/apis/core/validation/validation_test.go`

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

Unit and E2E tests provide sufficient coverage for Alpha. For Beta, the testing plan re-evaluates whether integration tests are required.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

These cases will be added in the existing e2e_node tests to verify that the Downward API correctly exposes CPU manager states via `assigned.cpuset` field and Memory Manager states via `assigned.memset` field.

Prerequisites:

1. Enable the following feature gates:
    * `InPlacePodVerticalScalingExclusiveCPUs`
    * `DownwardAPIAssignedResources`
2. Configure the CPU Manager policy to `static`.
3. Configure the Memory Manager policy to `Static`.

The following scenarios will be tested:

| No | Test | Description | Expected Result |
|----|------|-------------|-----------------|
| 1 | Validate Downward API Exposure | Create a pod with exclusive CPUs and `assigned.cpuset` downward API volume. | • Pod reaches Running state<br />• `/etc/podinfo/assigned.cpuset` contains correct cpuset (e.g., `0-3`)<br />• `/etc/podinfo/assigned.memset` contains correct memory NUMA nodes (e.g., `0-1`) |
| 2 | Validate Downward API for Non-Exclusive CPU Pods | Create a pod without exclusive CPUs but with `assigned.cpuset` downward API volume. | • Pod reaches Running state<br />• `/etc/podinfo/assigned.cpuset` is empty ("")<br />• `/etc/podinfo/assigned.memset` is empty ("") |
| 3 | Validate Downward API Update on Resize | Create a pod with 4 exclusive CPUs, scale down to 2 CPUs. | • Initial `assigned.cpuset` contains 4 CPUs<br />• After scale-down, `assigned.cpuset` updates to 2 CPUs<br />• `assigned.memset` reflects the memory NUMA nodes after resize |


### Graduation Criteria

#### Alpha

* Feature implemented behind the `DownwardAPIAssignedResources`.
* Validation logic is in-place in kube-apiserver
* Exposing CPU manager (e.g., `assigned.cpuset`) and Memory Manager information (e.g., `assigned.memset`) via the Downward API.
* unit testing and e2e testing for downward API enhancement for CPU and memory exposure.

#### Beta

* No unresolved critical bugs.
* Bugs reported by users have been addressed

#### GA

* Allow time for feedback (6+ months).
* Make sure all risks have been addressed.

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

The new field `assigned.cpuset` and `assigned.memset` is exposed behind the `DownwardAPIAssignedResources` feature gate. Field validation in kube-api-server depends on this gate. Kubelet handles file creation and updates based on pod spec and dependency on feature gate.

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

Exposition of the field `assigned.cpuset` and `assigned.memset` is behind feature gate `DownwardAPIAssignedResources`. The feature gate is Alpha and disabled by default. The documentation states: "Only enable this feature gate when all kubelets in the cluster support this feature." The operator must upgrade all kubelets first, then enable the feature gate.

Considering the following scenarios:

* Cluster has kubelets both: without the feature (1.36-) and with feature implemented (1.37+) (mixed versions): The operator does not enable the feature gate. Nobody can use `assigned.cpuset` and `assigned.memset`.

* All kubelets upgraded to versions having feature implemented (1.37+): The operator enables the feature gate on the API server and kubelets.
`assigned.cpuset` and `assigned.memset` can be mounted as DownwardAPI volume files by pods.

* Operator enables the feature gate while some kubelets still don’t have feature implemented (1.37-): This is an operator error. The feature gate is Alpha and disabled by default. The documentation states: "Only enable this feature gate when all kubelets in the cluster support this feature." The operator must upgrade all kubelets first, then enable the feature gate.

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

This feature requires enabling the following feature gates

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DownwardAPIAssignedResources`
  - Requires `--cpu-manager-policy` kubelet configuration set to `static`

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Enabling `DownwardAPIAssignedResources`, it will expose the CPU states via downward API. Feature gate `DownwardAPIAssignedResources` only gates the kube-apiserver validation.

The feature gate behavior is as follows:
- **Feature gate disabled (apiserver implements the feature but gate is off):** The pods with `assigned.cpuset` or `assigned.memset` will be rejected, but the field has no effect. 
- **Feature gate enabled:** The `assigned.cpuset` or `assigned.memset` field is validated and accepted in pod specs.
- **Feature not implemented (older apiserver version):** The apiserver does not recognize the `assigned.cpuset` or `assigned.memset` field and will reject pods using it with a validation error: "unsupported container resource: assigned.cpuset".

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes.

Disabling `DownwardAPIAssignedResources` on kube-apiserver causes the `assigned.cpuset`  or `assigned.memset` field will be rejected. Existing pods with `assigned.cpuset`  or `assigned.memset` continue to run without interruption. However, the field will have no effect until the feature gate is re-enabled.

###### What happens if we reenable the feature if it was previously rolled back?

The `assigned.cpuset` and `assigned.memset`  will contain proper values.

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

The feature gate `DownwardAPIAssignedResources` is Alpha and is disabled by default.

However, since this feature only affects the timing of cpuset changes and not the allocation itself, a failure does not impact already running workloads — their current cpuset remains in effect.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Local Testing Plan: (Need check)

**Test `DownwardAPIAssignedResources` feature upgrade and rollback**

Note: The `DownwardAPIAssignedResources` feature gate is configured on kube-apiserver (not kubelet). The kubelet always supports reading `assigned.cpuset` from pod specs and exposing CPU manager state via Downward API volumes. The feature gate only controls API server validation of the `assigned.cpuset` field.

1. Deploy cluster with `DownwardAPIAssignedResources` feature gate disabled on kube-apiserver.
2. Create a pod with `assigned.cpuset` downward API volume.
   - Verify the pod is admitted (field is silently ignored by apiserver).
   - Verify kubelet creates the downward API volume file (empty or with current cpuset).
3. Initiate a pod downscaling request.
   - Verify the pod scales down after timer expiry.
   - Verify the downward API volume file is updated with the new cpuset by kubelet.
4. Enable the feature gate on kube-apiserver (restart apiserver) and initiate another pod downscaling request.
   - Verify the pod remains in `Running` state without errors.
   - Verify CPU manager states are exposed through the Downward API.
   - Verify new pods with `assigned.cpuset` are accepted by apiserver.
5. Disable the feature gate on kube-apiserver again and initiate another pod downscaling request.
   - Verify existing pods with `assigned.cpuset` continue running without errors (field is silently ignored).
   - Verify the pod still scales down after timer expiry.
   - Verify kubelet continues to update the downward API volume file (kubelet behavior is independent of the feature gate).
6. Finally, re-enable the feature gate on kube-apiserver and initiate another pod downscaling request.
   - Verify the pod remains in `Running` state without errors.
   - Verify CPU manager states are exposed through the Downward API.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

N/A

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

Check the downward API volume in the pod. Files /etc/podinfo/cpuset should be present respectively for CPU info.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

Check whether there is a downward API volume of cpuset in the pod.

After containers scaling, the cpuset exposed via downward API will changed accordingly.

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

- The downward API volume must reflect the new cpuset before the cpuset is applied, ensuring the workload has the full delay window to prepare.
- After at least `scale-delay-time` has elapsed, the new cpuset is applied to the container at the next cpuset actuation time.
- Scale-up operations are not delayed by this feature and follow the existing behavior.
- The cpuset values exposed via the downward API must always be accurate: `assigned.cpuset` and `assigned.memset` must reflect the allocated cpuset (preAssignments if scale-down is pending, otherwise assignments).


###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- Time from the CPU manager determining the new cpuset to the downward API volume being updated. This indicates whether the workload receives timely notification of the upcoming change.

- Number of scale-down operations where the downward API volume was updated after the cpuset was applied (should be zero). This indicates whether the critical guarantee — that the workload is notified before the change — is being upheld.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

No

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

No

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

A new field `assigned.cpuset` is added to the existing `ResourceFieldRef.Resource`:

* resource: limits.cpu
   + A container's CPU limit
* resource: requests.cpu
   + A container's CPU request
* resource: limits.memory
   + A container's memory limit
* resource: requests.memory
   + A container's memory request
* resource: limits.hugepages-*
   + A container's hugepages limit
* resource: requests.hugepages-*
   + A container's hugepages request
* resource: limits.ephemeral-storage
   + A container's ephemeral-storage limit
* resource: requests.ephemeral-storage
   + A container's ephemeral-storage request
* **resource: assigned.cpuset** *(NEW)*
   + **A container’s CPU desired assignments**
* **resource: assigned.memset** *(NEW)*
   + **A container's Memory desired assignments**

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

No

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

N/A

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

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

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

N/A

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->
