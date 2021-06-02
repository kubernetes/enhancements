# KEP-2775: kubectl delete interactivity and delay

<!--
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
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
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
<!-- /toc -->

## Release Signoff Checklist

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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

When a namespace is deleted it also deletes all of the resources under it. The deletion runs without further confirmation, and can be devastating if accidentally run against the wrong namespace (e.g. thanks to hasty tab completion use).  

```  
kubectl delete namespace prod-backup  
```

When all namespaces are deleted essentially all resources are deleted. This deletion is trivial to do with the `--all` flag, and it also runs without further confirmation. It can effectively wipe out a whole cluster.  

```
kubectl delete namespace --all
```  

There are certainly things cluster operators should be doing to help prevent this user error (like locking down permissions) but it doesn't matter how many mistakes have to pile up to create a perfect storm of a bad thing because we're allowing a bad thing to happen without a confirmation gate.

This KEP proposes two protections when deleting resources with kubectl.

## Motivation

Over the years we've heard stories from users who've accidentally deleted resources in their clusters - often all of their resources. This trend seems to be rising lately as newer folks venture into the Kubernetes/DevOps/Infra world. Experienced users are also not accident proof and have shared tragic tales.

### Goals

* Mitigate the potential for accidental imperative deletes

### Non-Goals

* Server side solutions

## Proposal

1. `kubectl delete [--all | --all-namespaces]` will warn about the destructive action that will be performed and artificially delay for x seconds allowing users a chance to abort.

2. Add a new `--interactive | -i` flag to `kubectl delete` that will require confirmation before deleting resources imperatively (i.e. not `-f`). This flag will be false by default. If this flag is explicitly set to false it will not prompt or artificially delay.

These two changes introduce significant protections for users and do not break backwards compatibility. Scripts will see an x second delay unless they explicitly set `-i=false`.

This KEP is specifically targeting imperative deletes to speed up rollout. Future KEPs may expand the scope of interactive confirmation.

```
kubectl delete deployment nginx --interactive
```

### User Stories (Optional)

#### Story 1

> Creating basic safeguards is not just about junior users, I have deleted massive amounts of infrastructure several times because I was in the wrong kubectl context. 

https://groups.google.com/g/kubernetes-dev/c/y4Q20V3dyOk/m/8xdQ8TM_BgAJ

> We had this run by a developer by mistake today and it wiped all resources in the cluster, with the exception of the  `kube-system`,  `kube-public`  and  `default`  namespaces, and the resources within.

https://github.com/kubernetes/kubectl/issues/696

> One of the members of our technical staff with admin privileges tried to delete all objects in a namespace, in an event of hurry, he ran, something like `kubectl delete namespace -n some-namespace --all` which caused all namespaces and their objects deleted to a very granular level, apart from objects in kube-* namespaces.

As a user I should be warned before accidentally deleting all of the resources in my cluster.

### Notes/Constraints/Caveats (Optional)

Several contributors have raised concerns about making breaking changes here. Our goal is to strike a balance between not breaking existing scripts and protecting users from making mistakes.

The major restriction is around users running kubectl in CI/CD pipelines. We do not want to break their scripts.

### Risks and Mitigations

CI/CD jobs that upgrade their version of kubectl will see an x second slow down if they are deleting with `--all | --all-namespaces` without setting `-i=false`. Some CI/CD platforms set `CI=true` in their default environment. We could potentially skip the delay if this is set.

Do:
* https://circleci.com/docs/2.0/env-vars/#built-in-environment-variables
* https://docs.travis-ci.com/user/environment-variables/#default-environment-variables
* https://docs.semaphoreci.com/ci-cd-environment/environment-variables/#ci
* https://codefresh.io/docs/docs/codefresh-yaml/variables/#system-provided-variables
* https://docs.gitlab.com/ee/ci/variables/predefined_variables.html
* https://docs.github.com/en/actions/reference/environment-variables
* https://docs.cloudbees.com/docs/cloudbees-codeship/latest/basic-builds-and-configuration/set-environment-variables
* https://docs.drone.io/pipeline/environment/reference/ci/
* https://devcenter.wercker.com/administration/environment-variables/available-env-vars/
* https://buildkite.com/docs/pipelines/environment-variables#bk-env-vars-ci

