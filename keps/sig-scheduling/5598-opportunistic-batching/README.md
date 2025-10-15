<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-5598: Opportunistic batching
<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

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
  - [Pod scheduling signature](#pod-scheduling-signature)
  - [Batching mechanism](#batching-mechanism)
    - [Create](#create)
    - [Update](#update)
    - [Nominate](#nominate)
  - [Opportunistic batching](#opportunistic-batching)
  - [Comparison with Equivalence Cache (circa 2018)](#comparison-with-equivalence-cache-circa-2018)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Plugins need to keep signatures up to date](#plugins-need-to-keep-signatures-up-to-date)
    - [We are narrowing the feature set where batching will work](#we-are-narrowing-the-feature-set-where-batching-will-work)
    - [We don't have experience with batching in production](#we-dont-have-experience-with-batching-in-production)
- [Design Details](#design-details)
  - [Pod signature v1](#pod-signature-v1)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
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

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes an opportunistic batching mechanism in the scheduler to improve performance of scheduling many compatible pods at once, and to begin building the infrastructure required for gang scheduling. To implement this mechanism we propose the following additions:

 - **Pod scheduling signature:** A signature that captures the properties of a pod that impact scoring and feasibility.
 - **Batching mechanism:** A mechanism to reuse the scheduling output from one pod to set the nominated node name for multiple subsequent pods with matching scheduling signatures.
 - **Opportunistic batching:** Transparent inclusion of the batching mechanism in the scheduler to improve the performance of targeted workloads that could benefit from it.

## Motivation

Today our scheduling algorithm is O(num pods x num nodes). As the size of clusters and jobs continue to increase, this leads to low performance when scheduling or rescheduling large jobs. This increases user cost and slows down user jobs, both unpleasant impacts. Optimizations like this one have the potential to dramaticly reduce the cost of scheduling in these scenarios.

We are also working on gang scheduilng (in addition to other forms of multi-pod scheduling), which will give us a way to consider multiple pods at the same time. "Opportunistic batching" provides a starting point for these mechanisms by providing signatures and batching, both necessary foundational mechanisms, and including them initially in a simple way.

Another change is the shift towards 1-pod-per-node in batch and ML environments. Many of these environments (among others) only attempt to run a single user pod on each node, along with a complement of daemon set pods. This simplifies our scheduling needs significantly, as it allows to reuse not only filtering, but also scoring results.

### Goals

 * Improve the performance of scheduling large jobs on large clusters where the constraints are simple.
 * Begin building infrastructure to support gang scheduling and other "multi-pod" scheduling mechanisms.
 * Ensure that the infrastructure we build is maintainable as we update, add and remove plugins.
 * Provide improved performance with identical results to our current scheduler for targeted workloads in this release.
 * Provide a path where we can expand batching to apply to most or all workloads over the next few releases.

### Non-Goals

 * We are not attempting to apply this optimization to all pods in this release. We will make the addition of batching transparent, but only applicable to a reduced set of workloads in v1.
 * We are not adding gang scheduling of any kind in this KEP. This is purely a performance improvement without adding dependency on the Workload API [KEP-4671](https://github.com/kubernetes/enhancements/pull/5558), although we hope the work on this KEP will help us with gang scheduling as we build it.

## Proposal

See https://github.com/bwsalmon/kubernetes/pull/1 for a WIP version of the code (currently out of date). We discuss each of the added items: pod scheduling signature, batching mechanism and opportunistic batching in turn.

### Pod scheduling signature

The pod scheduling signature is used to determine if two pods are "the same" from a scheduling perspective. In specific, what this means is that any pod with the given signature will get the same scores / feasibility results from any arbitrary set of nodes. This is necessary for the cache to work, since we need to be able to reuse the previous work.

Note that some pods will not have a signature, because the scoring uses not just pod and node attributes, but other pods in the system, global data about pod placement, etc. These nodes get a nil signature, and we fallback to the slow path.

To allow non in-tree plugins to construct a signature, we add a new framework function to implement. This function takes a pod and generates a signature for that plugin as a string. The signature is likely a set of attributes of the pod, or something derived from them. To construct a full signature we take the signatures of all the plugins and aggeregate them into a single string. If any plugin cannot generate a signature for a given pod (because it depends on information other than the pod and node), then we generate a "nil" signature and don't attempt to batch the pod.

Initially we won't require plugins to implement the new function, but we will turn off signatures for all pods if a plugin is enabled that does not implement it. In subsequent releases we might make implementation of the function a requirement, but of course plugins are also able to say pods are unsignable. 

### Batching mechanism

The second component of this KEP is a batching mechanism. Fundamentally the batching mechanism will have two operations that can be invoked wherever they are needed:

 * **Create:** Create a new set of batch information from the scheduling results of a "canonical" pod that has a valid signature. This effectively tracks the scores of a set of nodes sorted in score order.
 * **Nominate:** Using batching information from create, set the nominated node name of a new pod whose signature matches the canonical pod's signature.

Internally the mechanism will use an **update** operation which we will also describe.

#### Create 

The create operation will use the sorted output from the scheduling of a "canonical" pod. After copying the feasible node list from the results, it will attempt to update the results using the update operation, which we describe next. If the results can't be updated, we will just drop the information without reusing it. If we can update the results, we will keep the batch information ready for use to nominate node names for subsequent pods.

#### Update

The update operation will attempt to update the batch information after a scheduling or nomination has taken place. In service of this updating we will introduce an optional plugin interface for "Rescoring". The rescoring interface will take the pod and the scoring information for the node we bound the pod to. The update operation will call the rescoring interface on all plugins. Each plugin can return one of three results:

 * **Infeasible:** If the plugin can determine another pod of this kind cannot be placed on the node, then it returns infeasible. If *any* plugin returns infeasible for the node, we will simply drop this node from the results, and save the rest for our next round.
 * **Updated:** If the plugin can update the score it will do so in the node object and return this response. If *all* plugins return updated for the node, then we can keep the node in our results and just reorder it. We will not implement this in v1, but will leave it possible for v2.
 * **Unknown:** If the plugin does not know how to update the score / feasibility of the node it can return unknown. If a scoring / filtering plugin doesn't implement the interface we assume unknown for all calls. If *no* plugin returns infeasible, and *any* plugin returns unknown, we will drop all of the scheduling results and not attempt to reuse them.

In v1 we will implement "infeasible" rescoring functions for key plugins that we know can tell (nodeports, noderesources, etc). This will effectively limit the use of batching to specific workloads we can identify as "1-pod-per-node" but will open the path for us to continually expand the cases where we can apply batching by enhancing and adding rescoring functions.

#### Nominate

The nominate operation will take a pod with a matching signature and assign its nominated node name, using the first node in the list.  Nomination will also call the update operation to update the results for use on more pods in the future. Note that nomination doesn't actually schedule the pod, but it ensures that when the pod is scheduled it will take the fast path and not re-evaluate the full set of nodes. By separately these decisions we can use the batching mechanism in multiple places, including gang scheduling.

### Opportunistic batching

We will then apply the batching mechanism to simple cases in the current code. For example, we might simply attempt to use the batching mechanism on a pod if the previous pod had the same signature, and we have had no interruptions in our stream of pods. We will apply the batching mechanism in a limited and simple fashion to give us experience in production and provide focused value without requiring us to do significant effort before integrating gang scheduling.

### Comparison with Equivalence Cache (circa 2018)

This KEP is addressing a very similar problem to the Equivalence Cache (eCache), an approach suggested in 2018 and then retracted because it became extremely complex. While this KEP addresses a similar problem it does so in a very different way, which we believe avoids the issues experienced by the eCache

The issues experienced by eCache were:

 * eCache performance was still O(num nodes).
 * eCache was complex
 * eCache was tightly coupled with plugins.

 We'll address each in turn, but at a high level the differences stem from our scope reduction in this cache, where
 we focus on simple constraints in a 1-pod-per-node world, and are comfortable extending our "race" period slightly.

 #### eCache performance was still O(num nodes)

 The eCache was caching a fundamentally different result than this cache. In the case of the eCache they were caching
 the results of a predicate p, (which is sounds like was one of a number of ops for a given plugin) for a specific pod and node.
 This meant the number of cache lookups per pod was O(num nodes * num predicates) where num predicates was O(num plugins). Because 
 the cache was so fine-grained, the cache lookups were, in many cases, more expensive than the actual computation. This also meant
 that while the cache could improve performance, it fundamentally did not remove the O(num nodes) nature of the per pod computation.

 In the case of this cache, we are looking up the entire host filtering and scoring for a single pod, so the number of cache lookups
 per pod is 1. We are caching the entire filtering / scoring result, so the map lookup is guaranteed to be faster even
 than just iterating over the plugins themselves, let alone the computation needed to filter / score. As the number of nodes go up,
 the fact that the cache lookup is O(1) per pod will make it an increasingly perfromant alternative to the full computation.

 We can cache this more granular data because we only cache for simple plugins, and in fact avoid the complex plugins entirely.
 Thus we do not need to be concerned about cross pod dependencies, meaning we do not need to try to keep detailed information 
 up-to-date. Because we assume 1-pod-per-node and some amount of "staleness" we simply need to invalidate whole hosts, rather 
 than requiring upkeep of complex predicate results required to keep the eCache functional.

 #### eCache was complex

 Because the eCache cached predicates, the logic for computing these results went into the cache as well. This meant that significant 
 amount of the plugin functionality was replicated in the cache layer. This added significant complexity to the cache, and also made 
 keeping the cache results themselves up to date complex, involving multiple pods, etc. Because the eCache only improved performance 
 for complex queries, it needed to include this complexity to provide value.

 In contrast, the signature used in this cache is just a subset of the pod object, without complex logic. It is static and as the pod object changes slowly, it will change slowly as well. In addition, we explicitly avoid all the complex plugins in this cache because they are rarely used. Thus we do not have the same complexity needed in the cache.
 
 #### eCache was tightly coupled with plugins
 
 Because a significant amount of the plugin complexity made into the eCache, it was difficult for plugin owners to keep the things in sync. Since in this cache the signature is just parts of the pod object, and the pod object is fairly stable, this makes keeping the signature up to date a much simpler task. The creation of the signature is also spread across the plugins themselves, so instead of needing to keep the cache up to date, plugin owners simply have a new function they need to manage within their plugin, which the cache only aggregates.

 We will also provide tests that evaluate different pod configurations against different node configurations and ensure that any time the signatures match the results do as well. This will help us catch issues in the future, in addition to providing testing opportunities in other areas.

See https://github.com/kubernetes/kubernetes/pull/65714#issuecomment-410016382 as starting point on eCache.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations


#### Plugins need to keep signatures up to date

The cache will only work if plugin maintainers are able to keep their portion of the signature up-to-date. We believe this should be doable because the logic is put into the plugin interface itself, and we are restricting it to portions of the pod spec, but there is still risk of subtle dependencies creeping in.

If plugin changes prove to be an issue, we could codify the signature as a new "Scheduling" object that only has a subset
of the fields of the pod. Plugins that "opt-in" could only be given access to this reduced scheduling object, and we could then use the entire scheduling object as the signature. This would make it more or less impossible for the signature and plugins to be out of sync, and would naturally surface new dependencies as additions to the scheduling object. However, as we expect plugin changes to be relatively modest, we don't believe the complexity of making the interface changes is worth the risk today.

#### We are narrowing the feature set where batching will work

Because we are explicitly limiting the functionality that this cache will support, we run the risk of designing something that will not work for enough users for it to be useful. To mitigate this risk we are actively engaging with users and doing analysis of data available on K8s users to ensure we are still capturing a large enough number of user use cases. We also will address this by expanding over time; we expect to have a few interested parties up front, but will then evaluate expansions that could onboard more.

#### We don't have experience with batching in production

Because we haven't deployed batching in production before, we are still somewhat limited in the information we have about user workloads. To mitigate this concern we will build in tracing / analytics to help us understand how frequently we see specific patterns, how often we are able to batch, and the most common reasons we are unable to batch. When possible we will collect this information even when the feature itself is disabled, to allow us to approach our next iterations with more data.

## Design Details

### Pod signature v1

The follow section outlines the attributes we are currently proposing to use as the signature for each of the 
plugins in the scheduler. We need the plugin owners to validate that these signatures are correct, or help
us find the correct signature.

Note that the signature does not need to be stable across versions, or even invocations of the scheduler. 
It only needs to be comparable between pods on a given running scheduler instance.

 * DynamicResources: For now we mark a pod unsignable if it has dynamic resource claims. We should improve this in the future, since most DRA claims are node specific and we should be able to determine this with a little effort. We will attempt to pull forward at least some integration of simple DRA claims with batching into this version as well.
 * ImageLocality: We use the canonicalized image names from the Volumes as the signature.
 * InterPodAffinity: If either the PodAffinity or PodAntiAffinity fields are set, the pod is marked unsignable, otherwise an empty signature.
 * NodeAffinity: We use the NodeAffinity and NodeSelector fields, plus any defaults set in configuration as the signature.
 * NodeName: We use the NodeName field as the signature.
 * NodePorts: We use the results from util.GetHostPorts(pod) as the signature.
 * NodeResourcesBalancedAllocation: We use the output of calculatePodResourceRequestList as the signature.
 * NodeResourcesFit: We use the output of the computePodResourceRequest function as the signature.
 * NodeUnschedulable: We use the Tolerations field as the signature.
 * NodeVolumeLimits: We use all Volume information except from Volumes of type ConfigMap or Secret.
 * PodTopologySpread: If the PodTopologySpead field is set, or it is not set but a default set of rules are applied, we mark the pod unsignable, otherwise it returns an empty signature. Because the plugin itself is creating the signature, it knows whether and what kind of default it will apply.
 * TaintToleration: We use the Tolerations field as the signature.
 * VolumeBinding: Same as NodeVolumeLimits.
 * VolumeRestrictions: Same as NodeVolumeLimits.
 * VolumeZone: Same as NodeVolumeLimits.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

###### Coverage of existing packages

Will add an extra function and test for plugins we touch.

- `k8s.io/kubernetes/pkg/scheduler`: `2025-10-7` - `86.1`
- `k8s.io/kubernetes/pkg/scheduler/framework`: `2025-10-7` - `51.8`
- `k8s.io/kubernetes/pkg/scheduler/framework/runtime`: `2025-10-7` - `84.3`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources`: `2025-10-7` - `80.5`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/imagelocality`: `2025-10-7` - `86.2`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/interpodaffinity`: `2025-10-7` - `89.7`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeaffinity`: `2025-10-7` - `85.8`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodename`: `2025-10-7` - `50`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeports`: `2025-10-7` - `83.7`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/noderesources`: `2025-10-7` - `89.5`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeunschedulable`: `2025-10-7` - `87.1`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodevolumelimits`: `2025-10-7` - `73.7`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/podtopologyspread`: `2025-10-7` - `87.8`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/tainttoleration`: `2025-10-7` - `86.9`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/volumebinding`: `2025-10-7` - `83.9`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/volumerestrictions`: `2025-10-7` - `74`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/volumezone`: `2025-10-7` - `84.8`

###### New unit tests

The code draft has first versions of most of these, will add more as we get through the discussion process.

- schedule_one_test.go - Add test cases for opportunistic batching.
- signature_test.go - Test cases for the framework signature call and the helper class
- signature_consistency_test.go - Test cases to ensure the signature captures all the necessary information. We will take a range of pod specs and node definitions, run them through the filtering / scoring code, then ensure that the pods with matching signatures always get equivalent results.
- batching_test.go - Test cases for the batching mechanism, separate from the actual integration into the scheduling pipeline.
- rescore_test.go - Test cases for rescoring plugins and calls.

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

Will add a few integration tests:
 - Perf tests: Add a few tests into scheduler_perf that look at performance with the cache enabled and disabled for a few target scenarios.
 - End-to-end consistency: Add tests that run a set of pods through the scheduler end-to-end with the cache enable and disabled. Ensure the scheduling decisions are the same with the cache on or off. Hopefully use same pod spec and node definitions from the signature_consistency_test.

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

<!--
- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

Users should continue to see the same behavior, just with better performance. If the feature has bugs, they can use the feature gates to disable it.

Users should be able to take advantage of batching without any change to their behavior, other than ensuring the feature gate is enabled. Batching will not speed up all workloads, but workloads it can improve will be improved transparently.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

This feature should be localized to the scheduler. So long as the scheduler is correctly built, we should not require other interactions from components in the system. Scheduler plugins will need to implement new methods to take advantage of the feature, but if they do nothing the feature will simply end up disabled.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [ X ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `SchedulerOpportunisticBatching`
  - Components depending on the feature gate: `kube-scheduler`

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No, it should not. Pods will only use the cache if they are targeted to a scheduler config with the feature enabled.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, it can be disabled. Because it only keeps in-memory state, setting the flag to false and restarting the scheduler should clear any previous state.

###### What happens if we reenable the feature if it was previously rolled back?

This feature only maintains in-memory, in-flight state, so changing the feature gate, which restarts the scheduler, should not cause issues with a running system.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

Not needed, as described above.

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

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
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

No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

No, other than potentially adding a single configuration field to the scheduler config object.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No, it should result in decreased time for scheduling operations.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

Enabling the cache will add RAM usage to store the cache. This is should be small (O(Mb)) but will be there.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

It could lead to RAM exhaustion if eviction mechanisms were to fail or the cache were configured to be too large.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

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

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
