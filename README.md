# Enhancement Tracking and Backlog

Enhancement tracking repo for Kubernetes releases. Owned by [PM SIG](https://github.com/kubernetes/community/blob/master/sig-pm/README.md).

This repo contains issues and [KEPs](https://git.k8s.io/enhancements/keps). These issues are umbrellas for new enhancements to be added to Kubernetes. An enhancement usually takes multiple releases to complete. And an enhancement can be tracked as backlog items before work begins. An enhancement may be filed once there is consensus in at least one [Kubernetes SIG](https://git.k8s.io/community/sig-list.md).

## Is My Thing an Enhancement?

We are trying to figure out the exact shape of an enhancement. Until then here are a few rough heuristics.

An enhancement is anything that:

- a blog post would be written about after its release (ex. [minikube](https://kubernetes.io/blog/2016/07/minikube-easily-run-kubernetes-locally/), [StatefulSets](https://kubernetes.io/blog/2016/07/thousand-instances-of-cassandra-using-kubernetes-pet-set/), [rkt container runtime](https://kubernetes.io/blog/2016/07/rktnetes-brings-rkt-container-engine-to-kubernetes/))
- requires multiple parties/SIGs/owners participating to complete (ex. GPU scheduling [API, Core, & Node], StatefulSets [Storage & API])
- will be graduating from one stage to another (ex. alpha to beta, beta to GA)
- needs significant effort or changes Kubernetes in a significant way (ex. something that would take 10 person-weeks to implement, introduce or redesign a system component, or introduces API changes)
- impacts the UX or operation of Kubernetes substantially such that engineers using Kubernetes will need retraining
- users will notice and come to rely on

It is unlikely an enhancement if it is:
- implemented using `CustomResourceDefinition` and/or in https://github.com/kubernetes/contrib
- fixing a flaky test
- refactoring code
- performance improvements, which are only visible to users as faster API operations, or faster control loops
- adding error messages or events

If you are not sure, ask someone in the SIG where you initially circulated the idea. If they aren't sure jump into
[#sig-pm](https://kubernetes.slack.com/messages/sig-pm/) on Slack or ping someone listed in [OWNERS](https://github.com/kubernetes/enhancements/blob/master/OWNERS).

## When to Create a New Enhancement Issue

Create an issue here once you:
- have circulated your idea to see if there is interest
   - through Community Meetings, SIG meetings, SIG mailing lists, or an issue in github.com/kubernetes/kubernetes
- (optionally) have done a prototype in your own fork
- have identified people who agree to work on the enhancement
  - many enhancements will take several releases to progress through Alpha, Beta, and Stable stages
  - you and your team should be prepared to work on the approx. 9mo - 1 year that it takes to progress to Stable status
- are ready to be the project-manager for the enhancement

## Why are Enhancements Tracked

Once users adopt an enhancement, they expect to use it for an extended period of time. Therefore, we hold new enhancements to a
high standard of conceptual integrity and require consistency with other parts of the system, thorough testing, and complete
documentation. As the project grows no single person can track whether all those requirements are met. The development
of an enhancement often spans three stages: Alpha, Beta, and Stable; Enhancement Tracking Issues provide a
checklist that allows for different approvers for different aspects, and ensures that nothing is forgotten across the
development lifetime of an enhancement.

## When to Comment on an Enhancement Issue

Please comment on the enhancement issue to:
- request a review or clarification on the process
- update status of the enhancement effort
- link to relevant issues in other repos

Please do not comment on the enhancement issue to:
- discuss a detail of the design, code or docs. Use a linked-to-issue or PR for that

## Enhancements Tracking Spreadsheet

All the enhancements from this repo are visualized in the Enhancements Tracking Spreadsheets.

Links:
- [1.18 Milestone](https://bit.ly/k8s-1-18-enhancements)
- [1.17 Milestone](https://bit.ly/k8s117-enhancement-tracking)
- [1.16 Milestone](https://bit.ly/k8s116-enhancement-tracking)
- [1.15 Milestone](https://bit.ly/115-enhancements)
- [1.14 Milestone](https://bit.ly/k8s114-enhancements)
- [1.13 Milestone](https://bit.ly/k8s113-features)
- [1.12 Milestone](https://bit.ly/k8s112-features)
- [1.11 Milestone](http://bit.ly/k8s-features-111)

Procedure:
*TBA*

## Labels

| Label Name | Purpose | How to use this label | Who should use this label |
| ------ | ------ | ------ | ------ |
| `sig/foo` | Denotes the SIG(s) which owns this enhancement e.g., `SIG Foo` | Set the label using the comment `/sig foo` (on a separate line) | Anyone |
| `kind/feature` | Denotes that the issue should be tracked as an enhancement (All enhancement issues should be marked with this label) | Set the label using the comment `/kind feature` (on a separate line) | Anyone |
| `tracked/yes` | Denotes an issue has been reviewed by a Feature Maintainer (SIG PM / SIG Release) and is actively tracked for the current milestone | Manually set | Feature Maintainers (SIG PM / SIG Release) ONLY |
| `tracked/no` | Denotes an issue has been reviewed by a Feature Maintainer (SIG PM / SIG Release) and will not actively tracked for the current milestone | Manually set | Feature Maintainers (SIG PM / SIG Release) ONLY |
| `stage/{alpha,beta,stable}` | Denotes the stage of an issue in the features process. | Set the label using the comment `/stage alpha` (on a separate line) | Anyone |
