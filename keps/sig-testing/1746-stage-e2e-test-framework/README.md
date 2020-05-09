<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up.  KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please ensure to complete all
  fields in that template.  One of the fields asks for a link to the KEP.  You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "title", "authors", "owning-sig",
  "status", and date-related fields.
- [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary", and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [x] **Create a PR for this KEP.**
  Assign it to people in the SIG that are sponsoring this process.
- [x] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly.  The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved.  Any KEP
marked as a `provisional` is a working document and subject to change.  You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused.  If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement", for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example.  If there are
new details that belong in the KEP, edit the KEP.  Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable` or significant changes once
it is marked `implementable` must be approved by each of the KEP approvers.
If any of those approvers is no longer appropriate than changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross cutting KEPs).
-->
# KEP-1746: Move E2E Test Framework into Staging

<!--
This is the title of your KEP.  Keep it short, simple, and descriptive.  A good
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
- [Key Terms](#key-terms)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Managing Internal Kubernetes Packages](#managing-internal-kubernetes-packages)
  - [Managing Test Packages](#managing-test-packages)
  - [Notes](#notes)
    - [Kubemark](#kubemark)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Key Terms

* **e2e test framework** refers to the e2e test framework, located in
  https://github.com/kubernetes/kubernetes/tree/master/test/e2e/framework and
  all of its sub-packages, unless otherwise explicitly stated.
* **Core Kubernetes** refers to code that is part of kubernetes/kubernetes,
  including the the repository excluding the contents of the staging directory.
  This also includes k8s.io/kubernetes/test but excludes
  k8s.io/kubernetes/test/framework and its subdirectories (e2e test framework).
* **Published Packages** refers to Go code packages that are available to
  Kubernetes end-users from source code present in the Kuberentes staging
  directory.


## Summary

<!--
This section is incredibly important for producing high quality user-focused
documentation such as release notes or a development roadmap.  It should be
possible to collect this information before implementation begins in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself.  KEP editors, SIG Docs, and SIG PM
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.
-->

In order to improve the user experience of all the people who have to maintain
e2e tests in Kubernetes and with the aim of reducing technical debt in e2e tests
(i.e., have more organized tests, better utility functions) we propose to work
towards moving the e2e test framework from its current place in
[kubernetes/test/e2e/framework](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/framework)
to the appropriate location under `k8s.io/kubernetes/staging/src/k8s.io/e2e-framework`.

Our main goal is to improve the experience of maintainers and building on top
of what we already have instead of starting from scratch.

Moving the e2e test framework into staging means that the e2e test framework
will be used and maintained as a component independent of core Kubernetes.
By doing this, the e2e test framework will only make use of public APIs and
will be self-contained.
It will also enable out-of-tree projects to import the e2e test framework
without importing all of Kubernetes.

Moving the e2e test framework to staging will also help maintainers of e2e tests
have tools to easily test any component that builds on Kubernetes in the same
manner as core Kubernetes.
People will be able to use all the same utility functions used in e2e tests in
core Kubernetes and will be able to use the e2e test framework's context to configure
their e2e test environments (i.e., manage information about the cluster being
used for e2e tests, the provider, ETCD).

This will also help external consumers of the e2e test framework by reducing
the amount of dependency management they have to do now when importing the e2e
test framework.


## Motivation

<!--
This section is for explicitly listing the motivation, goals and non-goals of
this KEP.  Describe why the change is important and the benefits to users.  The
motivation section can optionally provide links to [experience reports][] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

The e2e test framework is a Kubernetes component that has been evolving
organically to meet all the needs that contributors have in order to test
enhancements and behaviours of Kubernetes.
The SIG Testing sub-project
[Testing Commons](https://github.com/kubernetes/community/tree/master/sig-testing#testing-commons)
has been looking at ways of improving the user experience for contributors to
the Kubernetes community.
This KEP is part of that goal.

We foresee the following benefits coming out of this work:

1. Maintainability: the e2e test framework lacks a general API - there has been
  little discussion on what parts of the framework do what.
  From this work we plan to organize the code that is present and use only
  external libraries and APIs available for e2e tests.
  This decoupling will ensure the e2e test framework packages are better
  contained allowing for e2e tests to have a more standard flow.
2. External consumption: The Kubernetes ecosystem has grown considerably since
  the creation of the e2e test framework. Some projects, such as the cloud
  providers and CSI extensions, began within the core Kubernetes codebase and
  have been migrated out or are in the process of being migrated out. We want
  to provide them with the same set level of functionality they had in core
  without having to deal with the problems of importing the entirety of core
  Kubernetes.

There are two main components to the e2e test framework: its configuration
(used for setting up and running e2e tests) and its set of packages and utility
functions.
For this work, we plan on focusing on the latter.
The entrypoint into the framework, the configuration part, will stay as is.
Only the utility functions will be refactored.
What this means for consumers, is that the functions offered via the e2e test
framework will change but not the way e2e tests are setup and executed.

### Goals

<!--
List the specific goals of the KEP.  What is it trying to achieve?  How will we
know that this has succeeded?
-->

* Decouple the e2e test framework and all of its subpackages from any
  Kubernetes internal packages (i.e., pkg, other e2e packages).
* Impose import restrictions to prevent the use of core Kubernetes packages.
* Allow the e2e test framework to be easily used and without requiring the
  entirety of Kubernetes to be imported.
* Refactor the e2e test framework so that its sub-packages are well contained
  (i.e., `framework/pod/*` contains all necessary utilities to manage pods
  during e2e tests).
  * Update Kubernetes e2e tests to use the refactored version of any utility
    function we change.

### Non-Goals

* Change the way e2e tests/jobs are setup or executed.
* This enhancement is not directed at making the e2e test framework
  provider-agnostic. Cloud provider functionality will be kept as is unless it
  is used in inadequate e2e test framework sub-packages.
* The e2e test framework will be refactored as needed strictly to remove its
  dependence from other internal Kubernetes packages. It is not the goal of
  this enhancement to propose an entire redesign of the core e2e test framework
  (the part that handles the test context).

<!--
What is out of scope for this KEP?  Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation.  The "Design Details" section below is for the real
nitty-gritty.
-->

In order to cleanup and refactor the e2e test framework we will have to
decouple the e2e test framework from internal Kubernetes packages and to
organize test code in e2e tests and in the e2e test framework.

### User Stories

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system.  The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1
I, as a core Kubernetes contributor, want a e2e test framework that is well
organized and contains useful functions that will help me write clean and
maintainable e2e tests.

#### Story 2

I, as a maintainer of a project running on top of Kubernetes (e.g. external cloud providers, CSI drivers), want to use the
same infrastructure and tools used by core Kubernetes without having to import
core Kubernetes.

### Managing Internal Kubernetes Packages

To decouple the e2e test framework from internal packages we essentially have
to prevent any from `k8s.io/kubernetes/pkg` and from any other package in
`k8s.io/kubernetes/test` that aren't under
`k8s.io/kubernetes/test/e2e/framework`.

These core dependencies are introduced in the following ways
1. as primitives for test utility functions to manage Kubernetes resources and objects
2. as data structures to manage and consume data generated by a Kubernetes component
3. as variables and constants

To resolve the dependencies introduced by bullet point 1, we plan to move or
rewrite e2e test framework functions using official packages and APIs to match
the current desired behaviour.
For the most part this should be straightforward and it will only cause issues
when one of the other two following conditions are present.

Bullet point number 2 is present in cases where e2e tests involve collecting
metrics from unpublished APIs.
As an example, see
[kubeletstatsv1alpha1](https://github.com/kubernetes/kubernetes/blob/ba3bf32300574d86c98657981c96ca609de787a2/test/e2e/framework/resource_usage_gatherer.go#L37)
being imported to collect Kubelet statistics.
We want to prevent this kind of internal dependencies in order to ensure that
all artifacts needed for e2e testing are properly surfaced to users.
To deal with these cases we will collaborate with the owning SIGs to have the
necessary internal APIs used in e2e tests published in
https://github.com/kubernetes/api.

Bullet point 3 will be dealt in a similar manner by working with owning SIGs to
export any constants and variables defined in core Kubernetes that are required for e2e tests.

### Managing Test Packages

Decoupling the e2e test framework from other test packages will proceed in a
similar manner.

We will take a look at the utility functions surfaced by the e2e test framework
and we will see how and why they are used in e2e tests.
If we encounter utilities that are used sparingly in only a single test package
then we will opt to move it to that test package.

On the other hand, if we see a lot of dependencies and complexity introduced by
requiring other test packages then we will work to migrate the essential parts
within the e2e test framework.

Ultimately, we want a library that is well encapsulated and has useful methods
for maintainers to write e2e tests.


### Notes

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

#### Kubemark

One of the important e2e test components that we have been looking at is
[Kubemark](https://github.com/kubernetes/kubernetes/tree/72cdc8c2112cf14daeb8b4f29cb524c3aba890b4/test/kubemark)
which brings in a significant portion of dependencies from core Kubernetes.
Some of the use of internal unpublished APIs is believed to be related to this
and some other components built for scalability testing.

Given that Kubemark is not part of the e2e test framework, the current plan is
that no particular amount of work, other than the already mentioned, will be
needed to decouple the e2e test framework from Kubemark.

There is also the possibility that if Kubemark is the only reason why internal
Kubernetes APIs (such as the Kubelet stat API) are needed that we move the codes
that makes use of this to kubemark itself.



### Risks and Mitigations

<!--
What are the risks of this proposal and how do we mitigate.  Think broadly.
For example, consider both security and how this will impact the larger
kubernetes ecosystem.

How will security be reviewed and by whom?

How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.
-->

The work in this KEP involves one of the core components of e2e testing in the
Kubernetes community.
As with any kind of work, there is the risk that something will break.
And since the area we are focusing on is such a core part of CI, we plan to use
the vast set of e2e jobs and tests that already exist to monitor the health of our changes.

In particular, we will focus on
* https://testgrid.k8s.io/sig-release-master-blocking
* https://testgrid.k8s.io/sig-release-master-informing

These jobs are an integral part of ensuring that Kubernetes is working and in a good state to be used by people.
Monitoring these jobs will help us make sure that our changes are correct.


## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable.  This may include API specs (though not always
required) or even code snippets.  If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

As mentioned in the non-goals section, we want to keep the core of the e2e test
framework, the test context - which is sued for setting up and configurng e2e
tests - as it is.
What we do want to clean up and redesign are the multiple libraries and utility
functions used in e2e tests.

Our goal is to refactor the e2e test framework to the extent that any function
that is directly used in an e2e tests for managing an object or data (i.e.,
running a pod and checkingt that said pod meets a condition) is contained
within a e2e test framework sub-package that makes sense and is useful by
itself.

This will result in the import `"k8s.io/kubernetes/test/e2e/framework"` being
only used to configure a test or to pass information about the test environment
to e2e tests and framework sub-packages such as
`"k8s.io/kubernetes/test/e2e/framework/pod"` used to manage pods in e2e tests,
as an example.

In this context, decoupling the framework from core Kubernetes would entail
that e2e test framework sub-packages, such as
`"k8s.io/kubernetes/test/e2e/framework/pod"`, only use public APIs and test the
behaviours and workflows that users would expect.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.  Anything
that would count as tricky in the implementation and anything particularly
challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations).  Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

As stated in the [Risks and Mitigations](#risks-and-mitigations) section, we
will rely on the already existing e2e tests to ensure our work here is correct.
The e2e test framework also has some unit tests and we will work on extending
these.

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
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and
GA/stable, since there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

This enhancement is not a conventional KEP as it does not introduce any user
facing features, however, we plan to use the  conventional feature gate
nomenclature to organize our work into stages.

**Alpha**

During the alpha stage, we will focus on decoupling the e2e test framework
from core Kubernetes.
We will consider this stage as completed the moment we are able to use
import-boss to blanket disallow any core Kubernetes code from the e2e test framework.

This is the work we plan to do during the 1.19 release cycle.

**Beta**

For the beta stage we will resume our work refactoring the e2e test framework,
as stated in https://github.com/kubernetes/kubernetes/issues/76206.
This stage will be completed when the e2e test framework only contains code for
setting up and configuring e2e tests but no e2e test utility functions.
All utility functions used for e2e testing have to be contained in appropriate
e2e test framework subpackage.

We will also work on surfacing the coming move of the e2e test framework onto
staging to advise people that the e2e test framework is about to be pulled out
of core Kubernetes and published as its own package.
We will maintain core Kubernetes tests packages to reflect the work being done
here.

We plan to do so during the 1.20 release cycle.

**GA**

At this point, the e2e test framework should be completely decoupled from core
Kubernetes code and it should have been refactored to a point where we consumers
can expect a well designed API for e2e tests.
During this stage, we will move the e2e test framework from
`k8s.io/test/e2e/framework`
into the appropriate place in the staging directory as per
https://github.com/kubernetes/kubernetes/tree/master/staging.


<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable, can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/20190731-production-readiness-review-process.md

Production readiness review questionnaire must be completed for features in
v1.19 or later, but is non-blocking at this time. That is, approval is not
required in order to be in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.

-->

## Implementation History

<!--
Major milestones in the life cycle of a KEP should be tracked in this section.
Major milestones might include
- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

Current work for decoupling the e2e test framework from core Kubernetes is bein
tracked in this umbrella issue:
*  https://github.com/kubernetes/kubernetes/issues/74352

Work for cleaning up the e2e test framework can be found in this umbrella
issue:
* https://github.com/kubernetes/kubernetes/issues/76206

Umbrella issue for tracking work aimed at decoupling the core functionality of
the e2e test framework for e2e test configuration from utility functions can be
found here
* https://github.com/kubernetes/kubernetes/issues/81245

## Alternatives

<!--
What other approaches did you consider and why did you rule them out?  These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->


Another possibility is to create a new e2e test framework from scratch based on
all the current requirements and infrastructure used in the Kubernetes community.
However, this KEP is aimed at improving the current experience of Kubernetes maintainers.
Our main concern is with facilitating the work without introducing new processes
or requiring new infrastructure or components to be used.

Although, if a new e2e test framework was to later be created, the current e2e
test framework subpackages (where the bulk of the work for this KEP is going)
would still be useful as these deal with how to help execute e2e tests and not
don’t require or assume any particular e2e test configuration.
These e2e test framework sub-packages contain a general collection of utility
functions that rely on published packages (i.e., client-go, kubectl) and are not
specific to the e2e test framework, specially since they are being refactored
to be more general and not as tightly coupled to the configuration component
and logic of the core e2e test framework.
