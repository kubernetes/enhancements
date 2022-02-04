# KEP-2804: Consolidate Workload controllers life cycle status

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
  - [Overview of all conditions](#overview-of-all-conditions)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Proposed Conditions](#proposed-conditions)
    - [Progressing](#progressing)
    - [Complete](#complete)
    - [Failed](#failed)
    - [Available](#available)
    - [Batch Workloads Conditions: Waiting &amp; Running](#batch-workloads-conditions-waiting--running)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist


Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
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

The main goal of this enhancement is to compare all the workload conditions and consolidate the workload controller lifecycle
state. The secondary goal is to identify and expose other conditions that could bring benefit to the users.
This includes defining conditions (Waiting, Running, Progressing, Available) for:
- Deployment
- DaemonSet
- ReplicaSet & ReplicationController
- StatefulSet
- Job

## Motivation

Today only deployment controller has a [status](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#deployment-status) to fully reflect the state during its lifecycle.
This enhancement extends the scope of those and other conditions to other controllers (DaemonSet, Job, ReplicaSet & ReplicationController, StatefulSet).

### Goals

The current status of a workload can be depicted via its conditions. It can be a subset of:
- Progressing: designates the state of the latest rollout.
- Available: designates if the required number of available replicas is `available`.
- ReplicaFailure: ReplicaSet failed to create/delete a Pod.
- Suspended: execution of a Job is suspended.
- Complete: Job run to a completion, or rollout completed (via Progressing condition).
- Failed: Job failed to complete, or rollout failed to progress (via Progressing condition).
- Waiting (Job only): waiting for at least one Pod to run.
- Running (Job only): at least one Pod of a Job is running.

Workload controllers should have above conditions (when applicable) to reflect their states.

### Non-Goals

- Modifying the existing states of deployment controller
- Changing the definition of statuses
- Introduce new api for existing conditions
- To properly implement Progressing condition. `.spec.progressDeadlineSeconds` field has to be introduced as part of an additional KEP in
  DaemonSet and StatefulSet to describe the time when the controllers should declare the workload as `failed`.
- consider adding Conditions field to CronJob

## Proposal

### User Stories (Optional)

#### Story 1
As an end-user of Kubernetes, I'd like all my workload controllers to have consistent statuses.

#### Story 2
As an developer building Kubernetes Operators, I'd like all my operators deployed to have
consistent statuses.


### Overview of all conditions

The following table gives an overview on what conditions each of the workload resources support.

|                                    | Progressing | Available |  ReplicaFailure | Suspended | Failed | Complete |
| ---------------------------------  | ----------- | --------- | --------------- | --------- | ------ | -------- |
| ReplicaSet & ReplicationController | -           | -         |  failed to create / delete pod (FailedCreate, FailedDelete)  | -         | -           | -        |
| Deployment                         | True when scaling replicas / creating-updating new ReplicaSet / successfully finished progressing (Pods ready or available for MinReadySeconds). False when failed creating ReplicaSet / reached progressDeadlineSeconds. Unknown when rollout paused | True if if required number of replicas is available (takes MaxSurge and MaxUnavailable into account) | failure propagated from new or old ReplicaSet | -         | -*        | -*          |
| StatefulSet                        | -           | -         | -               | -         | -      | -        |
| DaemonSet                          | -           | -         | -               | -         | -      | -        |
| Job                                | -           | -         | -               | True / False when suspended / resumed | failed execution  (BackoffLimitExceeded, DeadlineExceeded)| completed execution |
| CronJob**                          | -           | -         | -               | -         | -      | -        |

**\* Success of the rollout is instead represented by a Progressing condition (status and reason)**

**\*\* CronJob does not even have Conditions field in its Status**

### Notes/Constraints/Caveats (Optional)

As observed in some issues (https://github.com/kubernetes/website/pull/31226) and talking to the users, there is a misunderstanding about the meaning of the `Progressing` condition. These include:
- Thinking that the `Progressing` condition reflects the state of the current Deployment instead of the last rollout. Which includes expectation that the `Progressing` condition will keep checking availability of replicas and revert to `progressing`/`failed` state even after the `complete` state is reached. And that the progressing condition will thus also reflect any changes in scaling.
- Confusion that ProgressDeadlineExceeded does not occur after the Deployment rollout completes when the availability of pods degrades before the  `.spec.progressDeadlineSeconds` times out.

To summarize, the name of the `Progressing` condition doesn't indicate its true meaning that its main responsibility is the indication of rollouts, and it confuses the users.

### Proposed Conditions

We are proposing 4 new conditions to be added to the following controllers:
- Available (DaemonSet, ReplicaSet & ReplicationController, StatefulSet)
- Progressing (DaemonSet, StatefulSet)
- Waiting (Job)
- Running (Job)

The definitions for Progressing condition (including Failed/Complete) are similar to what we have for [Deployment controller](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#deployment-status).


The following table is indicating what conditions are currently available (`X`) and what conditions will be added (`A`).

|                                    | Waiting | Running | Progressing | Available |  ReplicaFailure | Suspended | Failed | Complete |
| ---------------------------------  | --------| ------- | ----------- | --------- | --------------- | --------- | ------ | -------- |
| ReplicaSet & ReplicationController | -       | -       | -           | A         | X               | -         | -      | -        |
| Deployment                         | -       | -       | X           | X         | X               | -         | -      | -        |
| StatefulSet                        | -       | -       | A           | A         | -               | -         | -      | -        |
| DaemonSet                          | -       | -       | A           | A         | -               | -         | -      | -        |
| Job                                | A       | A       | -           | -         | -               | X         | X      | X        |
| CronJob                            | -       | -       | -           | -         | -               | -         | -      | -        |

#### Progressing
Individual workload controllers mark a DaemonSet or Stateful as `progressing` when:
- The DaemonSet or StatefulSet is created
- The DaemonSet or StatefulSet is scaling up or scaling down
- New DaemonSet or StatefulSet pods become Ready or available

#### Complete
Individual workload controllers mark a DaemonSet, ReplicaSet or Stateful as `complete` when it has the following characteristics:

- All of the replicas/pods associated with the DaemonSet or StatefulSet have been updated to the latest version you've specified, meaning any updates you've requested have been completed.
- All of the replicas/pods associated with the DaemonSet, ReplicaSet or StatefulSet are available.
- No old or mischeduled replicas/pods for the DaemonSet, ReplicaSet or Stateful are running.

#### Failed
In order to introduce this condition we need to come up with a new field called `.spec.progressDeadlineSeconds` (additional KEP) which denotes the number of seconds the controller waits before indicating(in the workload controller status) that the controller progress has stalled.

There are many factors that can cause failure to happen like:
- Insufficient quota
- Readiness probe failures
- Image pull errors
- Failed to create/delete pod

See the [Kubernetes API Conventions](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties) for more information on status conditions

Because of the number of changes that are involved as part of this effort, we are thinking of a phased approach where we introduce these conditions to DaemonSet controller first and then make similar changes to ReplicaSet and StatefulSet controller. We would graduate ExtendedWorkloadConditions to beta once all the features and `progressDeadlineSeconds` KEP are implemented.
This also needs some code refactoring of existing conditions for Deployment controller.

#### Available
Individual workload controllers mark a ReplicaSet or StatefulSet as `available` when number of available replicas reaches number of replicas.
- This could be confusing in ReplicaSets a bit since Deployment could get available sooner than its ReplicaSet due `maxUnavailable`.
- Available replicas is alpha feature guarded by StatefulSetMinReadySeconds gate in StatefulSets, but the value defaults to ReadyReplicas when the feature gate is disabled so using it shouldn't be an issue.

DaemonSet controller marks DaemonSet as `available` when `numberUnavailable` or `desiredNumberScheduled - numberAvailable` is zero.

#### Batch Workloads Conditions: Waiting & Running

Batch workloads behaviour does not properly map to the other workloads that are expected to be always running (eg. `Progressing` condition and its behaviour).
- Jobs are indicating a `Failed`/`Complete` state in a standalone condition compared to `Progressing` condition in other workloads.
- `.spec.activeDeadlineSeconds` variable, is similar to `progressDeadlineSeconds`, but does not have a default value.
  It also resets on suspension, so its behaviour is a bit different.


Job controller marks a Job as `waiting` if one of the following conditions is true:

- There are no Pods with phase `Running` and there is at least one Pod with phase `Pending`.

Job controller marks a Job as `running` if there is at least one Pod with phase `Running`.

This KEP does not introduce CronJob conditions as it is difficult to define conditions that would describe CronJob behaviour in useful manner.
In case the user is interested if there are any running Jobs, `.status.active` field should be sufficient.

### Risks and Mitigations

We are proposing a new field called `.spec.progressDeadlineSeconds` to DaemonSet and StatefulSet as part of a additional KEP whose default value will be set to the max value of int32 (i.e. 2147483647) by default, which means "no deadline".
In this mode, DaemonSet and StatefulSet controllers will behave exactly like its current behavior but with no `Failed` mode as they're either `Progressing` or `Complete`.
This is to ensure backward compatibility with current behavior. We will default to reasonable values for the controllers in a future release.
Since a DaemonSet can make progress no faster than "healthy but not ready nodes", the default value for `progressDeadlineSeconds` can be set to 30 minutes but this value can vary depending on the node count in the cluster.
The value for StatefulSet can be longer than 10 minutes since it also involves provisioning storage and binding. This default value can be set to 15 minutes in case of StatefulSet.

It is possible that we introduce a bug in the implementation. The bug can cause:
- DaemonSet and StatefulSet controllers can be marked `Failed` even though rollout is in progress
- The states could be misrepresented, for example a DaemonSet or StatefulSet can be marked `Progressing` when actually it is complete

The mitigation currently is that these features will be in Alpha stage behind `ExtendedWorkloadConditions` featuregate for people to try out and give feedback. In Beta phase when
these are enabled by default, people will only see issues or bugs when `progressDeadlineSeconds` is set to something greater than 0 and we choose default values for that field.
Since people would have tried this feature in Alpha, we would have had time to fix issues.


## Design Details

### Test Plan
Unit and E2E tests will be added to cover the
API validation, behavioral change of controllers with feature gate enabled and disabled.

- Validating all possible states of old and new conditions. Checking that the changes in underlying Pod statuses correspond to the conditions.
- Testing `progressDeadlineSeconds` and feature gates.


### Graduation Criteria

#### Alpha
- Complete feature behind featuregates
- Have proper unit, integration and e2e tests 

#### Alpha -> Beta Graduation
- Gather feedback from end users
- Tests are in Testgrid and linked in KEP
- all new features in the following controllers should be implemented: ReplicaSet & ReplicationController, StatefulSet, DaemonSet and Job. To fully support `failed` state of a progressing condition, `progressDeadlineSeconds` KEP should be also fully implemented.

#### Beta -> GA Graduation
- 2 examples of end users using this field

### Upgrade / Downgrade Strategy

- Upgrades
 When upgrading from a release without this feature, to a release with `ExtendedWorkloadConditions` feature,
 we will set new conditions on the mentioned workloads.
- Downgrades
 When downgrading from a release with this feature, to a release without, 
 we expect controllers to honor the existing handling behaviour and not to remove the stale conditions.

### Version Skew Strategy

The update of extended conditions is always dependent on a `ExtendedWorkloadConditions` feature gate and not on the version as such.
If the feature gate is enabled, the workload controllers will update the extended conditions to reflect the current state.
In case feature gate is disabled or the feature is missing, the conditions will not be removed and become stale.
This feature has no node runtime implications.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ExtendedWorkloadConditions
  - Components depending on the feature gate:
    - kube-controller-manager

###### Does enabling the feature change any default behavior?
No. The default behavior won't change. Only new conditions will be added with no effect on existing conditions.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?
 Yes. Using the featuregates is the only way to enable/disable this feature

###### What happens if we reenable the feature if it was previously rolled back?

When we disable a feature gate the extended conditions are expected to become stale and still be present in the statuses of workload objects.
Once we reenable the feature gate, the controllers should start updating the new workload conditions again.

###### Are there any tests for feature enablement/disablement?

Yes, unit, integration and e2e tests for feature enabled, disabled

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

###### How can someone using this feature know that it is working for their instance?

- [ ] API .status
  - Condition name: Progressing, Available, Waiting, Running

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

### Scalability

###### Will enabling / using this feature result in any new API calls?
No, the number of API calls will stay the same as we will reuse already existing status update calls. 
This is because other fields in the status influence the conditions.
But the size of the patches in these calls will increase.

###### Will enabling / using this feature result in introducing new API types?

No.
###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?
Yes.
API type(s): DaemonSet, Deployment, Job, ReplicaSet, ReplicationController, StatefulSet
  - Estimated increase in size:
    - On average, we are going to add 1 condition per workload, approximately 100 bytes for each condition.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. We will be adding new statuses but this should not affect existing SLIs/SLOs as the logic should be part of the already existing flow of updating other status fields.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

## Drawbacks
Adds more complexity to Deployment, DaemonSet, Job, ReplicaSet, ReplicationController, StatefulSet controllers in terms of checking conditions and updating the conditions continuously.

## Alternatives
Continue to check AvailableReplicas, Replicas and other fields instead of having explicit conditions. This is not always foolproof and can cause problems.

## Infrastructure Needed (Optional)
