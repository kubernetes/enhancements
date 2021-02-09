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
# KEP-2464: Kubetest2 CI migration

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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

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

Kubernetes currently uses [Kubetest](https://github.com/kubernetes/test-infra/tree/master/kubetest) as the interface for launching and running e2e tests. 
It also uses a set of scripts called [“scenarios”](https://github.com/kubernetes/test-infra/tree/master/scenarios) for running common use cases such as setting up end-to-end tests or pushing CI builds.  
Kubetest and scenarios have both been [deprecated](https://github.com/kubernetes/test-infra/tree/master/kubetest#deprecation-notice), [(2)](https://github.com/kubernetes/test-infra/tree/master/scenarios#deprecation-notice) for a while, with a recommendation to move jobs to [kubetest2](https://github.com/kubernetes-sigs/kubetest2). 

This KEP proposes a plan for migration/deprecation of CI jobs that are in the critical path of a release.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals

- Converge towards a single community-supported e2e launcher tool for kubernetes
- Ensure all kubernetes/kubernetes release-blocking and merge-blocking jobs use the same e2e test tool, to reduce community maintenance burden, increase community troubleshooting awareness
- Document guides for migrating jobs away from kubetest to kubetest2 to enable self-service migration of non-blocking jobs
- Accelerate the [migration from bootstrap.py to pod-utilities](https://github.com/kubernetes/test-infra/issues/20760) since Kubetest2 does not depend on bootstrap
- Enable SIG Release to deprecate legacy [push-build.sh](https://github.com/kubernetes/release#legacy)

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- Implementing kubetest2 deployers for all existing kubetest providers
    - kubetest2 enables implementing custom deployers out-of-tree so replacing all of the deployers upstream is not a hard requirement
- Migrate all 1700+ e2e jobs away from the deprecated e2e test tools (e.g. scenarios, bootstrap, kubetes)
    - This KEP only covers migrating all kubernetes/kubernetes release-blocking and merge-blocking jobs. 
    - We propose enforcing no further changes to the remaining deprecated tools to incentivize community migration and improvements to kubetest2.
    - Other community efforts are already underway (e.g. [removing the kubernetes_build scenario](https://github.com/kubernetes/release/issues/1711))
- Backporting changes to older release branches
    - In general, no changes should be made to the release branches or their CI and the old jobs will age out.
- Decoupling from kubekins to a more minimal image
    - The kubekins image has become a kitchen-sink over the years, used by more jobs than necessary, and containing more dependencies than required for effective kubetest2 usage
    - Migrating kubekins away from bootstrap as a base image is only feasible once all jobs have migrated away from bootstrap, which is a non-goal of this KEP


## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

For the development (main/master) branch only, **NOT** the existing release branches:

- Create shadow canary jobs for each [presubmit-kubernetes-blocking](https://testgrid.k8s.io/presubmits-kubernetes-blocking), [release-blocking](https://testgrid.k8s.io/sig-release-master-blocking) to use kubetest2 as optional, non-blocking
- Monitor these canary jobs till they are stable
- Make kubetest2 jobs non-optional, non-blocking
- Make kubetest2 jobs non-optional, blocking

No changes should be made to the release branches or their CI.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

#### Story 2

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

- Stage 1 of  shipping the kubetest2 binary involves keeping HEAD at kubetest2 unbroken (same as kubetest currently). 
- Stage 2 of the proposal involves monitoring a job to become stable which might encounter discrepancies between its existing kubetest2 counterpart.


## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### CI Jobs: 

The following deployers/testers are used as part of the jobs in consideration:

- KIND:  

    **KIND jobs neither uses scenarios nor kubetest so most KIND jobs are noops.**

    - KIND is already a supported kubetest2 deployer for basic end-to-end testing.
    - Possible future consideration for kubetest2 kind jobs:
        - A typical kind job runs using the krte image, downloads the latest kind binary, sets up the environment for the test job using e2e-k8s.sh
        - Future additions required to the KIND deployer
            - support fetching latest kind binaries
            - move over functionality from e2e-k8s.sh as part of the kind deployer
- GCE:

    GCE jobs are the ones that use scenarios and kubetest the most.
    Most of the GCE deployer logic has been ported over to kubetest2, see implementation history for details.
    We also have a [canary job for GCE conformance](https://testgrid.k8s.io/conformance-gce#Conformance%20-%20GCE%20-%20master%20-%20kubetest2) using kubetest2.


- Node:
    
    Most of the node e2e logic is consolidated as part of the e2e test themselves. 
    So the jobs basically create [noop clusters](https://github.com/kubernetes/test-infra/blob/master/kubetest/node.go) which will [need to be implemented](https://github.com/kubernetes-sigs/kubetest2/issues/45) in kubetest2. 
    This noop deployer will also be useful for other build jobs.
    Additionally, we can have a short wrapper over the noop deployer specifically for the node deployer to mimic what kubetest does, e.g. setting up GCP SSH Keys

- Scale:
    
    Scale tests use the clusterloader2 framework in [kubernetes/perf-tests](https://github.com/kubernetes/perf-tests). kubetest2 already has support for clusterloader2 tester.

- Ginkgo:
    
    All of the jobs (apart from scale jobs) are using the kubernetes test/e2e framework which uses ginkgo for the end-to-end tests. kubetest2 already supports this ginkgo tester.
    Some tests require checking for version-skew between the client and server version, support for which will need to be added in kubetest2.


#### Scenarios:

There are mainly 3 scenarios that are currently used:

- [`scenario=execute`](https://github.com/kubernetes/test-infra/blob/master/scenarios/execute.py) is used to run arbitrary test scripts. It can be replaced by the kubetest2 exec tester.

    ```shell script
    kubetest2 --test=exec – script.sh
    ```

- [`scenario=kubernetes_e2e`](https://github.com/kubernetes/test-infra/blob/2abb7ff26579325f6a335990003f665755c96d7a/scenarios/kubernetes_e2e.py) 
 is used for setting up the environment for end-to-end jobs.
    - It specifies it’s own set of flags to customize the setup all of which are deprecated and unused.
    - It has a lot of functionality for setting up Kops clusters which will need to be ported over to be part of the kubetest2 kops deployer. They are currently being monitored in testgrid here: https://testgrid.k8s.io/kops-kubetest2#e2e-kubetest2

    Most other usages in jobs are used to indirectly invoke kubetest, which will be replaced by their equivalent kubetest2 commands.

    ```shell script
    kubetest2 <deployer> --build --up --down --deployer-specific-setup-flags
    ```
- [`scenarios=kubernetes_build`](https://github.com/kubernetes/test-infra/blob/2abb7ff26579325f6a335990003f665755c96d7a/scenarios/kubernetes_build.py) which is primarily used for pushing CI builds to GCS.
    This is where we are also using the legacy push-build scripts (in addition to kubetest --stage).
    [These will be replaced by KREL.](https://github.com/kubernetes/release/issues/1711)

    Optionally they can also be migrated to use the kubetest2 KREL integration.
    ```shell script
    kubetest2 noop --build --stage=gs://bucket
    ```

    Additionally, there is exactly one usage of `scenario=canarypush` which can be replaced by an equivalent script and migrated to using kubetest2 exec tester.

#### Managing kubetest2 dependency

- Add a script to kubekins-e2e to fetch pre-built latest kubetest2 binaries (stored in GCS) as part of each job
- Distribute a tagged version of kubetest2 binary for each release and add to variants.yaml for kubekins-e2e
- Publish a kubetest2 specific images (optional)


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

- This will be declared **alpha** when:
    - Jobs fetch latest pre-built kubetest2 binaries at runtime
    - GCE jobs have corresponding kubetest2 shadow jobs on k/k master
    - Noop support is added for node deployer and build


- This will be declared **beta** when:
    - A guide for migration from kubetest to kubetest2 exists
    - GCE jobs are monitored to be stable, and have moved over to kubetest2
    - Node, Scale jobs have corresponding shadow jobs on k/k master

- This will be declared **GA** when:
    - Jobs use pre-installed tagged versions of kubetest2 binaries
    - Tracking issues exist for each SIG to migrate their jobs from kubetest to kubetest2 
    - All kubernetes/kubernetes master presubmit-kubernetes-blocking, release-blocking jobs are using kubetest2


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

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include 
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
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

n/a (this is a build/test change for the kubernetes project, maintains parity with existing merge-blocking and release-blocking jobs)

### Rollout, Upgrade and Rollback Planning

n/a (this is a build/test change for the kubernetes project, maintains parity with existing merge-blocking and release-blocking jobs)

### Monitoring Requirements

n/a (this is a build/test change for the kubernetes project, not relevant to monitoring production clusters)

### Dependencies

n/a (this is a build/test change for the kubernetes project, maintains parity with existing merge-blocking and release-blocking jobs)

### Scalability

n/a (this is a build/test change for the kubernetes project, maintains parity with existing merge-blocking and release-blocking jobs, including scalability jobs)

### Troubleshooting

n/a (this is a build/test change for the kubernetes project, not relevant to troubleshooting problems in production cluster)

## Implementation History

- 2020-07-07: Kind deployer support added in kubetest2 
- 2020-07-15: Boskos support is added to kubetest2
- 2020-08-10: GCE deployer support added in kubetest2. see also [this document](https://docs.google.com/document/d/157nSQNyy9cOjw4izG0rUs_9z9Suy31JQtng_g2peTsw/edit#heading=h.5irk4csrpu0y) for the implementation details
- 2020-07-30: Ginkgo tester support added in kubetest2
- 2020-09-14: Clusterloader2 tester support added in kubetest2


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

- Continue using kubetest, scenarios, push-build.sh as-is
    
    This has several drawbacks:
    - Kubetest lack proper owners and active contribution. Support is mostly added as bugs are discovered.
    - Scenarios are mostly legacy hacky scripts, which will provide better maintainability  if it is migrated to a proper framework.
    - Push-build.sh is another legacy script which SIG Release is actively moving away from with efforts such as KREL, Anago to be rewritten in go.

- Improve kubetest, scenarios, push-build.sh

    This has several drawbacks:
    - Kubetest lacks a good separation of concerns which leads to changes having a large blast radius. See: [why kubetest2](https://docs.google.com/document/d/1Dc7xg9lq4cxdDuz20YZjunuL5eju2WIXwO32praC5hs/edit#heading=h.ppb3an7ey1of)
    - Most scenarios are obsolete and have only trivial usages all of which can be handled by kubetest2 through a single entrypoint.
    - push-build.sh is written in bash which is hard to maintain and provides no additional value over it’s go equivalent KREL push.


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
