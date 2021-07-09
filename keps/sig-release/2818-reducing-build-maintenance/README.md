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
# KEP-2818: Reducing Build Maintenance in CIP

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
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
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
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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
Deprecate Bazel within the k8s-container-image-promoter.

## Motivation
In Feb 2020, the Kubernetes community produced a proposal to remove Bazel-based build infrastructure from kubernetes/kubernetes. Justified by decreased dependencies and a simpler build process, several of the project’s subprojects/repositories now primarily rely on make for builds. In an effort to attain the same benefits, CIP aims to reduce its own build maintenance by removing Bazel.

### Goals
Prow jobs must remain stable - While replacing existing Bazel infrastructure, it's important that any changes do not affect any Prow jobs. More specifically, the presubmit, postsubmit, and periodic jobs defined for CIP in kubernetes/test-infra are required for CI/CD. Any proposed solution should avoid interfering with these existing operations, and are crucial for future development and deployment.

Preserving functionality of e2e tests - It’s important when removing Bazel to preserve the business logic of the e2e tests. However, both tests rely on the predictability of Docker digests when verifying image promotions. Therefore, proposed solutions must maintain static golden image digests. This may warrant the adoption of another technology which mimic’s Bazel’s behavior, or avoid image building altogether. Either way, both e2e tests must remain unaffected.

Completely remove Bazel from CIP - The result of this project must remove all references of Bazel. Therefore, machines running the CIP source code, such as existing Prow jobs, should no longer require the installation of Bazel for compilation or execution. Since reducing build maintenance is the primary reason behind removing Bazel, adding additional tools to the build process should be avoided.

### Non-Goals
Improve performance - The execution of e2e tests, building of containers or code does not need to speed up. Perceived or measured performance gains from proposed solutions is not a focus of this project.

Complicate e2e test behavior - The behavior of e2e tests should change as little as possible when removing Bazel. Although deterministic image digests are required, desirable solutions must avoid adding complexity to the setup of e2e tests.


## Proposal

Bazel's removal will leverage existing project dependencies (Docker and Golang) to manage binary and container builds. Since Make is already relied upon for triggering specific behaviors, these targets will be be implemented with a function replacement for their existing Bazel function.

For example:
```
bazel build //cmd/cip
```
can be substituted for:
```
go build ./cmd/cip
```

In addition, go's dependency management system (go modules) is already setup within the project. This can relieve the need for Gazelle to generate and update existing Bazel BUILD files.

For containerization, Docker's CLI is more than capable to script the required image bundles previously defined in BUILD files. Existing make targets can trigger docker directly to pull, build, or push CIP images. End-to-end tests, which rely on static image definitions, can utilize local archives committed to source control. When testing begins, docker will load these images from tarbal, removing the need to build images with Bazel all together.

### Caveats
The nature of tarball archives conceal the information they compress. Once decompressed, Docker archives reveal multiple json files which define the layers, tag information, and versions of the image. Docker can understand these files to reliably reproduce the saved image. However, since this information is saved in a compressed form, developers will not be able to understand what these golden images contain directly and will have to untar them to inspect them, or use `docker load` themselves manually. 


### Risks and Mitigations
In a scenario where the contents of the golden images are lost,  recreating these contents from archives  would be a great challenge. Realistically, the CIP tool does not need to know the contents of the images promoted. The e2e tests currently require the test images to be static as the same digests are used between multiple e2e test runs. In an event where all golden image backups are lost, these manifests would need to be modified to match the digests of new test images. 

## Design Details
The CIP repository is written entirely in Golang, which simplifies the compilation process. Since the goal is to strip all Bazel dependencies, this section outlines the function of existing Bazel rules and recommends alternatives as a functional replacement.

### Golang
Building all gocode from CIP should only utilize existing targets from the Makefile. Most targets however use a Bazel wrapper to handle code compilation. This means Bazel is either invoking the build or run command. Fortunately Golang’s CLI is a direct replacement for such tasks.

