# KEP-5080: Ordered Namespace Deletion

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Feature Gate handling](#feature-gate-handling)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1 - Pod VS NetworkPolicy](#story-1---pod-vs-networkpolicy)
    - [Story 2 - having finalizer conflicts with deletion order](#story-2---having-finalizer-conflicts-with-deletion-order)
    - [Story 3 - having policy set up with parameter resources](#story-3---having-policy-set-up-with-parameter-resources)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Having ownerReference conflicts with deletion order](#having-ownerreference-conflicts-with-deletion-order)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Dependency cycle](#dependency-cycle)
- [Design Details](#design-details)
  - [DeletionOrderPriority Mechanism](#deletionorderpriority-mechanism)
  - [Handling Cyclic Dependencies](#handling-cyclic-dependencies)
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
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Using finalizers to define the deletion ordering](#using-finalizers-to-define-the-deletion-ordering)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This kep introduces an opinionated deletion process in the Kubernetes namespace deletion 
to ensure secure deletion of resources within a namespace. 
The current deletion process is semi-random, which may lead to security gaps or 
unintended behavior, such as Pods persisting after the deletion of their associated NetworkPolicies. 
By implementing an opinionated deletion mechanism, the Pods will be deleted before other resources with 
respects logical and security dependencies. 
This design enhances the security and reliability of Kubernetes by mitigating risks arising from the non-deterministic deletion order.



## Motivation

The existing random deletion process for resources in a namespace poses significant challenges, 
particularly in environments with strict security requirements. One critical issue is the potential for 
a Pod to remain active after its associated NetworkPolicy has been deleted, leaving it exposed to unrestricted network access. 
This creates a security vulnerability during the cleanup process.

Additionally, the lack of a defined deletion order can lead to operational inconsistencies, 
where any sort of safety guard resources (not just NetworkPolicy) are deleted before their guarded resources (e.g., Pods), 
resulting in unnecessary disruptions or errors.

By introducing an opinionated deletion process, this proposal aims to:

- Enhance Security: Ensure resources like NetworkPolicies remain in effect until all dependent resources have been safely terminated.
- Increase Predictability: Provide a consistent and logical cleanup process for namespace deletion, reducing unintended side effects.

This opinionated deletion approach aligns with Kubernetes' principles of reliability, security, and extensibility, 
providing a solid foundation for managing resource cleanup in complex environments.


### Goals

1. Introduce an Opinionated Deletion Order: Implement a mechanism while namespace deletion to prioritize the deletion of certain resource types before others based on logical dependencies and security considerations (e.g., Pods deleted before NetworkPolicies).2

2. Maintain Predictability and Consistency: Provide a more deterministic deletion process to improve user confidence and debugging during namespace cleanup.

3. Integrate with Existing Kubernetes Concepts: Build on the namespace deletion’s current architecture without introducing breaking changes to existing APIs or workflows.

4. Be safe - don’t introduce unresolvable deadlocks.

5. Make the most common dependency - workloads and the policies that govern them - safe by default for all types of policies, including CRDs, unless specifically opted out.


### Non-Goals

1. Reordering Deletion Across Namespaces: This design focuses on resource deletion within a single namespace. It does not attempt to enforce or prioritize deletion order across multiple namespaces.

2. Introducing Custom Per-Resource Deletion Order: While the proposal aims for opinionated ordering, it does not cover fine-grained customization by end-users for specific resources or workloads.

3. Guaranteeing Real-Time Enforcement: The proposal does not aim to guarantee real-time deletion of resources; the Kubernetes control plane’s reconciliation loop remains the underlying driver.

4. Replacing Finalizers or Current Garbage Collection Mechanisms: The design does not intend to replace or bypass the existing Finalizer mechanism but works alongside it to enhance the resource cleanup process.

5. Handling Non-Standard or External Resources: This design does not apply to external or non-Kubernetes resources managed outside the cluster (e.g., external databases, cloud resources).

6. Global Enforcement of Security Policies: While the proposed changes improve security during deletion, it is not a substitute for broader, cluster-wide security policies or mechanisms.

7. Implement a full graph: This proposal does not aim to model the dependency relationship between specific objects (instances) or between types as an arbitrary graph.


## Proposal

When the feature gate `OrderedNamespaceDeletion` is enabled,
the resources associated with this namespace should be deleted in order:

- Delete all pods in the namespace (in an undefined order).
- Wait for all the pods to be stopped or deleted.
- Delete all the other resources in the namespace (in an undefined order).

### Feature Gate handling

Due to this KEP is addressing the security concern and we do wanna give options to close security gaps in the past,
the feature gate will be introduced as beta and on by default in 1.33 release. We will backport the feature gate with off-by-default
configuration to all supported releases. See [the detailed discussion on slack](https://kubernetes.slack.com/archives/CJH2GBF7Y/p1741258168683299)

### User Stories (Optional)

#### Story 1 - Pod VS NetworkPolicy

A user has pods which listen on the network and network policies which help protect those pods. 
While namespace deletion, there could be cases that NetworkPolicy has deleted while the pods are running 
which cause the security concern of having Pods running unprotected.

After this feature was introduced, we would have NetworkPolicy always deleted after the Pods to 
avoid the above security concern.

#### Story 2 - having finalizer conflicts with deletion order

E.g. if the pod has a finalizer which is waiting for network policies (which is opaque to Kubernetes), 
it will cause dependency loops and block the deletion process.

Refer to the section `Handling Cyclic Dependencies`.

#### Story 3 - having policy set up with parameter resources

When ValidatingAdmissionPolicy is used in the cluster with parameterization, it is possible to use pod as the parameter resources. In this case, the parameter resources will be deleted before VAP and 
lead the VAP not in use. To make it even worse, if the ValidatingAdmissionPolicyBinding is configured with `.spec.paramRef.parameterNotFoundAction: Deny`, 
it could block certain resources operations and also hang the termination process. Similar concern applies to Webhooks with parameter resources.

It is an existing issue with current namespace deletion as well. As long as we don't plan to have a dependency graph built, it will rely more on 
best practices and user's configuration.

### Notes/Constraints/Caveats (Optional)

#### Having ownerReference conflicts with deletion order

When deciding the deletion priority for resources, it should take ownerReference into consideration.
E.g. the deployment VS pod. However, it should not matter much in terms of namespace deletion.
Namespace deletion specifically uses `metav1.DeletePropagationBackground` and all resources would be deleted and the ownerReference
dependencies would be handled by the garbage collection.

In Kubernetes, `ownerReferences` define a parent-child relationship where child resources are automatically deleted when the parent is removed.
This is mostly handled by garbage collection. While namespace deletion, the `ownerReferences` is not part of the consideration and the garbage collector controller will make sure  
no child resources still existing after the parent resource deleted.


### Risks and Mitigations

#### Dependency cycle

The introduction of deletion order could potentially cause dependency loops especially when finalizers are 
specified against deletion priority.

When a lack of progress detected(maybe caused by the dependency cycle described above), it could hang the deletion process 
same as the current behavior.

Mitigation: Delete the blocking finalizer to proceed.

## Design Details

### DeletionOrderPriority Mechanism

For the namespace deletion process, we would like to have the resources associated with this namespace be deleted as following:

- Delete all pods in the namespace (in an undefined order).
- Wait for all the pods to be stopped or deleted.
- Delete all the other resources in the namespace (in an undefined order).

The above order will be strict enforced as long as the feature gate is turned on.

### Handling Cyclic Dependencies

Cyclic dependencies can occur if resources within the namespace have finalizers set which conflicts with the DeletionOrderPriority. 
For example, consider the following scenario:

- Pod A has a finalizer that depends on the deletion of Resource B.

- Pod A suppose to be deleted before Resource B.

In this case, the finalizers set would conflict with the NamespaceDeletionOrder and could lead to cyclic dependencies and cause namespace deletion process hanging.

To mitigate the issue, user would have to manually resolve the dependency lock by either remove the finalizer or force delete the blocking resources which would be the same as current mechanism.


### Test Plan

[ X ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.



##### Prerequisite testing updates

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

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

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

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->
#### Beta

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled
- Complete features specified in the KEP
- Proper metrics added
- Additional tests are in Testgrid and linked in KEP

#### GA

- Related [CVE](https://github.com/kubernetes/kubernetes/issues/126587) has been mitigated  
- Conformance tests

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md


### Upgrade / Downgrade Strategy

In alpha, no changes are required to maintain previous behavior. And the feature gate
could be turned on to make use of the enhancement.

### Version Skew Strategy

Not applicable

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback


###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: NamespaceDeletionOrder
  - Components depending on the feature gate:
	  - kube-apiserver

###### Does enabling the feature change any default behavior?

No, default behavior is the same.


###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, through the feature gate.

###### What happens if we reenable the feature if it was previously rolled back?
The namespace deletion will respect the order specified again.

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
Yes. Unit test and integration test will be introduced in alpha implementation.

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
This feature should not impact rollout.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->
N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->
N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
No.

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
Check if the feature gate is enabled. The feature is a security fix which should not be user detectable.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

N/A

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
The feature only affect namespace deletion and should not affect existing SLOs.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->
The error or blocker will be updated to namespace status subresource to follow the existing pattern.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
Namespace status will be used to capture the possible error or blockers while deletion.

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
No.
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
No.
###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
No.
###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
No.
###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->
No.
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
The namespace controller will act exactly the same with/without this feature.
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
Namespace deletion might hang if pod resources deletion running into issues with the feature gate enabled.
###### What steps should be taken if SLOs are not being met to determine the problem?
Delete the blocking resources manually.
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

### Using finalizers to define the deletion ordering

Finalizers could potentially solve this problem or work as a workaround for this issue. 
Having a controller running and watching the NetworkPolicy and adding a finalizer to make sure Pods 
always deleted before NetworkPolicy is the alternative solution. However, it is not the best way to go because:
- User would always have customized controller introduced and it is hard to educate everyone to follow the best practices
- It could not address the previous behavior completely
- It is not generic enough in case of later there is request coming for other resources deletion ordering


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

