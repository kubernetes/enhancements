<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

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
# KEP-5901: Kubectl Checkpoint

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
  - [User Stories](#user-stories)
    - [Forensic Container Checkpointing](#forensic-container-checkpointing)
    - [Fast Container Startup - Warm Start](#fast-container-startup---warm-start)
    - [Optimize Resource Utilization](#optimize-resource-utilization)
    - [Container Migration](#container-migration)
      - [Fault Tolerance](#fault-tolerance)
      - [Load Balancing](#load-balancing)
      - [Spot Instances](#spot-instances)
      - [Scheduler Integration](#scheduler-integration)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

Introduce the `kubectl checkpoint` sub-command to allow users to checkpoint a
container. The *kubelet* already provides the possibility to checkpoint a
container and this *kubelet* only API endpoint is extended to *kubectl* for
easier user consumption.

## Motivation

The introduction of KEP-2008 (Forensic Container Checkpointing) in Kubernetes
1.25 was a *kubelet* only change initially as checkpointing containers is a
completely new concept in the context of Kubernetes. From the beginning it
was slightly confusing for users that it was only a *kubelet* API endpoint.
Now with KEP-2008 (Forensic Container Checkpointing) graduating from Alpha to
Beta in Kubernetes 1.30, which means that the corresponding feature gate
defaults to the feature being enabled, the next step would be to extend the
existing checkpointing functionality from the *kubelet* to *kubectl* for easier
user consumption. The main motivation is to make it easier by not requiring
direct *kubelet* API endpoint access for the users.

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

Extend the currently only as *kubelet* API endpoint available checkpoint
functionality to *kubectl*.

There is a proof of concept pull request which already implements `kubectl alpha
checkpoint`: [Add 'checkpoint' command to kubectl][pr120898]

The checkpoint archive will stay on the node it was created on. The user will
get the path to the checkpoint archive as well as the name of the node it was
created on. As the checkpoint archive will only be accessible with root
permissions on the compute node in the first step no additional attack surface
should exist as it is possible to get all information stored in the
checkpoint archive with root permissions on the compute node.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

The transfer of the checkpoint archive is not part of this KEP, yet. The
checkpoint archive will stay on the node it was created on. Using a
distributed file-system can be used to share the checkpoint archive between
cluster members.

This is another small step towards better integration of the checkpoint
and restore functionality into Kubernetes. Scheduler integration and container
migration are out of scope for this KEP.

This is also just about container checkpointing and not about pod checkpointing.
There have been proof of concepts to implement pod checkpointing and it is
not a technical challenge but it requires further discussion on how to integrate
it into Kubernetes.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

This KEP proposes to extend the *kubelet* checkpoint API endpoint to the
API server and to implement the `kubectl checkpoint` sub-command.

The existing PR ([Add 'checkpoint' command to kubectl][pr120898]) implements it
as `kubectl alpha checkpoint` as it would be an Alpha feature.

### User Stories

As the initial checkpointing related KEP-2008 was focusing on "Forensic
Container Checkpointing", the listed user stories were limited to the
forensic use case. The checkpoint/restore technology, however, opens up
the possibility to many different use cases. In the context of this KEP
("Kubectl Checkpoint") additional user stories, which were omitted for
simplicity reasons in KEP-2008, are listed. Listing additional user
stories updates the checkpoint related KEP to how people are using
the existing checkpointing functionality already today. Some of the
user stories would benefit from additional integration into Kubernetes,
but most user stories can already be used today. Implementation of this
KEP would make the user experience more pleasant. Especially the container
migration user story has many possibilities for optimization. CRIU, which is on
the lowest level of the checkpoint/restore stack, offers the possibility to
decrease container downtime during migration by techniques well known in virtual
machine migration like pre-copy or post-copy migration.

#### Forensic Container Checkpointing

To analyze unusual activities in a container, the container should
be checkpointed without stopping the container or without the container
knowing it was checkpointed. Using checkpointing it is possible to take
a copy of a running container for forensic analysis. The container will
continue to run without knowing a copy was created. This copy can then
be restored in another (sandboxed) environment in the context of another
container engine for detailed analysis of a possible attack.

#### Fast Container Startup - Warm Start

In addition to forensic analysis of a container checkpointing can be used to
offer a way to quickly start containers. This is especially useful for
containers that need a long time start. Either the software in the container
needs a long time to initialize by loading many libraries or the container
requires time to read data from a storage device. Using checkpointing it is
possible to wait once until the container finished the initialization and save
the initialized state to a checkpoint archive. Based on this checkpoint archive
one or multiple copies of the container can be created without the need to wait
for the initialization to finish. The startup time is reduced to the time
necessary to read back all memory pages to their previous location.

This feature is already used in production to decrease startup time of
containers since 2016. One example describing this setup is the session
(["Container Checkpoint/Restore at Scale for Fast Pod Startup Time"][lV0Y]) from
2021 ([Recording of "Container Checkpoint/Restore at Scale for Fast Pod Startup
Time"][BXVyszsbYmg]). There are also companies basing their product on
CRIU for quicker startup like ([cedana][cedana]) or ([weaversoft][weaversoft]).

Another similar use case for quicker starting containers has been reported in
combination with persistent memory systems. The combination of checkpointed
containers and persistent memory systems can reduce startup time after a reboot
tremendously:

- [MemVerge Memory Machine Solution][memoryMachine]
- [Parallel Processing with Persistent Memory Clones][memoryClones]

#### Optimize Resource Utilization

This use case is motivated by interactive long running containers. One very
common problem with things like Jupyter notebooks (also see [Optimizing Resource
Utilization for Interactive GPU Workloads with Transparent Container
Checkpointing][optimizingGPUWorkloads])or remote development environments is
that users want to have them running for a very long time to be able to come
back to them whenever needed. To not block unused resources while these
containers are not used checkpointing enables the possibility to make a
copy of these stateful workloads which can be restored when the user wants to
continue this interactive workload.

This use case is especially interesting for resources that are limited like
GPUs and discussed in multiple conference presentations in the last years:

- [Achieving K8S and Public Cloud Operational Efficiency using a New Checkpoint/Restart Feature for GPUs][cloudEfficiency]
- [Optimizing Resource Utilization for Interactive GPU Workloads with Transparent Container Checkpointing][optimizingGPUWorkloads]
- [Transparent, Infra-Level Checkpoint and Restore for Resilient AI/ML Workloads at Scale][workloadsAtScale]
- [Investigating Checkpoint and Restore for GPU-Accelerated Containers][acceleratedContainers]

#### Container Migration

One of the main use cases for checkpointing and restoring containers is
container migration. An open issue asking for container migration in
Kubernetes exists since 2015: [#3949][migration-issue].

With the current Alpha based implementation container migration is already
possible as documented in [Forensic Container Checkpointing Alpha][kubernetes-blog-post].

The following tries to give an overview of possible use cases for
container migration.

##### Fault Tolerance

Container migration for fault tolerance is one of the typical reasons to
migrate containers or processes. It is a well researched topic especially
in the field of high performance computing (HPC). To avoid loss of work
already done by a container the container is migrated to another node before the
current node crashes. There are many scientific papers (i.e., [Understanding
failures in petascale computers][petascaleFailures] or[A large-scale study of
failures in high-performance computing systems][failureStudy]) describing how to
detect a node that might soon have a hardware problem. The main goal of using
container migration for fault tolerance is to avoid loss of already done work.
This is, in contrast to the forensic container checkpointing use
case, only useful for stateful containers.

With GPUs becoming a costly commodity, there is an opportunity to help
users save on costs by leveraging container checkpointing to prevent
re-computation if there are any faults.

##### Load Balancing

Container migration for load balancing is something where checkpoint/restore
as implemented by CRIU is already used in production today. A prominent example
is Google as presented at the Linux Plumbers conference in 2018:
[Task Migration at Scale Using CRIU][task-migration]

Migrating containers with open network connections might not always work.
CRIU has the ability to close open TCP connections upon checkpointing or
migrate open TCP connections from one host to another. Migrating open TCP
connections requires additional support on the network layer to move the
IP address to another host. In combination with Podman with have implemented
the functionality to migrate a container with established TCP connections
by reusing the same IP address of the container on the destination host. It
still requires that the infrastructure also handles having the same container
IP address on another host. For the case where TCP connections are closed
by CRIU the application needs to be able to re-open TCP connections if they
have been closed.

If multiple containers are running on the same physical node in a cluster,
checkpoint/restore, and thus container migration, open up the possibility
of migrating containers across cluster nodes in case there are not enough
computational resources (e.g., CPU, memory) available on the current node.
While high-priority workloads can continue to run on the same node, containers
with lower priority can be migrated. This way, stateful applications with low
priority can continue to run on a different node without loosing their progress
or state. This might be especially useful in combination with
`InPlacePodVerticalScaling=true` when resources are scaled and a cluster node
could get overloaded.

This functionality is especially valuable in distributed infrastructure services
for AI workloads, as it helps reduce the cost of AI by maximizing the aggregate
useful throughput on a given pool with a fixed capacity of hardware accelerators.
Microsoft's globally distributed scheduling service, [Singularity][singularity],
is an example that demonstrates the efficiency and reliability of this mechanism
with deep learning training and inference workloads.

Another reason for load balancing of running containers might also be energy
efficiency. By consolidating workloads on less cluster nodes it might be possible
to shut down nodes to save energy.

##### Spot Instances

Yet another possible use case where checkpoint/restore is already used today
are spot instances. Spot instances are usually resources that are cheaper but
with the drawback that they might shut down with very short notice. With the
help of checkpoint/restore workloads on spot instances can either be
checkpointed regularly or the checkpointing can be triggered by a signal.
Once checkpointed the container can be moved to another instance and
continue to run without having to start from the beginning.

One example of how to deal with the limited time between the shutdown signal and
the actual shutdown can be seen at [Fast checkpointing with criu-image-streamer
][imageStreamer]

##### Scheduler Integration

All of the above-mentioned container migration use cases currently require
manual
checkpointing, manual transfer of the checkpoint archive, and manual restoration
of the container (see [Forensic Container Checkpointing Alpha][kubernetes-blog-post]
for details). If all these steps could be automatically performed by the scheduler,
it would greatly improve the user experience and enable more efficient resource
utilization. For example, the scheduler could transparently checkpoint, preempt,
and migrate workloads across nodes while keeping track of available resources and
identifying suitable nodes (with compatible hardware accelerators) where a container
can be migrated. However, scheduler integration is likely to be implemented at a later
stage and is a subject for future enhancements.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

As already mentioned in KEP-2008 one of the main risks creating a checkpoint
is that all memory pages are written to disk. This includes all secrets like
passwords, random numbers, private keys. The checkpoint archive that is
written to the checkpoint directory of the kubelet is only accessible by
the root user which already has the capability to access all that data.

One approach to ensure that non of the secret information stored in
the checkpoint archive is accessible would be to have encrypted checkpoint
archives.  This is currently being developed in CRIU: [Add support for encrypted
images][pr2297]

In its first incarnation this KEP will not transfer the checkpoint archive
from the node it was created on. The checkpoint archive will be created on
the node the container is running on and the user will get the information
about the node and the path to the checkpoint archive. This also means that
nothing changes from a security point of view compared to KEP-2008.

To ensure only users with appropriate access can create a checkpoint
a new role needs to be introduced in the RBAC system. For the resource "pods"
a new verb "checkpoint" will be introduced.

TODO: Security review is still open.

TODO: UX review is still open.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

Currently the design details are based on the existing pull request: [Add
'checkpoint' command to kubectl][pr120898]

The API server is extended to handle checkpoint requests from *kubectl*:

```go
// PodCheckpointOptions is the query options to checkpoint a container
type PodCheckpointOptions struct {
    metav1.TypeMeta

    // Container which to checkpoint
    Container string

    // timeout as specified in the kubelet API endpoint
    timeout int64
}
```

Also, *kubectl* is extended to call this new API server interface. The API
server, upon receiving a request, will call the kubelet with the corresponding
parameters passed from *kubectl*. Once the checkpoint has been successfully written
to disk *kubectl* will return the name of the node as well as the location of
the checkpoint archive to the user:

```shell
 $ kubectl alpha checkpoint test-pod-1 -c container-2
 Node:                  127.0.0.1/127.0.0.1
 Namespace:             default
 Pod:                   test-pod-1
 Container:             container-2
 Checkpoint Archive:    /var/lib/kubelet/checkpoints/checkpoint-archive.tar
```

The `kubectl checkpoint` behaviour is modeled after existing commands like
`kubectl exec`.

The current pull request implements the `kubectl checkpoint` sub-command
as `kubectl alpha checkpoint` as it will start as an alpha feature.

Checkpointing a container will be an operation that is only possible
with access to the appropriate role in RBAC.

Feature gate name: ContainerCheckpointAPI

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

The existing pull request includes unit and end-to-end tests: [Add 'checkpoint'
command to kubectl][pr120898]


##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

Not sure.

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

TODO: unit tests do already exist, need to fill in this data

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

TODO: integration tests do already exist, need to fill in this data

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

#### Alpha

- [X] Feature implemented behind a feature flag
- [X] Ensure proper tests are in place

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

TODO: unclear

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

This KEP depends on three components. API server, *kubectl* and *kubelet*.

If *kubectl* is too old for this feature it will just return `unknown command`.
If the API server it too old for this feature or the feature gate is disabled
*kubectl* will fail with an appropriate error that the feature does not exist.
The same will happen if the *kubelet* does not support the checkpoint API
endpoint.

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

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ContainerCheckpointAPI`
  - Components depending on the feature gate: ???
  - Feature gate `ContainerCheckpoint` enabled in the *kubelet*
- [ ] Other
  - Describe the mechanism: container engine and *kubelet* need to support checkpointing (CRI-O and containerd do as of today)
  - Will enabling / disabling the feature require downtime of the control
    plane? TODO: Probably no
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? TODO: Probably no

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes. By disabling the feature gate `ContainerCheckpointAPI` again.

###### What happens if we reenable the feature if it was previously rolled back?

Checkpointing containers will be possible again.

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

Currently the test will automatically be skipped if the feature is not enabled.

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

CRIU needs to be installed on the node, but on most distributions it is already
a dependency of runc/crun. It does not require any specific services on the
cluster.

The *kubelet* must have the feature gate `ContainerCheckpoint` enabled.

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

No

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

The API server will be extended to handle
`/pods/<pod>/<namespace>/checkpoint/PodCheckpointOptions{}`. This API will be
called from *kubectl*. The estimated throughput is unclear. The API call is
not expected to be called periodically. Once per checkpoint request.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

```go
// PodCheckpointOptions is the query options to checkpoint a container
type PodCheckpointOptions struct {
    metav1.TypeMeta

    // Container which to checkpoint
    Container string

    // timeout as specified in the kubelet API endpoint
    timeout int64
}
```

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

Probably. Not sure.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

Probably not as it is an API call independent of existing API calls.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

This has already been described in KEP-2008, the disk usage of the node running
the to be checkpointed container can increase as described below.

During checkpointing each memory page will be written to disk. Disk usage will increase by
the size of all memory pages in the checkpointed container. Each file in the container that
has been changed compared to the original version will also be part of the checkpoint.
Disk usage will overall increase by the used memory of the container and the changed files.
Checkpoint archive written to disk can optionally be compressed. The current implementation
does not compress the checkpoint archive on disk.  The cluster administrator is
responsible for monitoring disk usage and removing excess data.

The kubelet will request a checkpoint from the underlying CRI implementation. In
the checkpoint request the kubelet will specify the name of the checkpoint
archive as `checkpoint-<podFullName>-<containerName>-<timestamp>.tar` and also
request to store the checkpoint archive in the `checkpoints` directory below its
root directory (as defined by `--root-dir`). This defaults to
`/var/lib/kubelet/checkpoints`.

To avoid running out of disk space an operator has been introduced: <https://github.com/checkpoint-restore/checkpoint-restore-operator>


###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

See above and KEP-2008.

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

It will fail like any other *kubectl* sub-command targeting the API server.

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

From KEP-2008:

- The creation of the checkpoint archive can fail.
  - Detection: Possible return codes are:
    - 401: Unauthorized
    - 404: Not Found (if the ContainerCheckpoint feature gate is disabled)
    - 404: Not Found (if the specified namespace, pod or container cannot be found)
    - 500: Internal Server Error (if the CRI implementation encounter an error
    during checkpointing (see error message for further details))
    - 500: Internal Server Error (if the CRI implementation does not implement
    the checkpoint CRI API (see error message for further details))
    - Also see: <https://kubernetes.io/docs/reference/node/kubelet-checkpoint-api/>
  - Mitigation: Do not checkpoint a container that cannot be checkpointed by CRIU.
  - Diagnostics: The container engine will provide the location of log file created
    by CRIU with more details.
  - Testing: Tests are currently covering if checkpointing is enabled in the kubelet
    or not as well as covering if the underlying container engine supports the
    corresponding CRI API calls. The most common checkpointing failure is if the
    container is using an external hardware device like a GPU or InfiniBand which
    usually do not exist in test systems.

Checkpointing anything with access to an external hardware device like a GPU or
InfiniBand can fail. For each device a specific plugin needs to be added to CRIU.
For AMD GPUs this exists already today, but other GPUs will fail to be checkpointed.
If an unsupported device is used in the container the *kubelet* API endpoint will
return `500` with additional information in the error message.

###### What steps should be taken if SLOs are not being met to determine the problem?

As checkpointing is an optional feature outside of the pod lifecycle SLOs probably should
not be impacted. If SLOs are impacted then administrators should no longer call the
checkpoint *kubelet* API endpoint. During Alpha and Beta phase the feature gate can
also be used to turn the feature of. At this point in time it is unclear, but for a
possible GA phase this is maybe a feature that needs to be opt-in or opt-out. Something
that can be turned off during startup or runtime configuration.

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

- 2024-05-08: Initial version of this KEP
- 2025-03-10: Added more references; especially to the user story section

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

This has already been discussed in KEP-2008. Not sure it makes sense here.
This KEP is just about wiring through existing API endpoints from the *kubelet*
to *kubectl*.

## Alternatives

This has already been discussed in KEP-2008. Not sure it makes sense here.
This KEP is just about wiring through existing API endpoints from the *kubelet*
to *kubectl*.

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

[pr120898]: https://github.com/kubernetes/kubernetes/pull/120898
[migration-issue]: https://github.com/kubernetes/kubernetes/issues/3949
[kubernetes-blog-post]: https://kubernetes.io/blog/2022/12/05/forensic-container-checkpointing-alpha/
[task-migration]: https://lpc.events/event/2/contributions/69/attachments/205/374/Task_Migration_at_Scale_Using_CRIU_-_LPC_2018.pdf
[singularity]: https://arxiv.org/abs/2202.07848
[pr2297]: https://github.com/checkpoint-restore/criu/pull/2297
[BXVyszsbYmg]: https://youtu.be/BXVyszsbYmg
[lV0Y]: https://sched.co/lV0Y
[cedana]: https://www.cedana.com/
[weaversoft]: https://weaversoft.io/
[memoryMachine]: https://www.supermicro.com/solutions/solution-brief_MemVerge.pdf
[memoryClones]: https://memverge.com/parallel-processing-with-persistent-memory-clones/
[optimizingGPUWorkloads]: https://fosdem.org/2025/schedule/event/fosdem-2025-4042-optimizing-resource-utilization-for-interactive-gpu-workloads-with-transparent-container-checkpointing/
[cloudEfficiency]: https://www.nvidia.com/en-us/on-demand/session/gtc24-p63184/
[workloadsAtScale]: https://sched.co/1tx7u
[acceleratedContainers]: https://sched.co/1aPu9
[petascaleFailures]: https://doi.org/10.1088/1742-6596/78/1/012022
[failureStudy]: https://dl.acm.org/doi/10.1109/DSN.2006.5
[imageStreamer]: https://lpc.events/event/7/contributions/641/