For example, this:
```makefile
# Makefile (with Bazel)
REPO_ROOT:=$(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))

.PHONY: build
build: ## Bazel build
    bazel build //cmd/cip:cip \
        //test-e2e/cip:e2e \
        //test-e2e/cip-auditor:cip-auditor-e2e

.PHONY: install
install: ## Install
    bazel run //:install-cip -c opt -- $(shell go env GOPATH)/bin
```
can be transformed into this:
```makefile
# Makefile (without Bazel)
REPO_ROOT:=$(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))

.PHONY: build
build: ## Bazel build
    go build $(REPO_ROOT)/cmd/cip:cip && \
    go build $(REPO_ROOT)/test-e2e/cip:e2e && \
    go build $(REPO_ROOT)/test-e2e/cip-auditor:cip-auditor-e2e

.PHONY: install
install: ## Install
    go install $(REPO_ROOT)/cmd/cip
```
This one-to-one replacement works when building, installing, running and testing gocode since these Bazel rules are just using go tools behind the scenes. Therefore, BUILD files containing go_library, go_binary and go_test rules can be eliminated.

Gazelle is currently used as a tool to generate and update Bazel build files for Go projects that follow the conventional "go build" project layout. It is intended to simplify the maintenance of Bazel Go projects as much as possible. However, since Bazel is being removed, so will the use of gazelle. Fortunately, the CIP tool already uses Go Modules which handles dependency vendoring. Therefore, the removal of Gazelle, alongside Bazel, can allow go module tools like tidy to handle importing and pruning dependencies before build time.

### Docker
The go_image, container_image, container_layer, container_pull and container_bundle rules allow Bazel to define and build Docker containers. However, since Docker is already a dependency of CIP, it can act as a functional replacement for some of these commands.

| Existing Rule(s)      | Docker Equivalent | Without Dockerfile     |
| :---        |    :----:   |          ---: |
| container_bundle      | docker save [IMAGE...]       | Yes   |
| container_pull   | docker pull NAME[:TAG|@DIGEST]        | Yes     |
| container_layer, container_image, go_image   | docker build PATH | URL        | No     |

Though the first two Bazel rules seem to have straightforward equivalents, Docker’s build command will not complete everything Bazel accomplished. For instance, a Dockerfile must be provided for all images looking to be built. Additionally, Bazel builds images deterministically - digests remain constant for each build. This behavior is not consistent with Docker which fingerprints each image digest with a timestamp, making each digest unique. Therefore, replacing bazel build with docker build will pose a problem for golden images. 

#### E2E Tests
Since Docker can’t reproducibly build static digests, the testing behavior will cause test failures. Replacing bazel build for docker build would generate unique images that would clash with existing test manifests.

#### Non-Approach: Generate Test Manifests
If docker build produces new golden image digests each time, couldn't we also generate new test manifests to use these new digests? This would suggest the following behavior:

[IMAGE 1]

This adds two new steps to the behavior of the e2e tests. Before each test, old test manifest files are deleted, as they contain old digest images. After every golden image is built with docker, their unique image digests are saved into new test manifest files. Such an approach would allow e2e tests to pass, but produce some unwanted side-effects.

First, repeating the same e2e test twice is impossible, as the prior test fixtures are discarded. This makes debugging quite difficult since tests could no longer be repeated with the same images or manifests. Additionally, adding two extra steps to the behavior of the e2e tests complicates the testing process which is a non-goal of this project. Therefore, adopting docker build with this approach is non-viable.

#### Approach #1: Static Hosting
A simpler approach would be to host static golden images in a project owned image repository. This would remove the steps of building the same images for every PR. Instead, these images could be built once and permanently live in an isolated directory. Assuming these images now permanently reside in the source image repository, below is an example of the modified e2e-test behavior.

[IMAGE 2]

In this scenario, there’s no need to clear the src repo since the golden images will already reside there. The destination still needs to be purged in order to remove any residual testing artifacts. However, this approach avoids image building altogether, thus removing the need for pushing images in setup.

