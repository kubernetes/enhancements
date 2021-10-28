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
# KEP-2985: Public KRM Functions Registry

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
  - [Publisher](#publisher)
  - [Security](#security)
  - [Trust](#trust)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
    - [Story 4](#story-4)
    - [Story 5](#story-5)
    - [Story 6](#story-6)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Repo Location](#repo-location)
  - [Management Model](#management-model)
    - [Centralized Index and Release Management](#centralized-index-and-release-management)
    - [Prior Art](#prior-art)
    - [Pros and Cons](#pros-and-cons)
  - [Centralized Index with Distributed Release Management](#centralized-index-with-distributed-release-management)
    - [Prior Art](#prior-art-1)
    - [Pros and Cons](#pros-and-cons-1)
  - [Mixture Model](#mixture-model)
  - [Website](#website)
  - [Function Metadata](#function-metadata)
  - [Publishing Workflow](#publishing-workflow)
  - [Repo Layout Convention](#repo-layout-convention)
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

This KEP proposes to create a public [KRM functions] registry for the community
to contribute and discover useful KRM functions.
There will be a repo to host the centralized index for KRM functions and a
website that present the KRM functions to the users.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

[KRM functions] have gained more and more interests and become more and more
popular in the k8s configuration management space.

Google has a [GitHub repository](https://github.com/GoogleContainerTools/kpt-functions-catalog)
under GoogleContainerTools for hosting the source of the functions and a
[website](https://catalog.kpt.dev/) for presenting the functions to the end
users. However, not everyone in the community can contribute to it for various
reasons. E.g. a company’s policy may not allow its employees to contribute to
repo owned by another company.

To have a thriving ecosystem of KRM functions, we must enable contributions for
all community members in vendor-neutral registry. With a public KRM function
registry, we can significantly improve the Day 0 and Day 2 user experiences. On
Day 0, users can discover some KRM functions that may be useful for their needs.
On Day 2, users can query the registry to discover new function versions.

[KRM functions]: https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

* Enable end users of orchestrators (e.g. Kustomize, Kpt) that support KRM
  functions to discover and leverage a common ecosystem of compatible functions.
* Enable end users to discover and use sets of functions specifically from 
  publishers they trust.
* Enable function authors from any company to expose their function in a
  well-known index for discovery by end users.
* Provide a central place for first-party (SIG-sponsored) plugins to be built
  and added to the index.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

* Replace or compete with Catalog as publication format for collections of
  functions. Catalog should be used by this Registry, and the details of its
  internal format should be discussed in that KEP.
* Support building non-SIG-sponsored functions.
* Support SIG-sponsored functions written in a language other than Go or
  published in a format other than containerized.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

This KEP proposes to create a public KRM functions registry. There will be 2
components to back the registry:

- A repo sponsored by sig-cli to host the index of the functions in the registry.
- A website that presents the doc and examples to the end users and allows users
  to search and discover KRM functions.


### Publisher

When publishing a function, the contributor MUST publish it on behalf of a
publisher. We can revisit this decision later if there are requests to relax it
when publisher is an individual.

A publisher can be one of the following:

- A project, community or SIG in Kubernetes: e.g. Kustomize, KubeFlow or SIG-CLI.
- A company: e.g. Apple or Google.
- A GitHub organization: e.g. github.com/myorg
- An individual: e.g. foo@example.com

All publishers must specify maintainers in a OWNERS file which is a convention
in kubernetes. Whenever changes (e.g. adding new functions) are made in a
publisher, at least one of the maintainers must approve the change.

To publish on behalf of a company:

* The commit must be a [verified commit](https://docs.github.com/en/github/authenticating-to-github/managing-commit-signature-verification)
  from a person with that company’s email address.
* If there are maintainers for this publisher, at least one of the maintainers
  must approve the change.

To publish on behalf of a GitHub organization, the contributor must be a member
of the organization.

To publish as an individual, the commit must be a verified commit from the same
person's email.

Kubernetes already has the tooling to enforce approval with OWNERS files. We can
leverage it.
CI can be set up to enforce verifiable commit from desired email domain.

### Security

SIG-CLI is responsible for the security of the SIG-sponsored KRM functions but
not all KRM functions in the registry.

Publishers are responsible for the security of their KRM functions. Publishers
are responsible for clearly communicating the expectation (e.g. maturity) to
their users. For example, Kustomize can provide a small set of carefully vetted
KRM functions which can be published as _kustomize_.

We strongly suggest users to use container as a sandboxing mechanism to run the
KRM functions.

### Trust

A user should NOT trust every KRM function in the registry.

Trust can be established at the publisher level. Users can choose to trust a
publisher and use the KRM functions provided by this publisher.

Publisher information can be used to aggregate KRM functions. We can support
both dynamic aggregation of KRM functions and static, versioned collection of
KRM functions. Publishers can choose to create a snapshot of the dynamic
aggregation of their KRM functions at some time. The snapshot must be versioned,
but SemVer is not necessary here since it's meaningless for a catalog. The
snapshot can be accessed later as a static catalog.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a KRM functions user, I can browse the function registry website (e.g.
https://krm-functions.io) and search the KRM function by name (e.g. set-labels).
And I can find everything including doc, examples, homepage, maintainers,
publisher information about the function.

#### Story 2

As a KRM function user, I can query
https://krm-functions.io/catalogs/aggregate/latest.yaml?publisher=kubeflow,kustomize
to find a real-time aggregation of all KRM functions published by Kustomize and
KubeFlow.

#### Story 3

<<[UNRESOLVED]>>
This KEP propose using the date as the version of static catalog. Alternatively,
the hash of the contents can be used as the version. This is still TBD.
<<[/UNRESOLVED]>>

As a KRM function user, I can find the versioned catalog published by Pineapple
Co. at
https://krm-functions.io/catalogs/pineapple/v20210924.yaml.
It is a catalog provided by Pineapple Co. and snapshoted on 09/24/2021.

#### Story 4

As a kustomize user, I want to use a KRM functions catalog provided by a
publisher in kustomize.
The `kustomiztion.yaml` file may look like the following per [Catalog KEP](https://github.com/kubernetes/enhancements/pull/2954/files#diff-da31478d2b3a925c17471e989c953539b508eed0aae19d624ad943d08f4dd910R414-R415)

```
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
catalogs:
- https://krm-functions.io/catalogs/kustomize/v20210924.yaml.
resources:
- input.yaml
transformers:
- ...
```

#### Story 5

As a KRM function user, I want to have tab completion for function image names
when using imperative runs. The sugguested image names should from the registry.

When using with kustomize:
```shell
kustomize fn run --image <tab><tab>
```

When using with kpt:
```shell
kpt fn eval --image <tab><tab>
```

#### Story 6

As a Kustomize maintainer, I want to develop and publish a small, well-vetted
set of functions published from the SIG's registry. These plugins should behave
identically to built-ins from the end-user perspective.

Furthermore, I can release a version of Kustomize that trusts these functions.
This should be supported by kustomize, but building kustomize is not in scope
for this KEP.

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

Security might be a potential risk here. We have introduced the publisher
concept to mitigate the risk.

The dynamic generation of catalogs (e.g. from a versionless URL with a query
param), if supported and used in declarative configs, would lead to
non-reproducible builds.
To mitigate it, we would strongly suggest users to use versioned catalogs from
the registry in production.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Repo Location

The ideal repo location is in [kubernetes-sigs](https://github.com/kubernetes-sigs).
It should be sponsored by SIG-CLI.

The repo name can be `krm-functions`, `krm-function-registry` or something more
reasonable.

### Management Model

#### Centralized Index and Release Management

The source code, documentation, releases and function metadata are completely
managed in one repo. A website can be built from the repo.

#### Prior Art

[Kpt-functions-catalog](https://github.com/GoogleContainerTools/kpt-functions-catalog)
has been using this management model. Everything about the functions are managed
in the [Kpt-functions-catalog](https://github.com/GoogleContainerTools/kpt-functions-catalog)
repo and the [website](https://catalog.kpt.dev/) is rendered from the
information in the repo.

#### Pros and Cons

Pros:

* Easier to manage.
* Quality (e.g. test coverage) of the code can be enforced, since the
  maintainers can do the gatekeeping.

Cons:

* May have scalability issues: repo maintainers may become the bottleneck.
* The release pipeline (at least for containers) is required.
* Supporting releasing binaries (exec mode in kustomize) will be challenging,
  since there are many programming languages and tools to build binaries, and
  it’s almost impossible to meet everyone’s needs.

### Centralized Index with Distributed Release Management

The source code, documentation and releases are managed in repositories that are
owned by the original function authors. The function metadata is managed in the
central index repository owned by CNCF.

#### Prior Art

Similar model is used by [Krew Index](https://github.com/kubernetes-sigs/krew-index)
(kubectl’s plugin index) and [Terraform Registry](https://registry.terraform.io/)
(a public registry for Terraform modules).

In Krew Index, only the manifest files (plugin metadata) are published to the
[krew-index](https://github.com/kubernetes-sigs/krew-index) repo.
[Here](https://github.com/kubernetes-sigs/krew-index/blob/master/plugins/access-matrix.yaml)
is an example plugin manifest file.

In [Terraform Registry](https://registry.terraform.io/), when contributors want
to publish a module, they will need to allow Terraform Registry to register a
webhook in their public GitHub repository to detect git tags as releases.

#### Pros and Cons

Pros:

* Scale well. Maintainers only need to review function metadata files when
  contributors publish their functions.
* Encourage the KRM functions community to grow larger and faster.
* It’s possible to support binaries as KRM functions in the registry, since they
  are built and released by the publishers.

Cons:

* Quality (e.g. test coverage) of the code is hard to be enforced.
* Security may be a concern if we want to support binaries as KRM functions in
  the registry.

### Mixture Model

We can mix the 2 management models above.

We can manage the source code of the Kustomize provided KRM functions in-tree.
Generic KRM functions like set-labels, set-annotations and set-namespace can be
included. All the KRM functions provided by kustomize must go through a security
audit. These in-tree KRM functions can serve as examples for other publishers
about how to organize their functions.

The model of centralized index with distributed release is more flexible and
more suitable for all vendors and other contributors. We can manage the source
code of the functions contributed by the community out-of-tree.

All KRM functions in the registry must provide the metadata in the repo. We must
standardize the metadata format for KRM functions, since we will require all
contributors to follow it. A website can be built using the metadata information.

### Website

The website code will live in the registry as well.
Ideally, we don't need to check generated html files in the repo. We can use
tools to generate the site from Markdown files.

Kpt site is using [docsify](https://docsify.js.org/#/) and kubebuilder site is
using [mdBook](https://github.com/rust-lang/mdBook).

### Function Metadata

<<[UNRESOLVED]>>
The schema of the function metadata has not been finalized yet.
<<[/UNRESOLVED]>>

The following is an example function metadata for a container-based KRM
function. We will only support container-based KRM functions in the public
registry.

The content under field `spec` will be used directly in a Catalog resource.

```yaml
apiVersion: config.k8s.io/v1alpha1
kind: KRMFunction
spec:
  group: example.com
  kind: SetNamespace
  description: "A short description of the KRM function"
  publisher: example.com
  versions:
    - name: v1
      schema:
        openAPIV3Schema: ... # inline schema like CRD
      idempotent: true|false
      runtime:
        container:
          image: docker.example.co/functions/set-namespace:v1.2.3
          sha256: a428de... # The digest of the image which can be verified against. This field is required if the version is semver.
          requireNetwork: true|false
          requireStorageMount: true|false
      configMap: true|false # Support ConfigMap as functionConfig. Default is false if omitted.
      usage: <a URL pointing to a README.md>
      examples:
        - <a URL pointing to a README.md>
        - <another URL pointing to another README.md>
      license: Apache 2.0
    - name: v1beta1
      ...
  maintainers: # The maintainers for this function. It doesn't need to be the same as the publisher OWNERS. 
    - foo@example.com
  tags: # keywords of the KRM functions
    - mutator
    - namespace
```

The following is an example for exec-based KRM function. We will not allow
contributors to publish exec-based KRM functions. But we want to standardize the
metadata to allow an organization to share exec-based KRM functions internally.

```yaml
apiVersion: config.k8s.io/v1alpha1
kind: KRMFunction
spec:
  group: example.com
  kind: SetNamespace
  description: "A short description of the KRM function"
  publisher: example.com
  versions:
    - name: v1
      schema:
        openAPIV3Schema: ...
      idempotent: true|false
      runtime:
        exec:
          platforms:
          - bin: foo-amd64-linux
            os: linux
            arch: amd64
            uri: https://example.com/foo-amd64-linux.tar.gz
            sha256: <hash>
          - bin: foo-amd64-darwin
            os: darwin
            arch: amd64
            uri: https://example.com/foo-amd64-darwin.tar.gz
            sha256: <hash>
      configMap: true|false # Support ConfigMap as functionConfig. Default is false if omitted.
      usage: <a URL pointing to a README.md>
      examples:
        - <a URL pointing to a README.md>
        - <another URL pointing to another README.md>
      license: Apache 2.0
    - name: v1
      ...
  home: <a URL pointing to the home page>
  maintainers: # The maintainers for this function. It doesn't need to be the same as the publisher OWNERS. 
    - foo@example.com
  tags: # keywords of the KRM functions
    - mutator
    - namespace
```

### Publishing Workflow

We only support publishing container-based KRM function in the public registry.
We will only cover the workflow for that.

The developer needs to do the following:

1. Build a container image and pushed it to a publicly accessible container
   registry.
2. Ensure the usage doc is a markdown file and is up-to-date and publicly
   accessible.
3. Create a file called `krm-function-metadata.yaml` which contains the metadata
   that satisfies the KRM function metadata schema above.
4. Checkout the KRM function registry repo.
5. Move the `krm-function-metadata.yaml` file to the desired location in the
   repo by following the repo layout convention (discussed below).
6. Depending on the requirements (discussed in an earlier session) of different
   publisher type, choose the right email to commit the change.
7. Create a PR and get it reviewed and approved by the publisher OWNERS.

### Repo Layout Convention

```shell
├── publishers
│   ├── communities
│   │   ├── kustomize
│   │   │   ├── fn-foo
│   │   │   │   └── krm-function-metadata.yaml
│   │   │   ├── fn-bar
│   │   │   └── OWNERS # OWNERS of the publisher
│   │   ├── kubeflow
│   │   ├── sig-cli
│   │   └── OWNERS # OWNERS to approve new community publishers
│   ├── companies
│   │   ├── apple
│   │   │   ├── fn-baz
│   │   │   └── OWNERS # OWNERS of the publisher
│   │   ├── google
│   │   └── OWNERS # OWNERS to approve new company publishers
│   ├── github-orgs
│   └── individuals
├── krm-functions # in-tree functions implementation
│   ├── kustomize
│   │   ├── fn-foo
│   │   └── OWNERS # OWNERS to approve code change to the function
│   └── sig-cli
├── site # Stuff related to the site
└── OWNERS
```

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

For the sig-sponsored KRM functions, they should be tested in-tree. And if we
develop a test harness, it should live in-tree. If kustomize has an existing
test harness, we can leverage it or move it to the registry repo.

For KRM functions that are not sig-sponsored, the maintainers are responsible
for testing them.

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

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

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

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

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

[KRM functions spec]: https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md