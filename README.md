# Feature Tracking and Backlog

Feature tracking repo for Kubernetes releases

This repo only contains issues. These issues are umbrellas for new features to be added to Kubernetes. A feature usually takes multiple releases to complete. And a feature can be tracked as backlog items before work begins. A feature may be filed once there is consensus in at least one [Kubernetes SIG](https://github.com/kubernetes/kubernetes/wiki/Special-Interest-Groups-(SIGs)).

## Is My Thing a Feature?

We are trying to figure out the exact shape of a feature. Until then here are a few rough heuristics.

A feature is anything that:

- A blog post would be written about after its release (ex. [minikube](http://blog.kubernetes.io/2016/07/minikube-easily-run-kubernetes-locally.html), [PetSet](http://blog.kubernetes.io/2016/07/thousand-instances-of-cassandra-using-kubernetes-pet-set.html), [rkt container runtime](http://blog.kubernetes.io/2016/07/rktnetes-brings-rkt-container-engine-to-Kubernetes.html)).
- Requires multiple parties/SIGs/owners participating to complete (ex. GPU scheduling [API, Core, & Node], PetSet [Storage & API]).
- Needs significant effort or changes Kubernetes in a significant way (ex. something that would take 10 person-weeks to implement, introduce or redesign a system component, or introduces API changes).
- Impacts the UX or operation of Kubernetes substatially such that engineers using Kubernetes will need retraining.
- A users will notice and come to rely on.

It is unlikely a feature if it is:

- implemented using `ThirdPartyResource` and/or in [https://github.com/kubernetes/contrib]
- fixing a flaky test
- refactoring code
- performance improvements, which are only visible to users as faster API operations, or faster control loops.
- adding error messages or events

If you are not sure, ask someone in the SIG where you initially circulated the idea.  If they aren't sure, file an issue an

## When to Create a New Feature Issue

Create an issue here once you:
- Have circulated your idea to see if there is interest
   - through Community Meetings, SIG meetings, SIG mailing lists, or an issue in github.com/kubernetes/kubernetes.
- (optionally) have done a prototype in your own fork.
- Have identified people who agree to work on the feature.
  - many features will take several releases to progress through Alpha, Beta, and Stable stages.
  - you and your team should be prepared to work on the approx. 9mo - 1 year that it takes to progress to Stable status.
- Are ready to be the project-manager for the feature.

## Why are features tracked

Once users adopt a feature, they expect to use it for a extended period of time.   Therefore, we hold new features them to a
high standard of conceptual integrity, and require consistency with other parts of the system, thorough testing, and complete
documentation.   As the project grows, no single person can track whether all those requirements are met.  Also, a feature's
development lifetime often spans three stages: [Alpha, Beta, and Stable](
https://github.com/kubernetes/kubernetes/blob/master/docs/api.md#api-versioning). 
Feature Tracking Issues provide a checklist that allows for different approvers for different aspects, and ensures that nothing is forgotten across the long development lifetime of a feature.  

mention @kubernetes/kube-api.

## When to comment on a Feature Issue

Please comment on the feature issue to:
- request a review or clarification on the process
- update status of the feature effort
- link to relevant issues in other repos

Please do not comment on the feature issue to:
- discuss a detail of the design, code or docs.  Use a linked-to-issue or PR for that.