##### Pros
This simplification of e2e tests streamline the number of steps in order to set up promotion. Less steps improve the robustness of the testing procedure as a whole. Additionally, since both cip and cip-auditor tests run multiple sub-tests with multiple golden images, this modified testing strategy should yield performance gains. Though quickened e2e-tests cannot be verified without implementation, it may be a desirable side effect of this approach.

##### Cons
Since golden images will never be built or pushed to src in the test cycle, they must remain static at all times. Though image migrations are very rare for CIP, changing the placement or images in the src repository would result in immediate Prow job failures resulting in a halt of mergeable PRs. Such a hangup would disrupt the development of CIP and should be avoided at all costs.

To protect the integrity of the golden images, the CIP’s test service-account should have read-only permissions to the src repository. This would thwart Prow from modifying the test fixtures during testing. Developers also pose a risk of tampering with these static images. It's imperative that the specific testing directory, within the src repository, is well documented in the CIP source code.

#### Approach #2: Tarball Image Loading (recommended)
An even better approach, which also avoids dynamic image builds, could make use of Docker’s save command. Since all golden images should remain static, they can be archived and source controlled in the CIP repository. Whenever needed, these tarball archives can be loaded back into container images. As container images, they can be pushed to GCR or built locally. What’s desirable about this process is the circumvention of building an image from source which would have modified the image digest. Loading and saving archives retains all digest information and solves the issue of deterministic images.

Using the busybox image as an example, the following script outlines this behavior:
```bash
#!/usr/bin/env bash
docker pull busybox
docker save busybox -o archive.tar
docker load -i archive.tar
# push image
docker tag busybox gcr.io/testing/example
docker push gcr.io/testing/example
```
Running this script multiple times will always produce the same image digest. Therefore, docker save and docker load from tarballs provides similar reproducible builds as Bazel. Below is the behavior of the e2e tests when adopting this approach:

[IMAGE 3]

Notice how this series of events looks almost identical to the original flow of e2e tests. It replaces the second build step with loading images from local archives. This simple adjustment removes the need for any extra setup modification while preserving existing business logic.

##### Pros
This approach preserves the existing test behavior, minimizing the complexity of implementing this change. Tests will only need to replace the existing manifest creation with loading from a tarball. Of the two proposed approaches, this is simpler.

##### Cons
The existing tar files must not be modified for tests to work. This would mean committing all archives to source control within the CIP repository. It’s imperative the function of these files are well documented and not moved or modified. Though if these tarballs were moved to another directory or project, it would cause the existing e2e tests, which use them, to fail and raise concern. Such a PR would not be allowed to merge, making this a low risk.


### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

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

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

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
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?
No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?
No

###### What happens if we reenable the feature if it was previously rolled back?
N/A

###### Are there any tests for feature enablement/disablement?
Existing tests existing as make targets, triggered by Prow.

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
- Docker
- Goland

###### Does this feature depend on any specific services running in the cluster?
For Prow Jobs running particular make targets that require docker, the docker-in-docker feature must be enabled.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?
No
###### Will enabling / using this feature result in introducing new API types?
No
###### Will enabling / using this feature result in any new calls to the cloud provider?
No
###### Will enabling / using this feature result in increasing size or count of the existing API objects?
No
###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?
No
###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?
No
### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

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

## Alternatives

### Ko Image Builder
Ko is a simplified container image builder specifically designed for Go applications. This tool makes it easy to build, name, and publish Docker images for Go applications without even requiring Docker as a dependency. Such a lightweight tool seemed to be a promising replacement for existing Bazel build tools.

However, the deal breaker is that existing golden images do not contain actual go programs, but small data files. Therefore, Ko would not work for our use case. Additionally, Ko doesn't help with deterministic image digests either. Although Docker doesn’t have this feature either, adding Ko as a dependency would not solve any problems. If anything, migrating to another container build system would add complexity.

### Kaniko Image Builder
Kaniko is a build tool which converts Dockerfiles to container images. With a variety of features for automation and defining build context, it falls short to help us reduce the build maintenance of the project. Since Docker can already accomplish all of this behavior, adding this tool doesn't provide a useful substitution for Bazel.
