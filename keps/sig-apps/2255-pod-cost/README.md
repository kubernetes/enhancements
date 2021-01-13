# KEP-2255: Add pod-cost annotation for ReplicaSet


<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
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
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This feature allows making a suggestion to the ReplicaSet controller, which pod of a Deployment should be deleted first when a scale-down event happens. This can prevent session disruption in stateful applications in a trivial manner.

## Motivation

For some applications, it is necessary that the application can tell Kubernetes which pod can be deleted and which replica has to be protected. The reason for this is that some applications do have stateful sessions and it is not possible to put such an application into Kubernetes because of session termination resulting from "random" down-scale. If the application is able to tell Kubernetes which of the replicas contains no/few/less important active sessions, this would solve many problems. This feature is non-disruptive to the default behaviour. Only if the annotation is existing, it will make a difference in deletion order.

### Goals

To recommend which pod gets deleted next of a ReplicaSet. This should help to avoid major reworks in existing applications architecture:
* [45509](https://github.com/kubernetes/kubernetes/issues/45509) - Scale down a deployment by removing specific pods


### Non-Goals

Guaranteed (in contrast to the recommendation stated in Goals) deletion of a selected replica.

## Proposal

The application can set the `controller.kubernetes.io/pod-cost` annotation to a pod through the Kubernetes API. When a downscale event happens, the pod with the lower priority value of the previously set annotation will be deleted first. If one pod of the Deployment has no priority annotation set, it will be treated as the lowest priority.

If all pods have the same priority, there is no difference in the normal pod delete decision behaviour. The same applies if the pod-cost annotation is not used at all.

The pod-cost annotation can be changed during operation, for example, if workload changes or a new master gets elected.

### User Stories (optional)


#### Story 1

In an application environment with stateful worker (user-)sessions, it is essential to keep the user sessions alive as good as possible. In case of a scale-down event, the application has to tell the scheduler, which delete decision would have the lowest impact on existing sessions.

#### Story 2

An application consists of identical server processes, but one of the replicas will be the master, which should be kept as long as possible. All other replicas can be treated as cattle workload. Then the master can set the priority annotation with a high priority value as soon as it has finished its startup process. The other replicas can remain either without any priority set, or e.g. with all the same, lower priority. This ensures, that the master replica of this deployment will be protected in a downscale situation.


### Risks and Mitigations

On previous Kubernetes ReplicaSet controller versions that don't implement the pod-cost annotation feature, the same application might make false assumptions about the protection of a master instance or workers with open (user-)sessions on it. As the pod-cost annotation would be only a suggestion to the ReplicaSet controller, the application developer should, however, handle the case of a failed master instance or broken user sessions. The feature is just an improvement, not a guarantee, as there might happen timing issues between setting the annotation and the next controller scale-down event.

## Design Details


### Test Plan

* Units test in kube-controller-manager package to test a variety of scenarios.
* New E2E Tests to validate that replicas get deleted as expected e.g:
 * Replicas with lower pod-cost before replicas with higher pod-cost
 * Replicas with no pod-cost annotation set before replicas with low priority

### Graduation Criteria

#### Alpha -> Beta Graduation
* Implemented feedback from alpha testers
* Thorough E2E and unit testing in place

#### Beta -> GA Graduation
* Significant number of end-users are using the feature
* We're confident that no further API changes will be needed to achieve the goals of the KEP
* All known functional bugs have been fixed

### Upgrade / Downgrade Strategy

When upgrading no changes are needed to maintain existing behaviour as all of this behaviour is fully optional and disabled by default. To activate this feature either a user has to make an annotation to a pod in a Deployment by hand or the application annotates a pod in a Deployment through the API.

When downgrading, there is no need to changing anything, as this is just a pod annotation, which is uncritical.

### Version Skew Strategy

As this feature is based on pod annotations, there is no issue with different Kubernetes versions. The lack of this feature in older versions may change the efficiency and reliability of the applications.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Other
    - Make special pod annotations within a live Deployment


* **Does enabling the feature change any default behavior?**
  - No


* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  - One can either remove the annotations or downgrade to an older Kubernetes release


* **What happens if we reenable the feature if it was previously rolled back?**
  - Then the feature will be reenabled. Nothing special to consider here.


* **Are there any tests for feature enablement/disablement?**


### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  - As the feature is a simple annoation, the worst what could happen is that either the annotation is lost or ignored. In the worst case, a pod with a higher priority gets deleted before a pod with a lower priority.


* **What specific metrics should inform a rollback?**
  - None


* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  - Was tested. Behaviour change in both directions, as expected.


* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  - No. However, the exact same pod annotation string cannot be used for any other purposes.


### Monitoring requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  - Search for pod annotations with the exact same pod-cost annotation string.


* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  - A pod with a lower pod-cost annotation in a Deployment gets deleted first on a scale-down event.


* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  - All pods with a lower pod-cost annotation in a Deployment are deleted first on a scale-down event.

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**
  - N/A

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  - The feature requires the existing of the kube-controller-manager and the ability and permissions to set pod annotations.


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirms the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  - Whenever the application decides, that a change in pod-cost is needed for a replica, it will send out an API request and set the appropriate pod annotation(s).


* **Will enabling / using this feature result in introducing new API types?**
  - No.


* **Will enabling / using this feature result in any new calls to cloud
  provider?**
  - No.


* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**
  Describe them providing:
  - API type(s): Pod annotation
  - Estimated increase in size: Size of a new annotation
  - Estimated amount of new objects: new annotation for potentially every existing Pod


* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  - The time it takes to set/delete/change a pod annotation


* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  - The resources it takes to set/delete/change a pod annotation


### Troubleshooting

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
  - The pod annotation can't be set. The normal pod deletion behavior will be used for non-annotated pods in a Deployment.
* **What are other known failure modes?**
  - None.

* **What steps should be taken if SLOs are not being met to determine the problem?**
  - N/A

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History


## Drawbacks


## Alternatives

Similar behaviour can be achieved through the Operator Framework which however will take a lot more configuration and setup work and is not a built-in Kubernetes feature.
