# features
Feature tracking repo for Kubernetes releases

This repo only contains issues.  These issues are umbrellas for new features added to Kubernetes.  

## Why are features tracked

Once users adopt a feature, they expect to use it for a extened period of time.   Therefore, we hold new features them to a
high standard of conceptual integrity, and require consistency with other parts of the system, thorough testing, and complete
documentation.   As the project grows, no single person can track whether all those requirements are met.  Also, a feature's
development lifetime often spans three stages: [Alpha, Beta, and Stable](
https://github.com/kubernetes/kubernetes/blob/master/docs/api.md#api-versioning). 
Feature Tracking Issues provide a checklist that allows for different approvers for different aspects, and ensures that nothing is forgotten across the long development lifetime of a feature.  

## What changes require a Feature Tracking Issue

Features are things which users will notice, and come to rely on.  

Here are some examples things that require a Feature Tracking Issue:
- adding a new type to the core APIs (`pkg/api` or `pkg/apis` in [https://github.com/kubernetes/kubernetes]).
- adding fields to an existing type
- changing the behavior of the Kubernetes scheduler or Kubelet in ways that are easily visible to users
- adding commands and flags to `kubectl`.

Here are some examples things that do not require Feature Tracking Issues:
- features implemented using `ThirdPartyResource` and/or in [https://github.com/kubernetes/contrib]
- fixing a flaky test
- refactoring code
- performance improvments, which are only visible to users as faster API operations, or faster control loops.
- just adding error messages or events

If you are not sure, ask someone in the SIG where you initially circulated the idea.  If they aren't sure, file an issue an
mention @kubernetes/kube-api.

## When to create an issue here

Create an issue here once you:
- Have circulated your idea to see if there is interest
   - through Community Meetings, SIG meetings, SIG mailing lists, or an issue in github.com/kubernetes/kubernetes.
- (optionally) have done a prototype in your own fork.
- Have identified people who agree to work on the feature.
  - many features will take several releases to progress through Alpha, Beta, and Stable stages.
  - you and your team should be prepared to work on the approx. 9mo - 1 year that it takes to progress to Stable status.
- Are ready to be the project-manager for the feature.

## When to comment on a Feature Issue

Please comment on the feature issue to:
- request a review or clarification on the process
- update status of the feature effort
- link to relevant issues in other repos

Please do not comment on the feature issue to:
- discuss a detail of the design, code or docs.  Use a linked-to-issue or PR for that.

## Writing a Design Proposal

- Read some existing [design proposals](https://github.com/kubernetes/kubernetes/tree/master/docs/proposals) and [design docs](https://github.com/kubernetes/kubernetes/tree/master/docs/design).
- Include concrete use cases and describe the "roles" who will use your feature.
- Submitting the design proposal as a PR against `docs/proposals` in https://github.com/kubernetes/kubernetes allows line-by-line discussion of the proposal with the whole community.
 
## Coding

Use as many PRs as you need.  Write tests in the same or different PRs, as is convenient for you.  As each PR is merged, add 
a comment to this issue referencing the PRs.  Code goes in the http://github.com/kubernetes/kubernetes repository, and 
sometimes other repos.  Once the code is complete, you can check off the corresponding box in the checklist. 

## Docs

User docs go into http://github.com/kubernetes/kubernetes.github.io.  Use as many PRs are needed to add these.  Once your 
docs are merged, you can check check off the box on the checklist.