Do Not:
* https://docs.microsoft.com/en-us/azure/devops/pipelines/build/variables?view=azure-devops&tabs=yaml
* https://cloud.google.com/build/docs/configuring-builds/substitute-variable-values
* https://docs.aws.amazon.com/codebuild/latest/userguide/build-env-ref-env-vars.html
* https://www.jetbrains.com/help/teamcity/predefined-build-parameters.html#Server+Build+Properties
* https://wiki.jenkins.io/display/JENKINS/Building+a+software+project#Buildingasoftwareproject-belowJenkinsSetEnvironmentVariables
* https://jenkins-x.io/docs/resources/guides/using-jx/pipelines/envvars/
* https://confluence.atlassian.com/bamboo/bamboo-variables-289277087.html
* https://docs.cloudbees.com/docs/cloudbees-codeship/latest/pro-builds-and-configuration/environment-variables#_default_environment_variables (not on pro version?)
* https://docs.gocd.org/current/faq/dev_use_current_revision_in_build.html#standard-gocd-environment-variables

There is concern that these protections are not enough as they require users to opt-in for confirmation. This KEP is viewed as a starting point to introduce some form of protection that users may start to use. A way for users to set this as the default in the future will be in a future KEP.

## Design Details

If `--all` or `--all-namespaces` flags are passed in for a delete operation a warning will be printed and an artificial delay will be imposed. We need to determine what the correct length is for this. Initial thoughts were between 5 and 10 seconds. User testing will be done to gauge how long is needed to read the warning message. If `CI=true` is set in the environment this delay can be skipped.

The warning message will be printed to stderr. We need to decide if we print a generic warning message or detail out the resources that will be deleted. The latter will be more complicated (especially calculating the resources deleted under a namespace) but more rewarding to the user. Deleting a namespace with `--dry-run=server` does not return any resources in the namespace.

> Warning! You are deleting resources with `--all`. This will delete all of the (pods) in the (cluster or current namespace). Proceeding in x seconds. ctrl+c to interrupt

> Warning! You are deleting the following resources:
> Name:    Group:        Kind:
> my-pod    core/v1      Pod
> Proceeding in x seconds. ctrl+c to interrupt

If the `--interactive` flag is passed for imperative deletes we will prompt the user for confirmation. As above we need to determine the prompt -- a generic message or explicitly lay out the resources. No (`N`) will be the default response if enter is pressed without input.

```
kubectl delete namespace prod -i
Are you sure? (y/N)
```

```
kubectl delete namespace prod -i
Are you sure you want to delete namespace prod? (y/N)
```

Setting `--interactive=false` will intentionally declare that a user does not want to be prompted or delayed. I believe [Cobra/Viper can determine if a flag has been explicitly set](https://github.com/spf13/cobra/issues/453#issuecomment-303548563).

### Test Plan

Unit tests, e2e, and integration tests will be implemented. We already have these suites in place for kubectl.

### Graduation Criteria

TBD

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

n/a

### Version Skew Strategy

n/a

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

###### How can this feature be enabled / disabled in a live cluster?

- [x] Other
  - Describe the mechanism: The `-i` flag is opt-in from the client. The delay is opt-out from the client.
  - Will enabling / disabling the feature require downtime of the control
    plane? n/a
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled). n/a

###### Does enabling the feature change any default behavior?

An x second delay will be introduced client-side when deleting with `--all | --all-namespaces`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

n/a

###### What happens if we reenable the feature if it was previously rolled back?

n/a

###### Are there any tests for feature enablement/disablement?

n/a

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

The delay introduced in this KEP will cause CI/CD jobs to take x seconds longer unless authors add `-i=false`. As stated above we can potentially mitigate this by skipping if `CI=true` is set.

## Alternatives

We considered showing the confirmation prompt if a TTY was detected. After further research it was determined that we would run into issues with TTY detection due to platforms and pipelines spoofing a TTY in an attempt to match the output of developer's terminals (e.g. color).

https://groups.google.com/g/kubernetes-dev/c/y4Q20V3dyOk/m/LOe7Id1DBgAJ
https://groups.google.com/g/kubernetes-dev/c/y4Q20V3dyOk/m/W0y2fD-NAAAJ

During RFC discussion several ideas were shared. One of them was a locking mechanism for resources that would prevent deletion. This is considered orthogonal to this KEP because users don't intend to delete the resources they are accidentally deleting - not that there are things that should never be deleted. 

A similar mechanism is preset in OpenKruise.

https://openkruise.io/en-us/docs/deletion_protection.html

Resource locking would make a great future KEP.

https://groups.google.com/g/kubernetes-dev/c/y4Q20V3dyOk/m/zTLxKK5ABgAJ
