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
# KEP-5000: Go-based Kubernetes API linting and CRD schema change validation tooling

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

This proposal introduces two new sig-api-machinery subprojects to improve the experience of developing Kubernetes APIs (including CRDs).

A linter, `kube-api-linter` (aka `kal`) as a golangci-lint plugin for evaluating Go types that are used to generate both built-in types and CRDs, to ensure they follow the Kubernetes API conventions and best practices.

A CRD schema upgrade checker, `crdify` as a CLI that compares generated CRD schemas to identify changes to the CRD schema, and ensure that any changes are compatible.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Aside from existing documentation on [Kubernetes API conventions][kube-api-conventions] and [making changes to Kubernetes APIs][making-changes-to-apis],
there is little to no tooling that enable developers to ensure they are following best practices when developing Kubernetes-native APIs.

This KEP aims to improve the Kubernetes-native API development experience by adding two sig-api-machinery subprojects for:

- Linting Kubernetes APIs written in Go based on the [Kubernetes API conventions][kube-api-conventions]
- Validating changes to CustomResourceDefinition schemas in YAML

[kube-api-conventions]:(https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
[making-changes-to-apis]:(https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md)

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Create tooling to help developers writing Kubernetes APIs in Go to follow best practices and Kubernetes API conventions
- Create tooling to help developers writing CustomResourceDefinitions avoid making breaking changes

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- Create tooling for validating CustomResourceDefinition schema changes at admission time

## Proposal

This KEP introduces two new sig-api-machinery subprojects.
The projects are intended to complement each other, and take lessons learned from [Kubernetes API conventions][kube-api-conventions] and our experience
with [making changes to Kubernetes APIs][making-changes-to-apis], to enable the wider Kubernetes ecosystem to follow best practices and avoid breaking changes.

### Go-based Kubernetes API Linting

A linter, built as a `golangci-lint` plugin, that evaluates Go type defintions (typically `_types.go`), and flags deviations from Kubernetes API conventions and best practices.

The linter will be configurable, allowing users to choose which of the rules they wish to adhere to, and which they wish to ignore.

The linter will focus, as much as possible on providing a "built-ins" experience by default, targetting rules for types in Kubernetes/Kubernetes.
It will also provide configuration to allow for a more "CRD" experience, adjusting the configuration and rules to cater to the needs of CRD authors.

The linter will focus on static evaluation of CRD types.
It cannot, and will not detect changes to the code, and therefore cannot assert any rule that requires information about a previous version of the types.

### CRD Schema Change Validation

Create a CLI tool that evaluates the differences between an old and new CRD in YAML to identify changes that may break users.

All validations are exported in Go packages that can be imported and consumed by other projects.

Some examples of how the CLI may be used:

- Comparing CRD that exists on cluster to a local YAML file of the same CRD - `crdify kube://mycrd.example.org file://mycrd.yaml`
- Comparing CRD across git revisions (could be used in CI systems) - `crdify git://main?path=mycrd.yaml git://HEAD?path=mycrd.yaml`
- Comparing CRD from git revision to a local file - `crdify git://main?path=mycrd.yaml file://mycrd.yaml`

The schema change validation will focus on changes that may break users, and changes that can be detected by comparing the CRD schemas in YAML.
It will not focus on static analysis, which will be the focus of the linter.

### Why do we need both?

When authoring types for Kubernetes, we typically use Go types, and add "markers" (e.g. `+optional`) to provide information about how the fields should behave.

When authoring custom types specifically, these markers are processed by [controller-gen][controller-gen] to generate the CRD schema in YAML.
This generation removes information, such as the type of the field, or whether a particular marker was present.

To implement some of the desired checks, for example ensuring that a required field is not a pointer, or that all fields are marked either `+optional` or `+required`, we need to evaluate the Go types themselves.

On the other hand, typical linter implementations do not provide the ability to compare two versions of a file, and identify changes that may break users.
Some CRD authors are also known to have used tools like `yaml-patch` to make changes to CRDs post generation.

Capturing breaking changes (such as tightening constraints on a field) requires comparing the CRD schemas old and new state, in the form of YAML.

These two types of checks create a divide or static and transitional analysis, and are best served by two separate tools.
The linter will focus on all checks that do not need to inspect transitions, and the CRD schema change validation will focus on all checks that do need to inspect transitions.

[controller-gen]: https://book.kubebuilder.io/reference/controller-gen

### User Stories

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

*As an Operator Author, I want to ensure my Go APIs that are used to build my CustomResourceDefinition(s) follow Kubernetes API conventions and best practices*

**How**: Using a golangci-lint plugin (or CLI) in both local development and CI environments to check for adherence to Kubernetes API conventions

#### Story 2

*As an API reviewer, I want to automate the process of finding deviations from Kubernetes API conventions and best practices so that I can focus on reviewing API changes on a more fundamental level*

**How**: Using a golangci-lint plugin (or CLI) in both local development and CI environments to check for adherence to Kubernetes API conventions

#### Story 3

*As an Operator Author, I want to ensure that changes to my CustomResourceDefinition(s) will not break users*

**How**: Using a CLI in both local development and CI environments to check for breaking changes in CRD schemas

#### Story 4

*As a Cluster Administrator, I want to validate that updating a CustomResourceDefinition won't break my cluster*

**How**: Using a CLI in both local development and CI environments to check for breaking changes in CRD schemas

#### Story 5

*As a Kubernetes Package Manager maintainer, I want to identify when making updates to CustomResourceDefinition(s) may result in a breaking change so that we can alert users before making those changes*

**How**: Using a library for identifying breaking changes in CRD schemas

#### Story 6

*As a Cluster Administrator, I want Kubernetes to prevent me from making breaking changes to CRDs on my cluster*

**How**: Using an admission controller/plugin that rejects CRD update operations if it would result in a breaking change

**Note for reviewers**: Adding an admission controller/plugin to enact this behavior is not part of this enhancement, but it is a use case that could be enabled by the introduction of the projects being proposed.

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

Any practices that are codified into tooling like this are likely to influence design of APIs across the Kubernetes ecosystem to include these practices. This includes any mistakes that make it into these tools.

To mitigate the risk of having a negative impact to the Kubernetes ecosystem, it is important to have each practice that is codified into these tools thoroughly reviewed and agreed upon by stakeholders prior to creating any releases that include them. Some important stakeholders to have involved in the process would be:
- Developers creating CRDs and Kubernetes-APIs backed by Go implementations
- Kubernetes Package Manager project maintainers (Helm, Operator Lifecycle Manager, etc.)
- Cluster administrators (where it makes sense)
- sig-api-machinery tech leads

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Go-based API Linting

There is already existing work done on this in https://github.com/JoelSpeed/kal - the intention is to continue with this approach, promoting this repository to a sig-apimachinery sponsored subproject.

#### As a golangci-lint plugin
`goalngci-lint` is a popular linter for Go code, already adopted within the Kubernetes ecosystem, that allows for the creation of custom linters.

By building linters based on the `go/analysis` package, and integrating them as a [plugin](https://golangci-lint.run/plugins/module-plugins/), `kal` integrates into the existing `golangci-lint` framework.

`golangci-lint` here provides several helpful features:* Exception handling - `golangci-lint` has a powerful exception handling configuration system that is already well established. By integrating as a plugin, `kal` users can leverage this system to ignore specific rules for specific files or directories.
* Integration with CI/CD - `golangci-lint` is already integrated into many CI/CD systems, and by creating a plugin, `kal` can be integrated into these systems as well.
* Release tooling - `golangci-lint custom` can be used to build a custom `golangci-lint` binary with the `kal` plugin included, making it easy to distribute and use.
* Output formatting - `golangci-lint` already has a well established pattern for printing results, and integrations into IDEs and other tools. `kal` can also leverage this as a plugin.
* Fixes - `golangci-lint` can automatically apply fixes if a linter supplies them. `kal` implements `SuggestedFixes` for many rules, enabling users to automatically fix issues.
* Diffing - `golangci-lint` is often used with the `--new-from-rev` flag that allows catching only new issues in a PR. `kal` can leverage this to ensure that new types are compliant with the rules, without needing to fix existing issues.

Some of the above may be considered nice to have, but the decision to leverage `golangci-lint` enables us to focus on implementing the rules, and not the actual linter tooling. 

#### Rules

A number of rules have been inspired by the [kube-api-conventions][kube-api-conventions], as well as sourced from issues and discussion with the Kubernetes community.

The following table details the rules that are identified as being useful, at the time of writing:

Currently implemented linters are:
| Name | Scope | Description | Configuration | Implemented |
|------|-------|-------------|---------------|-------------|
| `conditions` | All | Checks that `Conditions` fields are correctly formatted. Checks for listType and patchStrategyMarkers, as well as tags including protobuf | Protobuf and patch strategy checks can be disabled for CRDs | Yes |
| `commentstart` | All | Checks that all comments in the API types starts with the serialized form of the type they are commenting on | N/A | Yes |
| `integers` | All | Checks for usage of unsupported integer types, only `int32` and `int64` are allowed. No other `int` or `uint` variants. | N/A | Yes |
| `jsontags` | All | Checks that all fields in API types have a `json` tag and that the `json` tags are correctly formatted. Expecting camelCase. | Regex to match json tags against | Yes |
| `maxlength` | CRD (future All) | Checks that string and array fields in the API are bounded by a maximum length. Checks for `controller-gen` markers to identify limits. Prevents high CEL runtime cost. | N/A | Yes |
| `nobools` | All | Checks that fields in the API types do not contain a `bool` type. | N/A | Yes |
| `nofloats` | All | Checks that fields in the API types do not contain a `float32` or `float64` type. | (Future: Do not apply this rule to status fields) | Yes |
| `nophase` | All | Checks that fields in the API types don't contain a `Phase` or any field where `Phase` is a substring. | N/A | Yes |
| `optionalorrequired` | All | Checks that all fields in an API type are marked as either optional or required. Will replace `controller-gen` markers with their upstream equivalent | Preference over upstream or `controller-gen` markers | Yes |
| `requiredfields` | All | Checks that all fields in an API type marked as required follow the convention of not being pointers and not having `omitempty` in their JSON tag. | N/A | Yes |
| `statussubresource` | CRD | Checks that the status subresource is correctly configured correctly for API types that back a CRD. | N/A | Yes |
| `arrayofstruct` | All | Checks that any struct used as an array item contains at least 1 required field. | N/A | No |
| `defaults` | CRD | Checks that fields with default markers use the `+default` marker, and not the `controller-gen` equivalent. | Option to reverse the check for consistency across existing codebases. | No |
| `discriminatedUnions` | CRD (future All?) | Checks that `+union` structs have a `+unionDiscriminator` field, and checks members of the union for `+unionMember` markers. | Option to enforce CEL validation of discriminator to member mapping. Option to allow non-member fields within the union struct. | No |
| `duplicatemarkers` | All | Checks that fields in the API types do not contain duplicate markers. | N/A | No |
| `enums` | All | Enumerations should use a type alias and `+enum` marker. Enum values should be PascalCase | Pattern to match for Enum values, a list of exceptions to allow (e.g. for CLI tool names) | No |
| `nameformats` | CRD (future All) | Checks that common fields like `Name` and `Namespace` in API types are formatted correctly, using CEL and the format library. | N/A | No |
| `noduration` | All | Checks that fields in the API types do not contain a `time.Duration` type. Should be `fooSeconds` or `fooMinutes` etc. | N/A | No |
| `nomaps` | All | Checks that fields in the API types do not contain a `map` type with a struct value. Maps of subobjects are not allowed. | N/A | No |
| `numericbounds` | CRD (future All) | Checks that numeric fields in the API types are bounded by a maximum and minimum value. | N/A | No |
| `optionalfields` | All | Checks that all fields in an API type marked as optional follow the convention of being pointers and having `omitempty` in their JSON tag. | Option to allow non-pointers for non-struct types. | No |
| `references` | All | Checks that fields are named `Ref/Refs` and not `Reference/References`. | Option to forbid usage of either `Ref` or `Reference`. | No |
| `ssatags` | All | Checks that arrays have a `listType` marker | Option to forbid the use of `listType=set` for array of struct | No |
| `statusoptional` | All | Checks that all first level children of a `status` field are marked as optional. | N/A | No |
| `timestamps` | All | Checks that fields in the API types that are timestamps are named `FooTime` and not `FooTimestamp`. | N/A | No |
| `typeandobjectmeta` | All | Checks for top-level types (i.e. Kinds) and checks that they have inline typemeta and objectmata is optional/omitempty. | N/A | No |

A number of rules implemented are currently only applicable to CRDs, but in the future could be applicable to built-in types as well.
There is on-going research into marking built-ins and generating validation from these markers.
Once this research reaches its conclusion, these markers could be integrated into the rules that are at present, CRD only, to make them applicable to all types.

The unimplemented rules are being tracked by an [open issue](https://github.com/JoelSpeed/kal/issues/1) on the repository.
This enhancement is not aimed at discussion about individual rules, and comments about the rules may be left on their individual tracking issues.

### CRD Schema Change Validation

There is some previous work done in both https://github.com/everettraven/crd-diff and https://github.com/openshift/crd-schema-checker. Each project implements schema checks in different ways and has different user workflows.

It is proposed that a new repository is created that merges select bits and pieces from each project into a single project.

Currently, it is proposed that:
- The CLI UX from https://github.com/everettraven/crd-diff is used. It currently allows sourcing both the old and new CRDs from various locations, allowing for use case flexibility.
- The reporting logic from https://github.com/openshift/crd-schema-checker is used. It enforces that all checks include a description of why the check matters and allows for communicating errors, warnings, and other important information found during the comparison process.
- The CRD manifest schema walking logic from https://github.com/openshift/crd-schema-checker is used. https://github.com/everettraven/crd-diff imports this logic.
- The validation/validator pattern from https://github.com/everettraven/crd-diff. It clearly distinguishes between individual property validations and broader scoped CRD validations. Additionally, it has a validation pattern in place that checks for general CRD changes, compatibility of version-to-version, and compatibility of served versions.

https://github.com/everettraven/crd-diff currently has some configuration logic in place, but it is highly repetitive and not a great user experience. This needs to be evaluated further and will likely result in a distinct configuration process for the new project.

Additionally, the following validations from both projects are proposed to be included in the move to the common project:
- **Enum** (crd-diff): Validates compatibility of changes to enum constraints on a property. Flags net new enum constraint on existing property, adding an enum, and removing an enum.
- **Default** (crd-diff): Validates compatibility of changes to default property values. Flags adding, removing, or changing of a default value.
- **Max\*** (crd-diff): Validates compatibility of changes to property constraints related to max value constraints (maximum, maxItems, maxLength, maxProperties). Flags adding net new max* constraints and decreasing existing max* constraints.
- **Min\*** (crd-diff): Validates compatibility of changes to property constraints related to min value constraints (minimum, minItems, minLength, minProperties). Flags adding net new min* constraints and increasing existing min* constraints.
- **Required** (crd-diff): Validates compatibility of required fields. Flags adding new required properties.
- **Type** (crd-diff): Flags changes in property type.
- **Scope** (crd-diff): Flags changes in CRD scope.
- **ExistingFieldRemoval** (crd-diff, crd-schema-checker): Flags removal of existing fields.
- **StoredVersionRemoval** (crd-diff): Flags removal of a stored version. Only applicable when comparing to a CRD that is already present on a cluster.

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

Each subproject will have it's own set of unit tests.

For `kube-api-linter`, this will be unit tests for each linter.

For `crdify`, this will be unit tests for each schema change validation.

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

n/a

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
    
The `kube-api-linter` project should not need to follow a version skew strategy as it is operating on Go code itself and not interacting with Kubernetes components.
    
The `crdify` project should follow the `kubectl` version skew strategy.

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
    
These are not on-cluster components or features. No feature enablement/rollback necessary.

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

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

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->
    
The proposed projects are not on-cluster components

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

The proposed projects are not on-cluster components
    
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
    
Using the `crdify` CLI tool may result in GET requests being made to the Kubernetes API server if the `kube` source type is used.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
    
No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
    
No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
    
No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
    
No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->
    
No

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

The `crdify` tool will fail to get an existing CRD from a cluster if the API server or etcd are unavailable. This would only be the case when using the `kube://` sourcing method.

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

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

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

Instead of making `kube-api-linter` a golangci-lint linter, creating a standalone CLI. This was ruled out because of the wide adoption of the golangci-lint tooling making it easier for users to adopt in their existing linting workflows.
    
## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

Two new GitHub Repositories:
- kubernetes-sigs/kube-api-linter (or kubernetes-sigs/kal)
- kubernetes-sigs/crdify
