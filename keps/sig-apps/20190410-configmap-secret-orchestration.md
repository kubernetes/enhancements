---
title: ConfigMap / Secret Orchestration
authors:
  - "@kfox1111"
owning-sig: sig-apps
participating-sigs:
  - sig-architecture
  - sig-bbb
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-04-10
last-updated: 2019-04-10
status: provisional|implementable|implemented|deferred|rejected|withdrawn|replaced
see-also: []
replaces: []
superseded-by: []
---

# ConfigMap / Secret Orchestration

configmap-secret-orchestration

To get started with this template:

## Table of Contents

- [Title](#title)
  - [Table of Contents](#table-of-contents)
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [User Stories [optional]](#user-stories-optional)
      - [Story 1](#story-1)
      - [Story 2](#story-2)
    - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
    - [Test Plan](#test-plan)
    - [Graduation Criteria](#graduation-criteria)
      - [Examples](#examples)
        - [Alpha -> Beta Graduation](#alpha---beta-graduation)
        - [Beta -> GA Graduation](#beta---ga-graduation)
        - [Removing a deprecated flag](#removing-a-deprecated-flag)
    - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Version Skew Strategy](#version-skew-strategy)
  - [Implementation History](#implementation-history)
  - [Drawbacks [optional]](#drawbacks-optional)
  - [Alternatives [optional]](#alternatives-optional)
  - [Infrastructure Needed [optional]](#infrastructure-needed-optional)

## Release Signoff Checklist

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

The `Summary` section is incredibly important for producing high quality user-focused documentation such as release notes or a development roadmap.
It should be possible to collect this information before implementation begins in order to avoid requiring implementors to split their attention between writing release notes and implementing the feature itself.
KEP editors, SIG Docs, and SIG PM should help to ensure that the tone and content of the `Summary` section is useful for a wide audience.

A good summary is probably at least a paragraph in length.

Kubernetes added support for higher level orchestration in the form of ReplicaSets/Deployments/DaemonSets/Statefulsets. They implement best practices for things including performing rolling upgrades and ensuring the containers are immutable. The orchestration resources currently do not fully implement a flow that provides immutable infrastructure without extra careful work by the user. Config/Secrets are injected into the immutable containers but aren't tracked along with the roll-forward/roll-back. Since config/secrets can change over time, its often the case where the user updates the config/secret for the new revision of the Deplyment but rollbacks fail as they now reference the new config file instead of the one they were deployed with. The other problem with config/secret orchestration is that of orchestrated updates of the immutable infra can only be triggered manally by changes. A config/secret only change can not trigger an upgrade either manually or automatically without making an unnessisary change to the corresponding orchestration object.

We would like to solve these issues by adding support to the orchestration resources to support orchestration support around config/secrets.


## Motivation

Currently there are a mismash of workarounds to the orchestration objects not supporting orchestration of configmaps/secrets.

Helm recommends you take a checksum of the config and add it as an annotation to the orchestration object. This triggers and update of the workload when the content of the configmap changes. This does not properly work with roll backs. It also does not work if the configmap's definition is in a nested chart or provided outside the chart.

### Goals

* Enhance the deployment/statefulset/daemonset with a flag asking for the workload to be automatically updated when a specified configmap/secret changes.
* Enhance the deployment/statefulset/daemonset with a flag asking for the specified configmap/secret's current state to be snapshotted and follow the orchestrated workload (roll-forward/roll-back)
* Implementations of the api changes

### Non-Goals

* Out of tree solutions. For applications to be portable, it really needs to be in Kubernetes proper so application developers know the functionality is always there.

## Proposal

The ConfigMapVolumeSource and SecretVolumeSource objects will get two new, optional fields:
 * snapshot: boolean
 * watch: boolean

Both will default to false for compatability.

snapshot can be either true or false. Watch can only be true if snapshot is also true. This ensures immutability.

The fields will be ignored by all objects other then Deployment, DaemonSet and StatefulSet.

DaemonSet/StatefulSet controllers will be modified such that:
 * During a "change", on seeing a snapshot=true flag on a configmap/secret, a copy of the configmap/secret will be made with a unique name. This unique name will be stored in the corresponding Controller revision.
 * All pods created will reference the unique configmap/secret snapshot name, not the base name.
 * When a ControllerRevision is deleted, a deletion of the corresponding configmap/secret snapshots will also be issued.
 * When an object with a watch flag of true is created/updated, watches on the specified configmap/secret will be added. Any watch triggered will be treated as a "change" of the object.

The Deployment controller will be modified such that:
 * During a "change", on seeing a snapshot=true flag on a configmap/secret, a copy of the configmap/secret will be made with a unique name. This unique name will be stored in the corresponding ReplicaSet.
 * When a ReplicaSet is deleted, a deletion of the corresponding configmap/secret snapshots will also be issued.
 * When an object with a watch flag of true is created/updated, watches on the specified configmap/secret will be added. Any watch triggered will be treated as a "change" of the object.

### User Stories [optional]

#### Story 1

As a user, I deploy an application by checking out a git repository and performing a "kubectl apply -f .". I then perform 'kubectl edit configmap foo' and change some settings. I would like for the settings to be applied to the workload automatically and consistently. I may then decide that the change I made was in error. I would like "kubectl rollout undo deployment.v1.apps/foo" to take the deployment safely back to the previous state.

### Implementation Details/Notes/Constraints [optional]

TODO

### Risks and Mitigations

What are the risks of this proposal and how do we mitigate.
Think broadly.
For example, consider both security and how this will impact the larger kubernetes ecosystem.

How will security be reviewed and by whom?
How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.

## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.
Anything that would count as tricky in the implementation and anything particularly challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage expectations).
Please adhere to the [Kubernetes testing guidelines][testing-guidelines] when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md

### Graduation Criteria

These are generalized examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

##### Alpha -> Beta Graduation

- In the alpha, annotations should be used. Migrate them to proper api changes.
- Gather feedback from developers and surveys
- Complete all features
- Tests are in Testgrid and linked in KEP

##### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

### Upgrade / Downgrade Strategy

Upgrade strategy:
- Completely new feature. Nothing to do

Downgrade strategy:
- Remove the flags from objects.
- References will be left in place and garbage collection will not automatically reap the objects but everything else will continue to work.

### Version Skew Strategy

There should be no version skew issues as the flags are only interpreted by the controllers.

## Implementation History

- KEP Started - Apr 10 2019

## Drawbacks

This functionality requires more complexity to be added to orchestration controllers. People have, in some cases, worked around these problems successfully outside of Kubernetes.

## Alternatives [optional]

There are several external projects implementing various solutions or workarounds for this problem. This includes:
* https://github.com/stakater/reloader
* https://github.com/xing/kubernetes-deployment-restart-controller
* https://github.com/mattmoor/boo-maps#kubernetes-mutablemap-and-immutablemap-crds
* helm and kustomize also are known to have workarounds

