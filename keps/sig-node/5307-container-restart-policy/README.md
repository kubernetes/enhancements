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
- [x] **Merge early and iterate.**
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

-->
# KEP-5307: Container Restart Rules

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
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Container Restart Count](#container-restart-count)
    - [Job PodFailurePolicy](#job-podfailurepolicy)
    - [​maxRestartTimes](#maxrestarttimes)
    - [Sidecar Containers](#sidecar-containers)
    - [Future Improvements](#future-improvements)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Unintended Restart Loops](#unintended-restart-loops)
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
  - [Wrapping entrypoint](#wrapping-entrypoint)
  - [Non-declarative (callbacks based) restart policy](#non-declarative-callbacks-based-restart-policy)
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
  - [x] e2e Tests for all Beta API Operations (endpoints)
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

This KEP introduces container restart rules for a container so kubelet can apply
those rules on container exits. This will allow users to configure special
exit codes of the container to be treated as non-failure and restart the
container in-place even if the Pod has a restartPolicy=Never. This scenario is
important for use cases, when rescheduling of a task is very expensive,
and restarting in-place is preferable.

This KEP is the first part of a larger plan to improve the container restart
behavior, for more discussion and details, see [this document](https://docs.google.com/document/d/1UmJHJzdmMA1hWwkoP1f3rG9nS0oZ2cRcOx8rO8MsExA/edit?resourcekey=0-OuKspBji_1KJlj2JbnZkgQ&tab=t.0)..

## Motivation

With the proliferation of AI/ML training jobs where each job takes hundreds
of Pods, each using expensive hardware and very expensive to schedule,
in-place restarts are becoming more and more important.

Consider the example, the Pod is a part of a large training job. The progress
of each training “step” is only made when all Pods completed the calculation
for this step. Each Pod starts from a checkpoint, they all make progress
together, and write a new checkpoint. If any of Pods failed, the fastest
way to restart the calculation is to interrupt all Pods by restarting them,
so they all will start from the previous checkpoint. Thus, a special handling
of this restart is required.

There are a few reasons why the OnFailure restart policy will not work:

1) The cases of failed hardware must result in a Pod failure and rescheduling.
There needs to be a differentiation of these two failures - caused by hardware
issue and caused by a in-place restart request.

2) Pods are often parts of JobSets with the Job failure policy configured
(see https://kubernetes.io/docs/tasks/job/pod-failure-policy/). The pod
failure policy is a server-side policy and is not compatible with the Pods
with restartPolicy OnFailure.

### Goals

- Introduce the Container RestartPolicyRuless API which allows to keep
restarting a container on specified exit codes.
- Allow extensibility of an API to support more scenarios in future.

### Non-Goals

- Implement the ​​maxRestartTimes https://github.com/kubernetes/enhancements/issues/3322
- Support all possible restart policy rules in this KEP - some may be ideas for the future,
for a more detailed discussion on possible actions and conditions, please refer to
[this document](https://docs.google.com/document/d/1UmJHJzdmMA1hWwkoP1f3rG9nS0oZ2cRcOx8rO8MsExA/edit?resourcekey=0-OuKspBji_1KJlj2JbnZkgQ&tab=t.0).

## Proposal

### User Stories (Optional)

#### Story 1

As a ML researcher, I'm orchestrating a large number of long-running AI/ML training
worklaods. Workload failures in such workloads are unavoidable due to various reasons. 
When a workload fails (with a retriable exit code), I would like the container to be 
restarted quickly and avoid re-scheduling the pod because this consumes significant
amount of time and resource. Restarting the failed container "in-place" is critical
for a better utilization of compute resources. The container should only restart
"in-place" if it failed due to a retriable error; the container and pod should
terminate and possibly reschedule if the failure is not retriable. 

Since AI/ML training workloads are often declared as Job, with [PodFailurePolicy](https://kubernetes.io/docs/tasks/job/pod-failure-policy/),
some errors should be treated as retriable and restart the container in-place
by the kubelet, without re-creating and re-scheduling the Pod.

See https://github.com/kubernetes-sigs/jobset/issues/876 for the detailed description.

### Notes/Constraints/Caveats (Optional)

#### Container Restart Count

Pod with `restartPolicy=Never` may have containers restarted and have the
restart count higher than 0 because container-level restart rules can restart
the container. 

This is already possible for sidecar containers which have container-level
`restartPolicy=Always`.

#### Job PodFailurePolicy

This will not affect how Job `podFailurePolicy` interacts with pod failures,
because container-level restart will not be considered as Pod termination.
The Job controller only checks and evaluates `podFailurePolicy` after a Pod
is terminated, as mentioned in the [KEP-3329](https://github.com/kubernetes/enhancements/blob/fc234c10fe886bb3dcdf3499ae822e0650fb1179/keps/sig-apps/3329-retriable-and-non-retriable-failures/README.md?plain=1#L902-L903).

This KEP is informed from the discussion about some future improvements 
we may need to implement as described here for JobSet:
https://github.com/kubernetes/enhancements/issues/3329#issuecomment-1571643421.
Instead of making Job controller to handle container-level restart and failures,
kubelet is more suitable to handle the container restart policy. This aligns
with the current implementation that Job controller only restart / reschedule
the Pod after it is terminated, and delegate the rest to the kubelet.

This enables efficient setups to accelerate container restart and to improve
resource utilization. For example, Jobs configured with `podFailurePolicy` for
hardware failures  (Pod needs to be rescheduled to other nodes), and containers
configured with `restartPolicyRules` to restart in-place for training errors.

#### ​maxRestartTimes 
`maxRestartTimes` is another ongoing [KEP-3322](https://github.com/kubernetes/enhancements/issues/3322)
that provides an Pod API to allow the user to specify a maximum number of
restarts. How should Container `restartPolicyRules` interacts with the Pod 
`maxRestartTimes` is being discussed. The current understanding is that the
containers restarted by `restartPolicyRules` will count towards container restarts
of all other APIs.

#### Sidecar Containers
This proposal does not change how Sidecar containers will be detected and their
lifecycles. For future improvements on Sidecar containers, please see below.

#### Future Improvements
This proposal fits into the larger improvement to support other container restart
conditions and actions. Please refer to [this document](https://docs.google.com/document/d/1UmJHJzdmMA1hWwkoP1f3rG9nS0oZ2cRcOx8rO8MsExA/edit?resourcekey=0-OuKspBji_1KJlj2JbnZkgQ&tab=t.0).

### Risks and Mitigations

#### Unintended Restart Loops

A container might persistently exit with an "Restart" exit code due to
an unresolvable underlying issue, leading to frequent restarts that consume
node resources and potentially mask the problem.

The container restart will still follow the exponential backoff to avoid
excessive resource consumption due to restarts.

Although this introduces exponential delay for container restart, it still
aligns with the goal of expediting in-place container restart. First, the
in-place restart avoids the expensive Pod re-scheduling to a different node.
Second, if the container keeps restarting due to an exit code specified in
the restart rules and stuck in a CrashLoop, it is probably not a retry-able
error, the exponential backoff can avoid overwhelming the node with frequent
restarts.

## Design Details

The proposal is to extend the Pod's [Container API type](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/types.go#L2528) 
with a new field `restartPolicyRules`.

The Pod’s specified restartPolicy (Always / Never / OnFailure) will act as 
the default behavior for each container. The user has the ability to specify
a  `restartPolicy` on the container, which will override the `restartPolicy`
from the Pod. If the container `restartPolicy` is not specified, the pod
`restartPolicy` will be used. Same as now, for Sidecar containers, the user
needs to specify container `restartPolicy=Always` on an init container.

Additionally, each container could have its own `restartPolicyRules`. 
If the `restartPolicyRules` field is specified, then the user must 
also specify the container `restartPolicy` which is defined next to it. 
The `restartPolicyRules` define a list of rules to apply on container exit.
Each rule will consist of a condition (onExitCodes, OOM killed, eviction,
resource contention etc.) and an action (Restart, Terminate, TerminatePod, etc.)
The rules will be evaluated in order; if none of the rules’ conditions matched,
the default action will fallback to container’s `restartPolicy`.

The initial proposal supports only one action, "Restart", to restart the container.

The initial proposal supports only exit code as requirement for the rules.

The proposed API is as following:

```go
type ContainerRestartPolicy string

const (
  ContainerRestartPolicyAlways ContainerRestartPolicy = "Always"
  ContainerRestartPolicyNever ContainerRestartPolicy = "Never"
  ContainerRestartPolicyOnFailure ContainerRestartPolicy = "OnFailure"
)

type Container struct {
  // Omitting irrelevant fields...
  // RestartPolicy must be specified if RestartPolicyRules is specified.
  RestartPolicy *ContainerRestartPolicy

  // Represents a list of rules to be checked to determine if the
  // container should be restarted on exit. The rules are evaluated in
  // order. Once a rule matches a container exit condition, the remaining
  // rules are ignored. If no rule matches the container exit condition,
  // the Pod-level restart policy determines the whether the container
  // is restarted or not. Constraints on the rules:
  // - At most 20 rules are allowed.
  // - Rules can have the same action.
  // - Identical rules are not forbidden in validations.
  RestartPolicyRules []ContainerRestartRule
}

// ContainerRestartRule describes how a container exit is handled.
type ContainerRestartRule struct {
  // Specifies the action taken on a container exit if the requirements
  // are satisfied. The only possible value is "Restart" to restart the
  // container.
  // +required
  Action ContainerRestartRuleAction

  // Represents the exit codes to check on container exits. The oneOf
  // field must be provided.
  // +optional
  // +oneOf=when
  ExitCodes *ContainerRestartRuleOnExitCodes

  // Other conditions in the future:
  // OOMKill *ContainerRestartRuleConditionOOMKill
  // RestartTimes *ContainerRestartRuleConditionRestartTimes
  // Exit *ContainerRestartRuleConditionExit
}

type ContainerRestartRuleAction string

const (
  ContainerRestartRuleActionRestart ContainerRestartRuleAction = "Restart"

  // Future actions: "Complete", "TerminatePod", "RestartPod".
)

// ContainerRestartRuleOnExitCodes describes the condition
// for handling an exited container based on its exit codes.
type ContainerRestartRuleOnExitCodes struct {
  // Represents the relationship between the container exit code(s) and the
	// specified values. Possible values are:
	//
	// - In: the requirement is satisfied if the container exit code is in the 
  //   set of specified values.
	// - NotIn: the requirement is satisfied if the container exit code is 
  //   not in the set of specified values.
  // +required
  Operator ContainerRestartRuleOnExitCodesOperator

  // Specifies the set of values to check for container exit codes.
  // At most 255 elements are allowed.
  Values []int32
}

type ContainerRestartRuleOnExitCodesOperator string

const (
  ContainerRestartRuleOnExitCodesOpIn ContainerRestartRuleOnExitCodesOperator = "In"
  ContainerRestartRuleOnExitCodesOpNotIn ContainerRestartRuleOnExitCodesOperator = "NotIn"
)
```

To specify a container to only restart with an exit code of 42, it
can be specified as following in a Pod manifest:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
spec:
  containers:
  - name: my-container
    image: nginx:latest
    # restartPolicy must be specified to specify restartPolicyRules
    restartPolicy: Never
    restartPolicyRules:
    - action: Restart
      when:
        exitCodes:
          operator: In
          values: [42]
```

Below is the example of the shape of the API for future improvements.
NOT all actions and conditions are included for this KEP.

To deploy a pod with 
- an init container that should be retried for 10 times,
- a sidecar container,
- a regular container that should only be restarted on exit code 42, and
- a regular (keystone) container, the exit of which should fail the pod:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
spec:
  restartPolicy: Never
  initContainers:
  - name: retry-init
    image: xxx
    restartPolicy: Never # This needs to be specified because restart rules is specified.
    restartPolicyRules:
    - action: Complete
      when:
        restartTimes: 10
    - action: Restart
      when:
        exitCodes:
          operator: NotIn
          values: [0]
  - name: sidecar
    image: xxx
    restartPolicy: Always # Indicates a sidecar container
  containers:
  - name: regular-container
    image: xxx
    restartPolicy: Never
    restartPolicyRules:
    - action: Restart
      when:
        exitCodes:
          operator: In
          values: [42]
  - name: keystone-container
    image: xxx
    restartPolicyRules:
    - action: TerminatePod
      when:
        exitCodes:
          operator: NotIn
          values: []
```

The proposal is to support the following combinations:

- The action can only be `Restart`;
- Only `onExitCodes` rules are allowed, no other conditions;
- The `operator` can be either `In` or `NotIn`;
- Values only support an array of integers and no wildcard.

With the limitations above, an API will do nothing for containers with
pod-level and container-level `restartPolicy=Always`, as the only action is 
`Restart`. Same for the containers with pod-level and container-level
`restartPolicy=OnFailure`. Except that exit code 0 can be configured 
to be restartable, which is effectively the same as `restartPolicy=Always`.

For the containers with the `restartPolicy=Never`, it will allow restarting
the container for the subset of exit codes. The sync and restart logic
will be implemented in k8s.io/kubelet/container.

Similarly for sidecar init containers with `restartPolicy=Always`, setting
`restartPolicyRules` has no effect because the only action is `Restart`.

This API change is only intended to restart the container if the container
itself exited with the given list of exit codes. It is not intended to change
the behavior of other means that lead to container being restarted, for
example, pod resize or pod restart.

See more discussion on how this API interacts with other components like Job
controller in [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional).

This API will support regular "app" containers as discussed above. 

For init containers, this API will repeatedly restart the init container if
it failed with the exit code specified in the restartPolicyRules until it succeeds
(exit=0). For Pods with `restartPolicy=Never`, the `restartPolicyRules` override it.
This means if the container exited with a code specified in the `restartPolicyRules`,
the container will be restarted by kubelet, until it succeeds (exit=0) or fails
(exited with a code not in the `restartPolicyRules`).

For sidecar containers, this API effectively has no affect, because the sidecar
container is always restarted.

For ephemeral containers, this API is not allowed, because restarting the
ephemeral containers is not meaningful.

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
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `k8s.io/apis/core`
- `k8s.io/apis/core/v1/validations`
- `k8s.io/features`
- `k8s.io/kubelet`
- `k8s.io/kubelet/container`

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

Unit and E2E tests provide sufficient coverage for the feature. Integration
tests may be added to cover any gaps that are discovered in the future.

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

- Verify that containers can specify restartPolicyRules.
- Verify that containers exited with exit codes specified in the restartPolicyRules
are restarted and the pod keeps running.
- Verify that containers exited with exit codes not specified in the restartPolicyRules
are not restarted and the pod fails.
- Verify that PodFailurePolicy works with the restartPolicyRules; containers restarted
by the restartPolicyRules should not fail the Pod and trigger PodFailurePolicy.

E2E tests:
- https://github.com/kubernetes/kubernetes/blob/9a3dce00ae32c81346883fb5a689a8240d48c218/test/e2e/node/pods.go#L722
- https://github.com/kubernetes/kubernetes/blob/9a3dce00ae32c81346883fb5a689a8240d48c218/test/e2e/apps/job.go#L1331
- https://testgrid.k8s.io/sig-release-master-informing#kind-master-alpha-beta&include-filter-by-regex=ContainerRestartRules

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
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved 

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

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

- Container restart policy added to the API.
- Container restart policy implemented behind a feature flag.
- Initial e2e tests completed and enabled.
- Public documentation on pod restart policy is updated to distinguish between
pod restart policy, container restart policy, and container restart rules.

#### Beta

- Container restart policy functionality running behind feature flag
for at least one release.
- Container restart policy runs well with Job controller.
- All monitoring requirements completed.
- All testing requirements completed.
- All known pre-release issues and gaps resolved.

#### GA

- No major bugs reported for three months.
- User feedback (ideally from at least two distinct users) is green.

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

API server should be upgraded before Kubelets. Kubelets should be downgraded
before the API server.

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

Previous kubelet client unaware of the container restart policy will ignore
this field and keep the existing behavior determined by pod's restart policy.

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
  - Feature gate name: ContainerRestartRules
  - Components depending on the feature gate: kubelet, kube-apiserver

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No. The feature introduces a new API field, restartPolicyRules, to the container
spec. If this field is not specified, the existing behavior determined by
the Pod's restartPolicy remains unchanged.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes. To roll back, the feature gate should be disabled in the API server and
kubelets, and components should be restarted. If a Pod was created with the
restartPolicyRules field while the gate was enabled, those rules will be ignored by
kubelets once the feature is disabled.

###### What happens if we reenable the feature if it was previously rolled back?

If the feature is re-enabled, the kubelet will once again recognize and enforce
the restartPolicyRules for any Pods that have the field defined. The container
restart logic described in the KEP will become active again.

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

- Unit test for the API's validation with the feature enabled and disabled.
- Unit test for the kubelet with the feature enabled and disabled.
- Unit test for API on the new field for the Pod API. First enable
the feature gate, create a Pod with a container including restartRules,
validation should pass and the Pod API should match the expected result.
Second, disable the feature gate, validate the Pod API should still pass
and it should match the expected result. Lastly, re-enable the feature
gate, validate the Pod API should pass and it should match the expected result.

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

If this feature is being actively used in a cluster that has this feature
partially enabled on some nodes, the Pod may behave differently on exit.
Pods on nodes with this feature may restart in-place, while pods on nodes
without this feature may not be restarted.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

Repeated restart of container or pods.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Manual testing was performed to verify the upgrade and rollback paths.
- **Upgrade:** A cluster with the feature disabled was upgraded to a version with the feature enabled. Pods with `restartPolicyRules` were deployed and observed to behave as expected.
- **Rollback:** A cluster with the feature enabled and `restartPolicyRules` Pods running was rolled back to a version with the feature disabled. Existing Pods continued to run, but `restartPolicyRules` were ignored. New Pods created with `restartPolicyRules` also ignored the rules.
- **Upgrade->Downgrade->Upgrade:** This path was tested by performing the above steps sequentially. The feature behaved as expected at each stage, with `restartPolicyRules` being respected when the feature was enabled and ignored when disabled.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

Operators can determine if the feature is in use by checking the Pod spec for the presence of the `restartPolicyRules` field within container definitions. Additionally, monitoring the `kube_pod_container_status_restarts_total` metric can indicate container restarts that might be governed by these rules.

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
- [x] API .status
  - Other field: ContainerStatuses
    - Container statuses will have the history of the container restarts.
- [x] Other (treat as last resort)
  - Details: The metric `kube_pod_container_status_restarts_total` will show the total count of container restarts.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- The rate of unexpected container restarts (i.e., not matching a `restartPolicyRules`) should remain below 1%.
- The time taken for a container to restart after an exit code matching `restartPolicyRules` should be within typical container restart latencies, accounting for exponential backoff.
- Kubelet SLOs should not be impacted.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `kube_pod_container_status_restarts_total`
  - Aggregation method: Sum over time, grouped by container and pod.
  - Components exposing the metric: kube-state-metrics
- [x] Other (treat as last resort)
  - Details: PodStatus API will also have a full history of containers restarted in ContainerStatuses field. Containers restarted by RestartPolicyRules will be included in the statuses history.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

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

No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

Enabling this feature will introduce a new field `restartPolicyRules` on the
[Container API type](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/types.go#L2528).

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

[Container API type](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/types.go#L2528)
will be increased. The rules can handle at most 256 int32 exit values, plus
the action name ("In" or "NotIn"), the size will increase by at most 1029B.

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

The container will keep running or restarted by kubelet. Deletion of the pod / container may be delayed.

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

If kubelet becomes unavailable or is being restarted, there might be delays in container restarts.

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

- 1.34: Implemented in Alpha
  - https://github.com/kubernetes/kubernetes/pull/132642
  - https://github.com/kubernetes/kubernetes/pull/133243

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

### Wrapping entrypoint

One way to implement this KEP as a DIY solution is to wrap the entrypoint
of the container with the program that will implement this exit code handling
policy. This solution does not scale well as it needs to be working on multiple
Operating Systems across many images. So it is hard to implement universally.

### Non-declarative (callbacks based) restart policy

An alternative to the declarative failure policy is an approach that allows
containers to dynamically decide their faith. For example, a callback is called
on an “orchestration container” in a Pod when any other container has failed.
And the “orchestration container” may decide the fate of this container - restart
or keep as failed.

This may be a possibility long term, but even then, both approaches can work
in conjunction.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
