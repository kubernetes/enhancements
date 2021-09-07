# KEP-1440: kubectl events

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Limitations of the Existing Design](#limitations-of-the-existing-design)
  - [Goals](#goals)
  - [Non-goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
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
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Presently, `kubectl get events` has some limitations. It cannot be extended to meet the increasing user needs to
support more functionality without impacting the `kubectl get`. This KEP proposes a new command `kubectl events` which will help
address the existing issues and enhance the `events` functionality to accommodate more features.

For example: Any modification to `--watch` functionality for `events` will also change
the `--watch` for `kubectl get` since the `events` is dependent of `kubectl get`

Some of the requested features for events include:

1. Extended behavior for `--watch`
2. Default sorting of `events`
3. Union of fields in custom-columns option
4. Listing the events timeline for last N minutes
5. Sorting the events using the other criteria as well

This new `kubectl events` command will be independent of `kubectl get`. This can be
extended to address the user requirements that cannot be achieved if the command is dependent of `get`.

## Motivation

A separate sub-command for `events` under `kubectl` which can help with long standing issues:
Some of these issues that be addressed with the above change are:

- User would like to filter events types
- User would like to see all change to the an object
- User would like to watch an object until its deletion
- User would like to change sorting order
- User would like to see a timeline/stream of `events`

### Limitations of the Existing Design

All of the issues listed below require extending the functionality of `kubectl get events`.
This would result in `kubectl get` command having a different set of functionality based
on the resource it is working with. To avoid per-resource functionality, it's best to
introduce a new command which will be similar to `kubectl get` in functionality, but
additionally will provide all of the extra functionality.

Following is a list of long standing issues for `events`

- kubectl get events doesn't sort events by last seen time [kubernetes/kubernetes#29838](https://github.com/kubernetes/kubernetes/issues/29838)
- Improve watch behavior for events [kubernetes/kubernetes#65646](https://github.com/kubernetes/kubernetes/issues/65646), [kubernetes/kubectl#793](https://github.com/kubernetes/kubectl/issues/793),
- Improve events printing [kubernetes/kubectl#704](https://github.com/kubernetes/kubectl/issues/704), [kubernetes/kubectl#151](https://github.com/kubernetes/kubectl/issues/151)
- Events query is too specific in describe [kubernetes/kubectl#147](https://github.com/kubernetes/kubectl/issues/147)
- kubectl get events should give a timeline of events [kubernetes/kubernetes#36304](https://github.com/kubernetes/kubernetes/issues/36304)
- kubectl get events should provide a way to combine ( Union) of columns [kubernetes/kubernetes#82950](https://github.com/kubernetes/kubernetes/issues/82950)

### Goals

- Add an experimental `events` sub-command under the kubectl
- Address existing issues mentioned above

### Non-goals

- This new command will not be dependent on `kubectl get`

## Proposal

Have an independent *events* sub-command which can perform all the existing tasks that the current `kubectl get events`,
and most importantly will extend the `kubectl get events` functionality to address the existing issues.

### User Stories

* As an application developer I want to view all the events related to a particular resource.
* As an application developer I want to watch for a particular event taking place.
* As an application developer I want to filter all warning events happening in a particular namespace.

### Risks and Mitigations

None.

## Design Details

The above use-cases call for the addition of several flags, that would act as filtering mechanisms for events,
and would work in tandem with the existing --watch flag:

- Add a new `--watch-event=[]` flag that allows users to subscribe to particular events, filtering out any other event kind
- Add a new `--watch-until=EventType` flag that would cause the `--watch` flag to behave as normal, but would exit the command as soon as the specified event type is received.
- Add a new `--watch-for=pod/bar flag` that would filter events to only display those pertaining to the specified resource. A non-existent resource would cause an error. This flag could further be used with the `--watch-until=EventType` flag to watch events for the resource specified, and then exit as soon as the specified `EventType` is seen for that particular resource.
- Add a new `--watch-until-exists=pod/bar` flag that outputs events as usual, but exits as soon as the specified resource exists. This flag would employ the functionality introduced in the wait command.

Additionally, the new command should support all the printing flags available in `kubectl get`, such as specifying output format, sorting as well as re-use server-side printing mechanism.

### Test Plan

In addition to standard unit tests for kubectl, the events command will be released as a kubectl alpha subcommand, signaling users to expect instability. During the alpha phase we will gather feedback from users that we expect will improve the design of debug and identify the Critical User Journeys we should test prior to Alpha -> Beta graduation.

### Graduation Criteria

Once the experimental kubectl events command is implemented, this can be rolled out in multiple phases.

##### Beta
- [ ] Gather the feedback, which will help improve the command
- [ ] Extend with the new features based on feedback

##### GA
- [ ] Address all major issues and bugs raised by community members

### Upgrade / Downgrade Strategy

This functionality is contained entirely within `kubectl` and shares its
strategy. No configuration changes are required.

### Version Skew Strategy

`kubectl events` will be using only GA features of the `Events` API from kube-apiserver,
so there should be no problems with Version Skew.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [X] Other
  - Describe the mechanism:
    A new command in `kubectl alpha`
  - Will enabling / disabling the feature require downtime of the control
    plane?
    No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
    No

###### Does enabling the feature change any default behavior?

It's a new command so there's no default behavior in kubectl. If a user
has installed a plugin named "events", that plugin will be masked by the
new `kubectl events` command. This is a known issue with kubectl plugins,
and it's being addressed separately by sig-cli, likely by detecting this
condition and printing a warning.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, you could roll back to a previous release of `kubectl`.

###### What happens if we reenable the feature if it was previously rolled back?

There will be explicit command for retrieving events.

###### Are there any tests for feature enablement/disablement?

No, because it cannot be disabled or enabled in a single release

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

- *2020-01-16* - Initial KEP draft
- *2021-09-06* - Updated KEP with the new template and mark implementable for alpha implementation.

## Alternatives

Currently available alternative exist in `kubectl describe` command and has been
described in [Limitations of the Existing Design](#limitations-of-the-existing-design).
