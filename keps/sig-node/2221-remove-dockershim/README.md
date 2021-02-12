# KEP-2221: Removing dockershim from kubelet

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Terms](#terms)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Pros](#pros)
  - [Cons](#cons)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Dockershim removal criteria](#dockershim-removal-criteria)
  - [Dockershim removal plan](#dockershim-removal-plan)
  - [Risks and Mitigations](#risks-and-mitigations)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Terms

- **CRI:** Container Runtime Interface – a plugin interface which enables kubelet to use a wide variety of container
runtimes, without the need to recompile.

## Summary

CRI for docker (i.e. dockershim) is currently a built-in container runtime in kubelet code base. This proposal aims 
at a deprecation and subsequent removal of dockershim from kubelet.

## Motivation

In Kubernetes, the CRI interface is used to talk to a container runtime, The design of CRI is to be able to run a CRI
implementation as a separate binary. However currently the CRI of docker (a.k.a. dockershim) is part of kubelet code, runs
as part of kubelet and is tightly coupled with kubelet's lifecycle. 

This is not ideal as kubelet then has dependency on specific container runtime which leads to maintenance burden for not 
only developers in sig-node, but also cluster administrators when critical issues (e.g. runc CVE) happen to container 
runtimes. The pros of removing dockershim is straightforward:

### Pros

- Docker is not special and should be just a CRI implementation just like every other CRI implementation in our ecosystem.
- Currently, dockershim "enjoys" some inconsistent integrations for various reasons (see [legacyLogProvider](https://cs.k8s.io/?q=legacyLogProvider&i=nope&files=&repos=kubernetes/kubernetes) for example) . Removing these "features" should eliminate maintenance burden of kubelet.
- A cri-dockerd can be maintained independently by folks who are interested in keeping this functionality
- Over time we can remove vendored docker dependencies in kubelet.
- Due to convenience of inheriting from this builtin shim for the container runtime, there is less incentive to move to 
  new container runtimes. The production issues found and addressed in both Containerd and CRI-O might cause the 
  production issues for some users, which causes a lot of maintenance burdens to Kubernetes community.
- The community can focus and move faster on the new container runtime-related enhancements once we drop dockershim

Having said that, cons of removal built-in dockershim requires lots of attention:

### Cons

- Deployment pain with a new binary in addition to kubelet.
  - An additional component may aggravate the complexity currently. It may be relieved with docker version evolutions.
- The number of affected users may be large.
  - Users must change existing use experience when using Kubernetes and docker.
  - Users have to change their existing workflows to adapt to this new changes.
  - And other unrecorded stuff.
- CRI is still technically in alpha, and should graduate to GA before removing dockershim from kubelet. There 
is a [KEP 2041](https://github.com/kubernetes/enhancements/pull/2041) for graduating the CRI API version.
- cri-dockerd will vendor kubernetes/kubernetes, that may be tough.
- cri-dockerd as an independent software running on node should be allocated enough resource to guarantee its availability.

> You can check [the discussion in sig-node mailing list](https://groups.google.com/forum/#!msg/kubernetes-sig-node/0qVzfugYhro/l6Au216XAgAJ) for more details. 

### Goals

- A concrete dockershim removal criteria.
- A brief plan to remove dockershim spanning multiple releases.

### Non-Goals

- Refactoring or re-design of dockershim itself due to deprecation.

## Proposal

### Dockershim removal criteria

- CRI itself is alpha. So we need another KEP to graduate CRI API.
- kubelet has no dependency on dockershim/docker in its whole lifecycle. This is already done using the `dockerless` tag
- All node related features are CRI generic and have no "back door" dependency on dockershim/docker.
- Deprecate and remove, or replace all Docker-specific features.
- Reasonable benchmark result of performance degradation after moving dockershim to out-of-tree.
- E2E test framework has been updated with fully support of out-of-tree CRI container runtime.

### Dockershim removal plan

Step 1: Deprecate in-tree dockershim and decouple dockershim from kubelet.

Target releases: 1.20

Actions:

- Mark in-tree dockershim as "maintenance mode":
  - CRI generic changes/features can continue on dockershim.
  - WIP efforts on dockershim can continue and go to complete.
  - dockershim/docker specific changes/features should be rejected.
- Deprecate the legacy features of dockershim in kubelet by providing a specific timeline. Currently, kubelet still has:
  - vendored dockershim 
  - flags that are used to configure dockershim.
  - support to get container logs when docker uses journald as the driver.
  - logic of moving docker processes to a given cgroup
- Ensure e2e/Node e2e test framework is CRI generic and test cases are independent of container runtime.
- Refactoring e2e/Node e2e test framework to include CRI for docker installation (or use other CRI container runtime).
  - Ensure cluster/node e2e are 100% CRI focused.
  - Ensure test-infra install appropriate CRI implementations in e2e machines. 
- Ensure Windows scenarios that currently depend on docker are fully supported in alternative CRI implementations

Step 2: Release kubelet without dockershim

Target releases: 1.24 (assuming 3 release a year or after April 2022)

Actions:

- Document and announce migration guide.
- Release harness would build kubelet with `dockerless` tag on. So the default build will not support docker out of
the box.
- If folks need this support, they would have to build kubelet by themselves as the code is still present in the
source tree.

Step 3: Completely remove in-tree dockershim from kubelet

Deprecation should be for at least a year. Deprecation was announced in December 2020
so dockershim might be deleted the same release it is not built.

Target releases: same as Step 2.

Actions:

- Delete in-tree dockershim code from kubelet after certain "grace period".

### Risks and Mitigations

The easier we make it for folks to switch to CRI implementations the lesser the risk. Another option would be for
folks for a brand new CRI implementation that targets docker. Though even this option means that folks will have to
run an extra process outside of kubelet. The worst case scenario is for us to carry on the dockershim for a couple
of more releases.

### Test Plan

Node e2e testing will be augmented to test kubelet built with `dockerless` tag.

### Graduation Criteria

- All feedback gathered from users
- Adequate test signal quality for node e2e
- Tests are in Testgrid and linked in KEP
- Allowing time for additional user feedback and bug reports
- Kubelet switched to use CRI API v1

### Upgrade / Downgrade Strategy

Upgrade: Users should follow the [migration guide](https://kubernetes.io/docs/tasks/administer-cluster/migrating-from-dockershim/)
before upgrading to a version of the kubelet that no longer includes dockershim.

Downgrade: Not applicable.

### Version Skew Strategy

Not applicable.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
Not applicable for this feature.

* **Does enabling the feature change any default behavior?**
There are slight differences in behavior. Differences in behavior are [listed here](https://kubernetes.io/docs/tasks/administer-cluster/migrating-from-dockershim/check-if-dockershim-deprecation-affects-you/).

* **Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?**
No.

* **What happens if we reenable the feature if it was previously rolled back?**
Not applicable. Roll back is not supported.

* **Are there any tests for feature enablement/disablement?**
Not applicable. Enablement/disablement are not supported.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**
TBD

* **What specific metrics should inform a rollback?**
None.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
I do not believe this is applicable.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**
Even if applying deprecation policies, they may still surprise some users.
No.

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**
Not applicable (no feature gate).

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
This does not seem relevant to this feature.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
This does not seem relevant to this feature.

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
No.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
No

### Scalability

* **Will enabling / using this feature result in any new API calls?**
No.

* **Will enabling / using this feature result in introducing new API types?**
No

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
No

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
No

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
No

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
No

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**
No impact.

* **What are other known failure modes?**
Not applicable.

* **What steps should be taken if SLOs are not being met to determine the problem?**
Not applicable

## Implementation History

- 12/02/2020 (v1.20): [Dockershim Deprecation FAQ](https://kubernetes.io/blog/2020/12/02/dockershim-faq/) published.
- 12/08/2020 (v1.20): dockershim deprecation [warning added to kubelet](https://kubernetes.io/blog/2020/12/08/kubernetes-1-20-release-announcement/#dockershim-deprecation).

## Drawbacks

None.

This eliminates unnecessary vendoring of code from docker/docker github repository and many others dragged in
transitively.

## Alternatives

None.

## Infrastructure Needed (Optional)

None.
