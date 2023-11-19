# KEP-4341: Eviction Manager should check disk usage of dead containers

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
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Troubleshooting](#troubleshooting)
- [Drawbacks](#drawbacks)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


## Summary

Node pressure eviction is the process by which the kubelet proactively terminates pods to
reclaim resources on nodes. The kubelet monitors resources like memory, disk space, and 
filesystem inodes on our cluster's nodes. The Kubelet uses various parameters like eviction 
signals and eviction thresholds to make eviction decisions.

The kubelet report node conditions to reflect that the node is under pressure because hard or 
soft eviction threshold is met, like node condition of memory pressure is caused when 
available memory on the node has satisfied an eviction threshold and disk pressure is caused 
when available disk space has satisfied an eviction threshold.

It seems that under the condition of disk pressure, Eviction-manager checks disk usage of 
only living containers of a pod, then decide which pod should be evicted. If in the env, there 
are dead containers that are not running but consuming the disk space then at the time of the 
disk pressure on node, the Eviction manager will not check the disk usage of those dead 
containers due to which the living containers are evicted first. In idol case, the Eviction 
manager should first evict the dead containers consuming disk space so that the living 
containers could be saved from eviction.

For more information and results see this issue:
https://github.com/kubernetes/kubernetes/issues/115201

## Motivation

Eviction Manager checks usage of only living containers, there is no problem in Memory 
Pressure, since memory used by containers are freed up when they are dead but disk space 
aren't freed up even after they are dead. Due to this the Eviction Manager evicts the wrong pods.

For example there are two Pods, Pod A and Pod B. 
Pod A has 1 living container whose disk usage is 5 GB
Pod B has 1 living container whose disk usage is 0 GB and 1 dead container whose disk usage is 10 GB 
In current scenario, under the condition of disk pressure, Eviction manager attempts to evict 
Pod A(total disk usage is less than Pod B) prior to Pod B

The expectation of this enhancement is that in disk pressure case, Eviction manager checks 
disk usage of living and dead containers of a pod and then decide which pod should be evicted (Pod B in above case)


### Goals

Eviction manager should check disk usage of both living and dead containers of the pod.

Implementing this could save the eviction of wrong pods.

For future, it will be an improvement in the Node pressure Evction and Garbage Collection.

### Non-Goals

The eviction related to memory pressure is not in the scope of this enhancement, it is related to disk pressure only.

## Proposal

The enhancement is in provisional phase. We want this enhancement to be implemented as due to the current behaviour of Eviction manager, bugs could arise when the wrong and unexpected pods 
are evicted. It will be better if eviction manager is checking the total disk usage of the pod instead of checking the living containers of pod only. 

Previously, this issue is opened regarding this proposal and community asked to open the enhancement. Help from the community will be needed for implementing this enhancement

### Risks and Mitigations


## Design Details



### Test Plan

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates


##### Unit tests





- `<package>`: `<date>` - `<test coverage>`

##### Integration tests


- <test>: <link to test coverage>

##### e2e tests



- <test>: <link to test coverage>

### Graduation Criteria



### Upgrade / Downgrade Strategy


## Production Readiness Review Questionnaire



### Monitoring Requirements


###### How can an operator determine if the feature is in use by workloads?


###### How can someone using this feature know that it is working for their instance?



- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?



### Dependencies


###### Does this feature depend on any specific services running in the cluster?



###### Will enabling / using this feature result in any new API calls?

###### Will enabling / using this feature result in introducing new API types?

###### Will enabling / using this feature result in any new calls to the cloud provider?

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?
No



### Troubleshooting




## Drawbacks

This enhancement should be implemented and currently we have no reason for not implementing this but the suggestions are always welcome.


