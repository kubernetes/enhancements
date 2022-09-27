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
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
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
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

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

- Add an new `events` sub-command under the kubectl
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

Accessing events to which users don't have access to. This should be mitigated by a proper RBAC rules
allowing access based on a need to know principle.

## Design Details

The above use-cases call for the addition of several flags, that would act as filtering mechanisms for events,
and would work in tandem with the existing --watch flag:

- Add a new `--watch-event=[]` flag that allows users to subscribe to particular events, filtering out any other event kind
- Add a new `--watch-until=EventType` flag that would cause the `--watch` flag to behave as normal, but would exit the command as soon as the specified event type is received.
- Add a new `--watch-for=pod/bar flag` that would filter events to only display those pertaining to the specified resource. A non-existent resource would cause an error. This flag could further be used with the `--watch-until=EventType` flag to watch events for the resource specified, and then exit as soon as the specified `EventType` is seen for that particular resource.
- Add a new `--watch-until-exists=pod/bar` flag that outputs events as usual, but exits as soon as the specified resource exists. This flag would employ the functionality introduced in the wait command.

Additionally, the new command should support all the printing flags available in `kubectl get`, such as specifying output format, sorting as well as re-use server-side printing mechanism.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

Before any additional functional updates we need to ensure the current functionality
is properly cover with unit and integration (`test/cmd`) tests.
Before promoting to beta at least a single e2e test should also be added in
`k8s.io/kubernetes/test/e2e/kubectl/kubectl.go`.

##### Unit tests

- `k8s.io/kubectl/pkg/cmd/events`: `2022-09-21` - `36.6%`

##### Integration tests

- `k8s.io/kubernetes/test/cmd/events.sh`: [test-cmd.run_kubectl_events_tests](https://testgrid.k8s.io/sig-release-master-blocking#integration-master)

##### e2e tests

- missing

### Graduation Criteria

Once the experimental kubectl events command is implemented, this can be rolled out in multiple phases.

##### Beta

- [x] Add e2e tests, increase unit coverage.
- [x] Gather the feedback, which will help improve the command
- [x] Extend with the new features based on feedback

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
    A new sub-command in `kubectl`
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

No, because it cannot be disabled or enabled in a single release.

### Rollout, Upgrade and Rollback Planning

None, kubectl rollout requires just shipping a new binary.

###### How can a rollout or rollback fail? Can it impact already running workloads?

A wrong binary might be delivered.

###### What specific metrics should inform a rollback?

There are no metrics to follow.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

E2E which will be added with beta promotion will allow us to verify if the command
behaves correctly during upgrade and downgrade.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

The `kubectl alpha events` is being moved under `kubectl events`. Invoking the old
location will print a warning that this command moved.

### Monitoring Requirements

None.

###### How can an operator determine if the feature is in use by workloads?

There is no way cluster operator to differentiate between `kubectl get events` and `kubectl events`
invocations since both invoke a GET operation on Events endpoint.

###### How can someone using this feature know that it is working for their instance?

`kubectl events` should be returning events similar to `kubectl get events`.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

None.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Other (treat as last resort)
  - Details: invoking `kubectl events` returns data in a timely manner

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

None.

### Dependencies

None.

###### Does this feature depend on any specific services running in the cluster?

None.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new API calls are expected if compared with `kubectl get events`.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Running `kubectl events` with unavailable API server and/or etcd will result
in an error reported to user stating that the cluster is not available.

###### What are other known failure modes?

- [No events]
  - Detection: Invoking `kubectl events` does not return any events.
  - Mitigations: Use `kubectl get events` instead.
  - Diagnostics: Compare with the output of `kubectl get events`. It's possible that
    there are no events in given namespace. Alternatively, use different namespace
    with `--namespace` flag.

###### What steps should be taken if SLOs are not being met to determine the problem?

None.

## Implementation History

- *2020-01-16* - Initial KEP draft
- *2021-09-06* - Updated KEP with the new template and mark implementable for alpha implementation.
- *2022-09-21* - Updated KEP for beta promotion.

## Alternatives

Currently available alternative exist in `kubectl describe` command and has been
described in [Limitations of the Existing Design](#limitations-of-the-existing-design).
