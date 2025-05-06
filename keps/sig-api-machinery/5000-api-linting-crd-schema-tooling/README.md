# KEP-5000: Go-based Kubernetes API linting and CRD schema change validation tooling

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Go-based Kubernetes API Linting](#go-based-kubernetes-api-linting)
  - [CRD Schema Change Validation](#crd-schema-change-validation)
  - [Why do we need both?](#why-do-we-need-both)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
    - [Story 4](#story-4)
    - [Story 5](#story-5)
    - [Story 6](#story-6)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Go-based API Linting](#go-based-api-linting)
    - [As a golangci-lint plugin](#as-a-golangci-lint-plugin)
    - [Rules](#rules)
  - [CRD Schema Change Validation](#crd-schema-change-validation-1)
    - [Validation configuration](#validation-configuration)
    - [Optional Admission Controller/Plugin](#optional-admission-controllerplugin)
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

This proposal introduces two new sig-api-machinery subprojects to improve the experience of developing Kubernetes APIs (including CRDs).

A linter, `kube-api-linter` as a golangci-lint plugin for evaluating Go types that are used to generate both built-in types and CRDs, to ensure they follow the Kubernetes API conventions and best practices.

A CRD schema upgrade checker, `crdify` as a CLI that compares generated CRD schemas to identify changes to the CRD schema, and ensure that any changes are compatible.

## Motivation

Aside from existing documentation on [Kubernetes API conventions][kube-api-conventions] and [making changes to Kubernetes APIs][making-changes-to-apis],
there is little to no tooling that enable developers to ensure they are following best practices when developing Kubernetes-native APIs.

This KEP aims to improve the Kubernetes-native API development experience by adding two sig-api-machinery subprojects for:

- Linting Kubernetes APIs written in Go based on the [Kubernetes API conventions][kube-api-conventions]
- Validating changes to CustomResourceDefinition schemas

[kube-api-conventions]:(https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
[making-changes-to-apis]:(https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md)

### Goals

- Create tooling to help developers writing Kubernetes APIs in Go to follow best practices and Kubernetes API conventions
- Create tooling to help developers writing CustomResourceDefinitions avoid making breaking changes
- Create tooling for validating CustomResourceDefinition schema changes at admission time

### Non-Goals

## Proposal

This KEP introduces two new sig-api-machinery subprojects.
The projects are intended to complement each other, and take lessons learned from [Kubernetes API conventions][kube-api-conventions] and our experience
with [making changes to Kubernetes APIs][making-changes-to-apis], to enable the wider Kubernetes ecosystem to follow best practices and avoid breaking changes.

### Go-based Kubernetes API Linting

A linter, built as a `golangci-lint` plugin, that evaluates Go type defintions (typically `_types.go`), and flags deviations from Kubernetes API conventions and best practices.

The linter will be configurable, allowing users to choose which of the rules they wish to adhere to, and which they wish to ignore.

The linter will focus, as much as possible on providing a "built-ins" experience by default, targeting rules for types in Kubernetes/Kubernetes.
It will also provide configuration to allow for a more "CRD" experience, adjusting the configuration and rules to cater to the needs of CRD authors.
For the use cases that require further customization, it will be possible to vendor the linter code and extend it
to produce your own customized linter that is compatible with golangci-lint.

The linter will focus on static evaluation of CRD types.
It cannot, and will not detect changes to the code, and therefore cannot assert any rule that requires information about a previous version of the types.

### CRD Schema Change Validation

Create a CLI tool and optional admission plugin that evaluates the differences between an old and new CRD to identify changes that may break users.

All validations are exported in Go packages that can be imported and consumed by other projects.
Other projects will be able to vendor and extend or customize the validation set to their use case.

Some examples of how the CLI may be used:

- Comparing CRD that exists on cluster to a local YAML file of the same CRD - `crdify kube://mycrd.example.org file://mycrd.yaml`
- Comparing CRD across git revisions (could be used in CI systems) - `crdify git://main?path=mycrd.yaml git://HEAD?path=mycrd.yaml`
- Comparing CRD from git revision to a local YAML file - `crdify git://main?path=mycrd.yaml file://mycrd.yaml`

The schema change validation will focus on changes that may break users, and changes that can be detected by comparing the CRD schemas.
It will not focus on static analysis, which will be the focus of the linter.

Additionally, there will be an optional way to run an admission plugin that can signal to Cluster Administrators
when a CRD author has made breaking changes. This can be used to signal when a migration plan may need to be created to successfully update the CRD.

### Why do we need both?

When authoring types for Kubernetes, we typically use Go types, and add "markers" (e.g. `+optional`) to provide information about how the fields should behave.

When authoring custom types specifically, these markers are processed by [controller-gen][controller-gen] to generate the CRD schema in YAML.
This generation removes information, such as the type of the field, or whether a particular marker was present.

To implement some of the desired checks, for example ensuring that a required field is not a pointer, or that all fields are marked either `+optional` or `+required`, we need to evaluate the Go types themselves.

On the other hand, typical linter implementations do not provide the ability to compare two versions of a file, and identify changes that may break users.
Some CRD authors are also known to have used tools like `yaml-patch` to make changes to CRDs post generation.

Capturing breaking changes (such as tightening constraints on a field) requires comparing the CRD schemas old and new state.

These two types of checks create a divide or static and transitional analysis, and are best served by two separate tools.
The linter will focus on all checks that do not need to inspect transitions, and the CRD schema change validation will focus on all checks that do need to inspect transitions.

[controller-gen]: https://book.kubebuilder.io/reference/controller-gen

### User Stories

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

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

Any practices that are codified into tooling like this are likely to influence design of APIs across the Kubernetes ecosystem to include these practices. This includes any mistakes that make it into these tools.

To mitigate the risk of having a negative impact to the Kubernetes ecosystem, it is important to have each practice that is codified into these tools thoroughly reviewed and agreed upon by stakeholders prior to creating any releases that include them. Some important stakeholders to have involved in the process would be:
- Developers creating CRDs and Kubernetes-APIs backed by Go implementations
- Kubernetes Package Manager project maintainers (Operator Lifecycle Manager, Carvel etc.)
- Cluster administrators (where it makes sense)
- sig-api-machinery tech leads
- Kubernetes API Reviewers

## Design Details

### Go-based API Linting

There is already existing work done on this in https://github.com/JoelSpeed/kal - the intention is to continue with this approach, promoting this repository to a sig-apimachinery sponsored subproject.

#### As a golangci-lint plugin
`goalngci-lint` is a popular linter for Go code, already adopted within the Kubernetes ecosystem, that allows for the creation of custom linters.

By building linters based on the `go/analysis` package, and integrating them as a [plugin](https://golangci-lint.run/plugins/module-plugins/), `kube-api-linter` integrates into the existing `golangci-lint` framework.

`golangci-lint` here provides several helpful features:* Exception handling - `golangci-lint` has a powerful exception handling configuration system that is already well established. By integrating as a plugin, `kube-api-linter` users can leverage this system to ignore specific rules for specific files or directories.
* Integration with CI/CD - `golangci-lint` is already integrated into many CI/CD systems, and by creating a plugin, `kube-api-linter` can be integrated into these systems as well.
* Release tooling - `golangci-lint custom` can be used to build a custom `golangci-lint` binary with the `kube-api-linter` plugin included, making it easy to distribute and use.
* Output formatting - `golangci-lint` already has a well established pattern for printing results, and integrations into IDEs and other tools. `kube-api-linter` can also leverage this as a plugin.
* Fixes - `golangci-lint` can automatically apply fixes if a linter supplies them. `kube-api-linter` implements `SuggestedFixes` for many rules, enabling users to automatically fix issues.
* Diffing - `golangci-lint` is often used with the `--new-from-rev` flag that allows catching only new issues in a PR. `kube-api-linter` can leverage this to ensure that new types are compliant with the rules, without needing to fix existing issues.

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
- The validation/validator pattern from https://github.com/everettraven/crd-diff is used. It clearly distinguishes between individual property validations and broader scoped CRD validations. Additionally, it has a validation pattern in place that checks for general CRD changes, compatibility of version-to-version, and compatibility of served versions.

The tables below show any existing and proposed validations for each of the projects and whether or not they will be carried over to the new common project. Any validations that behave the same in
both projects marked as being carried over will be combined into a single validation.

Validations in https://github.com/everettraven/crd-diff, both existing and proposed:

| Name | Scope | Description | Implemented | Carry |
| ---- | ----- | ----------- | ----------- | ----- |
| enum | Property | Validates compatibility of changes to enum constraints on a property. Net new enum constraints, adding and removing enum values are flagged. | Yes | Yes |
| default | Property | Validates compatibility of changes to default values on a property. Adding, removing, or changing of a default value are flagged. | Yes  | Yes |
| maximum | Property | Validates compatibility of changes to maximum constraints on a property. Net new maximum constraints or a decrease in maximum value are flagged | Yes | Yes |
| maxitems | Property | Validates compatibility of changes to maxitems constraints on a property. Net new maxitems constraints or a decrease in maxitems value are flagged | Yes | Yes |
| maxlength | Property | Validates compatibility of changes to maxlength constraints on a property. Net new maxlength constraints or a decrease in maxlength value are flagged | Yes | Yes |
| maxproperties | Property | Validates compatibility of changes to maxproperties constraints on a property. Net new maxproperties constraints or a decrease in maxproperties value are flagged | Yes | Yes |
| minimum | Property | Validates compatibility of changes to minimum constraints on a property. Net new minimum constraints or an increase in minimum value are flagged | Yes | Yes |
| minitems | Property | Validates compatibility of changes to minitems constraints on a property. Net new minitems constraints or an increase in minitems value are flagged | Yes | Yes |
| minlength | Property | Validates compatibility of changes to minlength constraints on a property. Net new minlength constraints or an increase in minlength value are flagged | Yes | Yes |
| minproperties | Property | Validates compatibility of changes to minproperties constraints on a property. Net new minproperties constraints or an increase in minproperties value are flagged | Yes | Yes |
| required | Property | Validates compatibility of changes to required constraints on a property. Adding new required properties are flagged | Yes | Yes | 
| type | Property | Validates compatibility of changes to type constraints on a property. Changes to property types are flagged | Yes | Yes |
| scope | CRD | Validates that scope doesn't change for the CRD | Yes | Yes |
| existingfieldremoval | CRD | Validates that existing fields are not removed from the CRD | Yes | Yes |
| storedversionremoval | CRD | Validates that stored versions are not removed from the CRD. Only valid for CRDs that are previously present on a cluster | Yes | Yes |
| format | Property | Validates compatibility of changes to the format of a property. Changes to property formats are flagged | No | Yes |
| exclusiveMaximum | Property | Validates compatibility of changes to the exclusiveMaximum of a property. Net new exclusiveMaximum constraints or a decrease in exclusiveMaximum are flagged | No | Yes |
| exclusiveMinimum | Property | Validates compatibility of changes to the exclusiveMinimum of a property. Net new exclusiveMinimum constraints or an increase in exclusiveMinimum are flagged | No | Yes |
| uniqueItems | Property | Validates compatibility of changes to the uniqueItems of a property. Net new uniqueItems constraints are flagged | No | Yes | 
| pattern | Property | Validates compatibility of changes to the pattern constraint of a property. Net new pattern constraints and more restrictive validations are flagged | No | Yes |
| nullable | Property  | Validates compatibility of changes to the nullable constraint of a property. Making a property no longer nullable is flagged | No | Yes |
| multipleOf | Property | Validates compatibility of changes to the multipleOf constraint of a property. Changes to the multipleOf constraint are flagged | No | Yes |
| allOf | Property | Validates compatibility of changes to the allOf constraint of a property. Net new and addition of allOf constraints are flagged | No | Yes |
| oneOf | Property | Validates compatibility of changes to the oneOf constraint of a property. Net new and removal of oneOf constraints are flagged | No | Yes |
| anyOf | Property | Validates compatibility of changes to the anyOf constraint of a property. Net new and removal of anyOf constraints are flagged | No | Yes |
| not | Property | Validates compatibility of changes to the not constraint of a property. Net new not constraints are flagged | No | Yes |

Proposed validations are tracked via https://github.com/everettraven/crd-diff/issues/3.
All validations have some form of configuration option. For more information on the configuration options that exist for each validation, see https://everettraven.github.io/crd-diff/#/validations
This existing configuration approach is not ideal. It is highly repetitive and allows for configuration options that may not actually make sense. We should be more opinionated
than this existing approach is. A new approach for configuration is proposed further down in this section.

Validations in https://github.com/openshift/crd-schema-checker  both existing and proposed:

| Name | Scope | Description | Implemented | Carry |
| ---- | ----- | ----------- | ----------- | ----- |
| NoBools | Property | Validates that CRD properties are not of type `boolean` | Yes | No, already implemented as a linter in `kube-api-linter` |
| NoFloats | Property | Validates that CRD properties are not of type `number` | Yes | No, already implemented as a linter in `kube-api-linter` |
| NoUints | Property | Validates that CRD properties are not of type `uint` | Yes | No, already implemented as a linter in `kube-api-linter` |
| NoFieldRemoval | CRD | Validates that existing fields are not removed from a CRD | Yes | Yes |
| NoEnumRemoval | Property | Validates that existing enum values are not removed from a property | Yes | Yes |
| NoMaps | Property | Validates that CRD properties of type `object` do not have an `additionalProperties` field specified | Yes | No, if desired this is better suited for a linter in `kube-api-linter` |
| NoDataTypeChange | Property | Validates that property type is not changed | Yes | Yes |
| MustHaveStatus | CRD | Validates that the CRD has a status subresource | Yes | No, this is already implemented as a linter in `kube-api-linter` |
| ListsMustHaveSSATags | Property | Validates that lists have the `x-kubernetes-list-type` tag for server side apply | Yes | No, this is desired as a linter in `kube-api-linter` |
| ConditionsMustHaveSSATags | Property | Validates that status conditions fields have the appropriate tags for server side apply | Yes | No, this is already implemented as a subset of the `conditions` linter in `kube-api-linter` |
| NoNewRequiredFields | CRD | Validates that no new required fields are added | Yes | Yes |
| MustNotExceedCostBudget | Property | Validates that `XValidations` don't exceed the CEL cost budget | Yes | No, if desired this is better suited for a linter in `kube-api-linter` |

#### Validation configuration

The proposed approach for configuration of individual validations when using as a CLI is to use a configuration file.
Projects that import validations in their Go code will have the ability to configure them through the Go API directly.

The configuration file will be a YAML file with the following format:
```yaml
apiVersion: crdify.sigs.k8s.io/v1alpha1
kind: ValidationConfiguration
validations:
- name: <validationName>
  enforcement: Error || Warn || None
  config: # map[string]interface{}
    configField: configValue
```

This pattern is inspired from the Kubernetes API server admission plugin configuration pattern.

Enforcement mode `Error` means that any incompatible changes the validation detects will result in an error being output to the terminal and the program exiting with a non-zero exit code.
Enforcement mode `Warn` means that any incompatible changes the validation detects will result in a warning being output to the terminal and the program exiting with an exit-code of 0.
Enforcement mode `None` means that the validation will not be run.

Not all validations will have an arbitrary set of extra configuration options, but all validations that do _must_ have a default configuration.
If a known validation is not specified in the configuration, the default for that validation will be used.

The only way to enable, disable, and configure validations will be via this configuration file to encourage users to explicitly define an acceptable risk profile.

#### Optional Admission Controller/Plugin
There is existing work done that allows for running the CLI produced by https://github.com/openshift/crd-schema-checker
as an admission plugin to validate CRD compatibility at admission time.

The existing work will be moved to the new project and updated to consume any changes necessary based on the proposed
merging of the two existing validation tools.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

Each subproject will have it's own set of unit tests.

For `kube-api-linter`, this will be unit tests for each linter.

For `crdify`, this will be unit tests for each schema change validation. The optional admission plugin will share the same tests as `crdify`.

##### Integration tests

Each subproject will be responsible for implementing their own integration tests.

##### e2e tests

Each subproject will be responsible for implementing their own e2e tests.

### Graduation Criteria

### Upgrade / Downgrade Strategy

n/a

### Version Skew Strategy

The `kube-api-linter` project should not need to follow a version skew strategy as it is operating on Go code itself and not interacting with Kubernetes components.

The `crdify` project should follow the `kubectl` version skew strategy.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

These are not on-cluster components or features. No feature enablement/rollback necessary.

###### How can this feature be enabled / disabled in a live cluster?

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

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

### Rollout, Upgrade and Rollback Planning

The proposed projects are not on-cluster components

###### How can a rollout or rollback fail? Can it impact already running workloads?

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

### Monitoring Requirements

The proposed projects are not on-cluster components

###### How can an operator determine if the feature is in use by workloads?

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

Using the `crdify` CLI tool may result in GET requests being made to the Kubernetes API server if the `kube` source type is used.

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

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The `crdify` tool will fail to get an existing CRD from a cluster if the API server or etcd are unavailable. This would only be the case when using the `kube://` sourcing method.

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

## Drawbacks

## Alternatives

Instead of making `kube-api-linter` a golangci-lint linter, creating a standalone CLI. This was ruled out because of the wide adoption of the golangci-lint tooling making it easier for users to adopt in their existing linting workflows.

## Infrastructure Needed (Optional)

Two new GitHub Repositories:

- kubernetes-sigs/kube-api-linter
- kubernetes-sigs/crdify
