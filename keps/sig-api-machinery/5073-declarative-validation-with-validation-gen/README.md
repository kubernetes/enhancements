# KEP-5073: Declarative Validation of Kubernetes Native Types With validation-gen
<!-- toc -->
- [KEP-5073: Declarative Validation of Kubernetes Native Types With validation-gen](#kep-5073-declarative-validation-of-kubernetes-native-types-with-validation-gen)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [Overview](#overview)
    - [Introduce <code>validation-gen</code>](#introduce-validation-gen)
      - [<code>validation-gen</code> Has No Plans To Use CEL Validation Directly](#validation-gen-has-no-plans-to-use-cel-validation-directly)
      - [IDL Tag Authoring DevEx and User Error Messaging](#idl-tag-authoring-devex-and-user-error-messaging)
    - [Introduce new validation tests and test framework](#introduce-new-validation-tests-and-test-framework)
      - [New Validations Vs Migrating Validations](#new-validations-vs-migrating-validations)
      - [New Validation Tests](#new-validation-tests)
      - [Ensuring Validation Equivalence With Testing](#ensuring-validation-equivalence-with-testing)
    - [Introduce Feature Gate: <code>DeclarativeValidation</code>](#introduce-feature-gate-declarativevalidation)
      - [<code>DeclarativeValidation</code> Will Target Beta From The Beginning](#declarativevalidation-will-target-beta-from-the-beginning)
    - [Linter](#linter)
    - [Documentation Generation](#documentation-generation)
  - [Analysis of existing validation rules](#analysis-of-existing-validation-rules)
    - [User Stories (Optional)](#user-stories-optional)
      - [Kubernetes developer wishes to add a field to an existing API version](#kubernetes-developer-wishes-to-add-a-field-to-an-existing-api-version)
      - [Kubernetes developer adds a new version (v1beta2) of an API](#kubernetes-developer-adds-a-new-version-v1beta2-of-an-api)
      - [Kubernetes Developer Using an Aggregated API and/or KRM Server Is Adding a New Field To Their Custom  API Type](#kubernetes-developer-using-an-aggregated-api-andor-krm-server-is-adding-a-new-field-to-their-custom--api-type)
      - [Kubernetes API reviewer is reviewing API changes for a PR for a new Kubernetes Native Type](#kubernetes-api-reviewer-is-reviewing-api-changes-for-a-pr-for-a-new-kubernetes-native-type)
    - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Risks and Mitigations](#risks-and-mitigations)
      - [Risk: The Migration Project Loses Steam And Work Is Abandoned](#risk-the-migration-project-loses-steam-and-work-is-abandoned)
        - [Mitigation: Roll Back All Migrated Fields](#mitigation-roll-back-all-migrated-fields)
      - [Risk: we get hundreds of PRs from people migrating fields and can't review them all.](#risk-we-get-hundreds-of-prs-from-people-migrating-fields-and-cant-review-them-all)
        - [Mitigation: These are not urgent and we will have patterns which can be reviewed by more people.](#mitigation-these-are-not-urgent-and-we-will-have-patterns-which-can-be-reviewed-by-more-people)
      - [Risk: Versioned validation drifts between versions.](#risk-versioned-validation-drifts-between-versions)
        - [Mitigation: round-trip testing + fuzzing + equivalence tests.](#mitigation-round-trip-testing--fuzzing--equivalence-tests)
      - [Risk: Migration to Declarative Validation introduces breaking change to API validation](#risk-migration-to-declarative-validation-introduces-breaking-change-to-api-validation)
        - [Mitigation: Ensure Invalid Objects Still Invalid](#mitigation-ensure-invalid-objects-still-invalid)
        - [Mitigation: Ensure Valid Old Objects Still Valid](#mitigation-ensure-valid-old-objects-still-valid)
      - [Risk: Added latency to API request handling.](#risk-added-latency-to-api-request-handling)
        - [Mitigation: Avoid Conversion to Internal Type](#mitigation-avoid-conversion-to-internal-type)
  - [Design Details](#design-details)
    - [Summary of Declarative Validation Components](#summary-of-declarative-validation-components)
    - [<code>validation-gen</code> Implementation Plan](#validation-gen-implementation-plan)
    - [Catalog of Supported Validation Rules &amp; Associated IDL Tags](#catalog-of-supported-validation-rules--associated-idl-tags)
    - [Supporting Declarative Validation IDL tags On Shared Struct Fields](#supporting-declarative-validation-idl-tags-on-shared-struct-fields)
      - [<code>subfield</code> IDL Tag](#subfield-idl-tag)
      - [<code>validation-gen</code> One-deep typedef support](#validation-gen-one-deep-typedef-support)
        - [Proposed Solutions](#proposed-solutions)
    - [Migration Plan](#migration-plan)
      - [Phase1: Initialization (Responsibility of Contributors Implementing the KEP)](#phase1-initialization-responsibility-of-contributors-implementing-the-kep)
      - [Phase2: Scaling the Migration (Responsibility of Contributors Implementing the KEP and broader community)](#phase2-scaling-the-migration-responsibility-of-contributors-implementing-the-kep-and-broader-community)
      - [Phase3: Finalization and GA (Core Team and community)](#phase3-finalization-and-ga-core-team-and-community)
    - [Tagging and Validating Against Internal vs Versioned Types](#tagging-and-validating-against-internal-vs-versioned-types)
    - [Handling Zero Values in Declarative Validation](#handling-zero-values-in-declarative-validation)
      - [Difficulties with <code>+required</code> and <code>+default</code>](#difficulties-with-required-and-default)
      - [Proposed Solutions](#proposed-solutions-1)
      - [Addressing the Problem with Valid Zero Values Using the Linter](#addressing-the-problem-with-valid-zero-values-using-the-linter)
    - [Ratcheting](#ratcheting)
    - [Test Plan](#test-plan)
        - [Prerequisite testing updates](#prerequisite-testing-updates)
        - [Unit tests](#unit-tests)
        - [Integration tests](#integration-tests)
        - [e2e tests](#e2e-tests)
    - [Graduation Criteria](#graduation-criteria)
      - [Beta](#beta)
        - [DeclarativeValidation](#declarativevalidation)
      - [GA](#ga)
        - [DeclarativeValidation](#declarativevalidation-1)
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
    - [Use CEL and OpenAPI libraries directly for K8s Native Types (<a href="https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/4153-declarative-validation">KEP-4153</a>)](#use-cel-and-openapi-libraries-directly-for-k8s-native-types-kep-4153)
  - [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

# KEP-5073: Declarative Validation of Kubernetes Native Types With validation-gen

## Summary

With this enhancement, Kubernetes Developers will declare validation rules using Interface Definition Language (IDL) tags in the `types.go` files that define the Kubernetes native API types. For example:

```go
// staging/src/k8s.io/api/core/v1/types.go

// ReplicationControllerSpec is the specification of a replication controller.
type ReplicationControllerSpec struct {
        // +k8s:optional
        // +k8s:minimum=0
        Replicas *int32 `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`

        // +k8s:optional
        // +k8s:minimum=0
        MinReadySeconds int32 `json:"minReadySeconds,omitempty" protobuf:"varint,4,opt,name=minReadySeconds"`
...
}
```

In this example `+k8s:optional`, and `+k8s:minimum` are IDL tags.

The declarative validation IDL tags will be used to generate code via a new code generator - `validation-gen`.  For the above IDL tags the generated code will look something like the snippet below (full file [here](https://gist.github.com/aaron-prindle/39302ba5685d072d02ce0bdb25c2c76c))

```go
func Validate_ReplicationControllerSpec(opCtx operation.Context, obj, oldObj *corev1.ReplicationControllerSpec, fldPath *field.Path) (errs field.ErrorList) {
        // field corev1.ReplicationControllerSpec.Replicas
        errs = append(errs,
                func(obj, oldObj *int32, fldPath *field.Path) (errs field.ErrorList) {
                        if e := validate.RequiredPointer(opCtx, fldPath, obj, oldObj); len(e) != 0 {
                                errs = append(errs, e...)
                                return // do not proceed
                        }
                        errs = append(errs, validate.Minimum(opCtx, fldPath, obj, oldObj, 0)...)
                        return
                }(obj.Replicas, safe.Field(oldObj, func(oldObj  *corev1.ReplicationControllerSpec) *int32 { return oldObj.Replicas }), fldPath.Child("replicas"))...)
        // field corev1.ReplicationControllerSpec.MinReadySeconds
        errs = append(errs,
                func(obj, oldObj *int32, fldPath *field.Path) (errs field.ErrorList) {
                        errs = append(errs, validate.Minimum(opCtx, fldPath, obj, oldObj, 0)...)
                        return
                }(&obj.MinReadySeconds, safe.Field(oldObj, func(oldObj *corev1.ReplicationControllerSpec) *int32 { return &oldObj.MinReadySeconds }), fldPath.Child("minReadySeconds"))...)
        return errs
}
```

This generated code will then be used in the kube-apiserver to validate API requests.

## Motivation

Kubernetes API validation rules are currently written by hand, which makes them difficult for users to access directly, review, maintain, and test.

Declarative validation will benefit Kubernetes maintainers:

*   It will make it easier to develop, maintain and review APIs.
*   It will make it easier programmatically inspect and analyze the API, enabling new tools and improved documentation.
*   It will enable improvements to the API machinery. For example, a feature like ratcheting validation will become more tractable to implement because the feature can be implemented once in the declarative validation subsystem rather than piecemeal across the 15k lines of hand written validation code.

Declarative validation will also benefit Kubernetes users:

*   Adding new fields and associated validation rules becomes a simpler process of adding IDL tags to the field definition, rather than writing and maintaining validation functions. This reduces potential inconsistencies and bugs.  Testing is also streamlined as centralized rules and frameworks enable the use of test fixtures across k8s types making creating tests simpler and faster to implement for contributors.
*   Creating k8s APIs becomes faster as developers can focus on the API’s structure and behavior, leaving the validation boilerplate to be generated automatically. This speeds up development and encourages experimentation with new APIs.
*   Validation is performed on versioned types (assuming recommended approach is used from design below), so error message and attributed field path is more likely to be relevant to the user. (ex: some fields in autoscaling/v2 will be mis-identified with current approach but resolved w/ Declarative Validation + versioned types)

Additionally Declarative Validation can be built upon to support features such as:

*   Ability to give users direct access to the actual API validation rules (via publishing OpenAPI rules for native types), which are currently only available to developers willing and able to find and read the hand written validation rules.
*   Ability to enable clients to perform validation of native types earlier in development workflows ("shift-left"), such as with a Git pre-submit linter.
*   Ability to improve API composition. In particular CRDs that embed native types (such as PodTemplate), which gain validation of the native type automatically. This has the potential to simplify controller development and improve end user experiences when using CRDs.
*   (Eventually) bring CRD validation and built-in validation together with the same set of server-side validations.

Please feel free to try out the [prototype](https://github.com/jpbetz/kubernetes/tree/validation-gen) to get hands-on experience with this proposed enhancement.

### Goals

*   Eliminate 90% of net-new hand-written validation within 5 kube releases (target start: v1.33)
*   Convert 50% of existing hand-written validation within 5 kube releases (target start: v1.33)
*   Migrate in such a way that if contributors lose steam and abandon this, we can roll it back relatively easily.
*   `types.go` files become the de-facto IDL of Kubernetes for native types. It is worth noting that `+enum` support, `+default` support and similar enhancements all moved our API development forward in this direction. This enhancement is an attempt to continue that story arc.
*   API Validations are readable and manageable. The IDL should be a joy to use for API authors, reviewers and maintainers. Common complex relationships have simple-to-express idioms in the IDL.
    *   Reduce API development costs for API authors and reviewers. Reduce long term maintainer costs. Reclaiming time from core project contributors.
    *   Reduce risk of incorrect validation rules reaching production via API development approach that is easier to get right and harder to get wrong.
*   No change to error message structure for clients. Difference to clients is limited to minimal changes to error detail strings.  The goal is to preserve the `v1.Status` validation error message’s `field` and `reason`, etc. but we reserve the ability to change the `details` which are allowed to be modified over time from k8s guarantees.
*   Enable development of linters and other API tool chains that use API validation rules and metadata. Further reducing development effort and risk of incorrect validation rules.
*   Retain native (or nearly native) performance.
*   Improve testing rigor by being vastly easier to test.

### Non-Goals

*   Eliminate 100% of existing hand-written validation.  The goal is to drastically reduce the # of hand-written validation cases only to those that are extra-ordinary but not replace all hand-written validation.
*   Eliminate 100% of net-new hand-written validation
*   To convert validation into a full-featured scripted language
*   It is not a goal of this KEP directly to publish validation rules to OpenAPI, that is the goal of a TBD complementary KEP
*   It is not a goal of this KEP directly to have declarative defaults, only declarative validation rules.  That is the goal of a TBD complementary KEP

## Proposal

### Overview

*   Introduce `validation-gen`
    *   Generates code that is very similar to hand written code (de-risks performance)
    *   Introduce a +tags system that is pluggable and extensible
        *   easy to write custom validators in go (de-risks migration problems that could be caused by a long tail of complex validation rules)
        *   allows fine grained opt-in generation: per group/type/+tag enablement (de-risks adoption)
        *   Can run declarative validation AND hand written generation code during migration (de-risk adoption)
*   Introduce new validation tests, test framework and migration test utilities
    * No field can go thru migration without a robust test for the field in question and maintainer review scrutiny which proves that it is validated correctly before the change and after.
    *   Create migration test pattern and utilities which support testing equivalence between hand-written validation and declarative validation (de-risks migration problems)
*   Introduce featuregate: `DeclarativeValidation`
    *   Safety mechanism in case a mistake is made so that users can turn off Declarative Validation and get back to a healthy validation state.  (de-risks migration problems)
*   Migration
    *   Migrate schema from one type of a core API group to prove the viability of the approach, in a single PR.
        *   PR will leverage `validation-gen` test framework to demonstrate 100% validation equivalence across hand-written and declarative validation for migrated field(s).
    *   For migrating core API groups the work will be:
        *   Highly incremental.
        *   Easy to review and test.
        *   Easy to distribute.
            *    Leveraging the broader community to speedup migration is key to the success of the project.
*   Build linters, documentation-generators, and other tools to make k8s development and API review easier and safer
    *   Example:  Lint rule that verifies that all required/optional field information is correct

### Introduce `validation-gen `

`validation-gen` will parse structured comments (IDL tags) within Kubernetes API type definitions (types.go files) and generate corresponding Go validation functions. `validation-gen` will be built as an extensible framework allowing new "Validators" to be added by describing what IDL tags they parse, the constraints on the IDL tags (for UX error messaging), the format of the IDL tag + how it is used (for documentation), and what actual validation logic will be for the generated code given the tagged field and associated args. The generators validators will be registered with the scheme in a similar way to generated conversion and defaulting.

#### `validation-gen` Has No Plans To Use CEL Validation Directly

The previous Declarative Validation proposal ([KEP-4153](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/4153-declarative-validation)) proposed using CEL for a number of the complex validations present in the current Kubernetes validation logic for native types (cross-field, transition, etc.).  The `validation-gen` solution presented here uses go code directly for the validations which means we do not plan on evaluating CEL server side as we can write arbitrary go code to perform server side validations.  This allows the `validation-gen` solution to be highly flexible and performant.  If we reach a point where CEL makes sense, we can evaluate it at that time.

#### IDL Tag Authoring DevEx and User Error Messaging

In order to properly support the User Stories for “Kubernetes developer wishes to add a field to an existing API version” and “Kubernetes developer adds a new version an API” it is important that `validation-gen` and the associated tooling for IDL tags have robust UX that immediately notifies users when tags are not used properly.  To support this `validation-gen` will have options for validators subject to when they register so that validation authors can express how their associated IDL tag should be used and the framework will error if a user uses an IDL tag incorrectly.  See this related WIP PR [here](https://github.com/jpbetz/kubernetes/pull/89) adding such functionality to the prototype for an example of what this might look like.

The goal is that when a user makes a mistake in authoring IDL tags we give a meaningful error.  Users are not expected to know the underlying system or be an insider on the project to successfully use Declarative Validation.

### Introduce new validation tests and test framework

New validation tests will be created as part of the migration process for contributors migrating fields using `validation-gen`.  Additionally, a test framework and associated migration test utilities will be created as part of `validation-gen` to leverage the new centralized validation rules and to ensure validation consistency across hand-written and declarative validation rules.  See the [Test Plan](?tab=t.0#bookmark=id.a86sekfgbuen) section for more details.

#### New Validations Vs Migrating Validations
FIXME...    

#### New Validation Tests
As part of the process for migrating fields, contributors will scrutinize and improve as needed the current validation tests.  No field can go thru migration without a robust test for the field being migrated, which proves that it is validated correctly before the change and after. Many existing tests are not sufficient to verify 100% equivalency and need retooling.  This allows us to de-risk migration problems by scrutinizing the current tests and enhancing them.

#### Ensuring Validation Equivalence With Testing

For testing the migration and ensuring that the validation is identical across current hand-written validation and declarative validations, an equivalence test will be added to all migrated fields, schemas, etc. in the respective validation_test.go.  This will verify that the the outputs for validation_test.go are identical across enabling and disabling the featuregate - `DeclarativeValidation`.

Verifying that a field/type that is migrated is appropriately tested with proper changes to validation_test.go, equivalence testing, etc. will be human-drivern enforced in PR review for the related community migration PR

### Introduce Feature Gate: `DeclarativeValidation`

A new feature gate - `DeclarativeValidation` will be created as part of Declarative Validation.  This feature gate will toggle the behaviour of routing the validation for specified number of fields for a schema (and later full schemas) to use declarative validation vs the hand-written validation.  In Beta this will enable a partial migration where initially a small number of fields of a single schema  will be ported to declarative validation and this will increase over time until all validation logic is declarative.  During the Beta lifecycle more and more validations will be ported until they all are in which case the feature will move to GA.  For more information see the “Migration Plan” section.

#### `DeclarativeValidation` Will Target Beta From The Beginning

Declarative Validation will target the Beta stage from the beginning (vs Alpha).  Additionally DeclarativeValidation is targeting Beta with default:true.  This is because Declarative Validation is not new functionality but an alternative implementation of validation and users should not be able to perceive any changes when swapping hand-written validation w/ identical declarative validation.  The feature gate, `DeclarativeValidation`, exists as a safety mechanism  in case a mistake is made so that users can turn it off and get back to safety.  There is prior art for this rationale where other feature gates did not target Alpha as they were not related to new functionality (changing underlying behaviour, bugfix, etc.).  An example of this is the current feature gate `AllowParsingUserUIDFromCertAuth` which was introduced in Beta as `default:true` as it is not a net new feature but fixes a current issue ([PR](https://github.com/kubernetes/kubernetes/pull/127897), [feature gate](https://github.com/kubernetes/kubernetes/blob/master/pkg/features/versioned_kube_features.go#L228-L230))

### Linter

As we transition from handwritten validation to a declarative approach using validation tags, a linter becomes essential to ensure the correct and consistent use of these tags. It will maintain the integrity of the validation process during and after the migration. It will also enforce the rules around using validation tags, preventing common mistakes (e.g. see Tri-state mutual exclusive option for handling zero values) and ensuring that the generated validation logic behaves as intended.

We will integrate the linter functionality directly into `validation-gen` and control it using a command-line flag, `--lint. `When `--lint `is added the command won’t generate code, but only prints out the linting result. The `--lint` would run in the presubmit to block the code change with invalid tags. Some pros of this method include:

*   **Leveraging Existing Infrastructure:** The linter can leverage the existing code in `validation-gen` for parsing Go source files, extracting validation tags, and traversing the type tree.
*   **Integration with Code Generation:** The linter can be integrated into the code generation workflow, ensuring that validation tags are checked before code is generated. For example, we could add a `--lint` flag.
*   **CI Integration:** It's easy to integrate the linter into CI pipelines as part of the build and test process. We could run `validation-gen` with `--lint` flag and fail the build if any linting errors are found.

### Documentation Generation

By having all validators, associated IDL tags, their descriptions, etc.  defined in code it is possible to automatically generate the documentation necessary for IDL tags which k8s developers can reference. This allows for: 

*   Publishing documentation on all tags including how they work, their intended usage, examples, etc.
*   Building a system to auto-gen docs from this

## Analysis of existing validation rules

At the time of writing this document, there are ~1181 validation rules written in about 15k lines of go code in [kubernetes/kubernetes/pkg/apis](https://github.com/kubernetes/kubernetes/commit/0c62b122c02bff9131b6db960042150a3638d3f3).

~15% of the ~1181 validation rules are `forbidden` rules which primarily check which fields are allowed when a union discriminator is set, ~10% are object name validations, and ~10% are cross field validation checks, mainly mutual exclusion and some "transition rules" (e.g. immutability checks).  The remaining 65% of validation rules can be represented using JSON Schema value validations. `optional`, `format` and `enum` will be the most frequently used.  

Based on this analysis we believe that the proposed `validation-gen` design can meet our final goals of 50% of the current hand-written validation logic via IDL tags with declarative validation replacement logic within the next five kube-releases and 90% of net-new validations within the next five kube-releases.  

![Validation types](validation-types.svg)
![Object metadata name validations](metadata-name-types.svg)

### User Stories (Optional)

#### Kubernetes developer wishes to add a field to an existing API version

1. Developer adds the field to the Go struct
2. Developer adds needed validation IDL tags to the Go struct
    *   In the case the developer incorrectly uses an IDL tag, `validation-gen` promptly gives a detailed error message identifying to the user which tag needs to be fixed.
3. Developer adds validation_test.go cases (same as today)
4. API reviewers review IDL tags along with Go struct change

#### Kubernetes developer adds a new version (v1beta2) of an API

1. Developer copies over v1beta1 API and creates v1beta2
2. Linter verifies that IDL tags match for both version of API (unless exceptions are put in exception file for the linter)
3. API reviewer can review change knowing that validation is consistent

#### Kubernetes Developer Using an Aggregated API and/or KRM Server Is Adding a New Field To Their Custom  API Type

1. Developer defines validation rules declaratively on the field using the same IDL tags (+k8s:minimum, +k8s:format, etc.) as Kubernetes native types directly in their types.go files.
2. Developer leverages the published validation-gen code generator from Kubernetes to automatically generate updated Go validation functions for their custom API type.
	
#### Kubernetes API reviewer is reviewing API changes for a PR for a new Kubernetes Native Type

1. Reviewer directly examines validation IDL tags on the APIs in the new types.go file (next to the API definitions).
    *    Reviewer grasps validation intent of each field as IDL tags provide a concise and declarative way to understand the validation rules at a glance
3. CI linter job automatically verifies the consistency and proper usage of the specified IDL tags (eg: field doesn’t have both +required and +optional, etc.)
4. Reviewer reviews PR focusing on the overall API design and ensuring the PR’s validation rules effectively support the API's intended behavior.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

#### Risk: The Migration Project Loses Steam And Work Is Abandoned

##### Mitigation: Roll Back All Migrated Fields

In the case that the workstream loses steam over time and the migration project is abandoned, users can still FIXME...

#### Risk: we get hundreds of PRs from people migrating fields and can't review them all. 

##### Mitigation: These are not urgent and we will have patterns which can be reviewed by more people.

FIXME...

#### Risk: Versioned validation drifts between versions.

##### Mitigation: round-trip testing + fuzzing + equivalence tests.

FIXME...

#### Risk: Migration to Declarative Validation introduces breaking change to API validation

##### Mitigation: Ensure Invalid Objects Still Invalid

Objects which previously did not pass validation must still not pass validation. This can be restated simply as: all currently handwritten validations must be included in the schema. If all validations are in the schema, it follows that the schema will reject the same set of objects as before.

We can mitigate this by:

1. All Validation Tests Have 100% Coverage
2. All Validation Tests Succeed against native and schema-based validation backends

To accomplish this, we will implement the test framework and test utilities mentioned in "Test Plan”

##### Mitigation: Ensure Valid Old Objects Still Valid

Objects which previously passed validation must still pass validation. For our purposes this can be thought of as ensuring the schema-based validation does not raise new errors which previously did not exist.

It is not possible to be 100% sure of this property, in fact, there may be new validations added to patch bugs in existing ones. But we can mitigate this risk:

0. PREREQUISITE - No field can go thru migration without a robust test for the field in question, which proves that it is validated correctly before the change and after. 

Many existing validation tests do not account for all required valdiation cases.  It is possible that if there is unintentionally a change some small aspect of it (making validation for an existing field weaker or stronger) there could be errors in the migration. The only real solution is to scrutinize and retool tests.

1. Catch and fix any significant differences in schema validations

To help catch these errors faster, we will also refactor all e2e tests to enable this feature and check for validation error differences.

2. Minimize the impact of any new changes introduced

As mentioned in the section - “Ratcheting” `validation-gen` is capable of supporting any desired ratcheting behaviour.  Validation Ratcheting would reduce impact to users by allowing any new schema validations errors due to this feature to be ignored for unchanged portions of an object.

#### Risk: Added latency to API request handling.

**NOTE:** The validation path has NEVER been a target for efficiency optimizations, it is likely there is currently low-hanging fruit with respect to optimizing.  Currently the code does such things as deep-copies, use reflect, build maps of 3 items, etc.  

The design decision to declaratively validate API versions has performance implications.

Today, the apiserver's request handler decodes incoming versioned API requests to the versioned type and then immediately performs an "unsafe" conversion to the internal type when possible. All subsequent request processing uses the internal type as the "hub" type. Validation is written for the internal type and so are per-resource strategies.

Assuming we use the recommended plan of declarative validation operating on versioned types, the internal type will no longer be responsible for validation.

If we were to convert from the internal back to the version of the API request for validation, we would introduce one additional conversion. If we make this an "unsafe" conversion, then it will be low cost for the vast majority of requests.

We will benchmark this approach and plan to use it for beta.

Long term, we could do better:

Since the internal type will no longer be used for validation, it becomes a lot less important. It is still important to have a hub type. But why not pick one of the versioned types to be the hub type? The vast majority of APIs only have one version anyway. The obvious candidate version to choose for the hub version would be the preferred storage version.

Switching to a versioned type for the hub type would have a few implications:

*   We would eliminate the need for internal versions.
*   We would introduce more conversion when API request version differs from the hub version. But beta APIs are off-by-default and we expect a lot less mixed version API usage than in the past.
*   Would require rewriting in tree admission plugins.

This "hub version" change feels like something that could be made somewhat independent of this KEP.

##### Mitigation: Avoid Conversion to Internal Type

Requests are received as the versioned type, so it should be feasible to avoid extra conversions for resources that have no need of handwritten validations.

## Design Details

### Summary of Declarative Validation Components

*   validation-gen - new code generator which parses declarative validation IDL tags and outputs validation go code
    *   Validators
    *   Test fixture
    *   Linter
    *   Documentation generator
*   Feature gate - `DeclarativeValidation`
*   Testing
    *   Equivalency tests (verifyVersionedValidationEquivalence in prototype)
        *   validation_test.go
    *   Fuzz Testing (TestVersionedValidationByFuzzing in prototype)

### `validation-gen` Implementation Plan

validation-gen will be a code generator (similar to defaulter-gen, conversion-gen, protobuf-gen, etc.) and integrated into the current k8s code generation framework similarly.  It can be invoked similar to other code generators (see command below) and will be plumbed through similar to other generators:
```bash
$ hack/update-codegen.sh validation
```

Validators are plugged into the `validation-gen` framework (new).  To implement a `Validator` (like the `enumValidator` below), users implement a Validator interface which in the prototype consists of:

*   Init
*   TagName
*   ValidScopes
*   GetValidations
*   Docs

Below is a snippet from the enumValidator detailing the core logic of `GetValidations `and registration:
```go
var enumValidator = types.Name{Package: validationPkg, Name: "Enum"}

func (etv *enumTagValidator) GetValidations(context Context, _ []string, payload string) (Validations, error) {
        if context.Type != types.String {
                return Validations{}, fmt.Errorf("can only be used on string types")
        }

        var result Validations

        if enum, ok := etv.enumContext.EnumType(context.Parent); ok {
                supportVarName := PrivateVar{Name: "SymbolsFor" + context.Parent.Name.Name, Package: "local"}
                supportVar := Variable(supportVarName, GenericFunction(enumTagName, DefaultFlags, setsNew, []types.Name{enum.Name}, enum.ValueArgs()...))
                result.AddVariable(supportVar)
                fn := Function("enum", DefaultFlags, enumValidator, supportVarName)
                result.AddFunction(fn)
        }

        return result, nil
}

func init() {
    AddToRegistry(EnumValidator) // Registers +enum
}
```

validation-gen processes types.go files, searching for tags. For example:

```go
type DeploymentStrategy struct {
    Type DeploymentStrategyType
    …
}
// +k8s:enum // <---- THIS IS A validation-gen TAG
type DeploymentStrategyType string
const (
    RecreateDeploymentStrategyType DeploymentStrategyType = "Recreate"
    RollingUpdateDeploymentStrategyType DeploymentStrategyType = "RollingUpdate"
)
```

When a tag is found by validation-gen.  validation-gen uses the plugins to generate the appropriate code:

```go
// generated
var symbolsForDeploymentStrategy = sets.New[E1](v1.Recreate, v1.RollingUpdate)

func Validate_DeploymentStrategy(in *v1.DeploymentStrategy, 
        fldPath *field.Path) (errs field.ErrorList) {
    errs = append(errs, validation.ValidateEnum(
        fldPath.Child("type"), in.Type, symbolsForDeploymentStrategy)...)
    return errs
}
```

The generator will also auto register all validations in the runtime.Scheme.

Once validation is generated, it will be easy to opt-in (see below snippet).

**NOTE:** This does not actually do anything until the tags are used on types/fields, and all tags are net-new. None of the existing tags cause any validation code to be generated as designed by having our own bespoke tags (eg: `+k8s:required`(new w/ `validation-gen`) vs `+required`(old), etc.)

```go
// +k8s:validation-gen=TypeMeta
// +k8s:validation-gen-input=k8s.io/api/apps/v1
```

### Catalog of Supported Validation Rules & Associated IDL Tags
A number of the rules in the below sections are not implemented but will be trivial to implement once we are are aligned on the right pattern/syntax for the given validator. Implementing a validator “just-in-time” means we don't add more without real fields using them. We estimate that in the limit we may have 30-40 validators, but today we have less than 10.  

The below rules are currently implemented or are very similar to an existing validator in the [validation-gen prototype](prototype )

<table>
  <tr>
   <td>
    <strong>Type of validation</strong>
   </td>
   <td>
    <strong>IDL tag</strong>
   </td>
   <td>
    <strong>Relative OpenAPI validation field</strong>
   </td>
  </tr>
  <tr>
   <td>
    string format
   </td>
   <td>
    `+k8s:format={format name}`
   </td>
   <td>
    `format`
   </td>
  </tr>
  <tr>
   <td>
    size limits
   </td>
   <td>
    `+k8s:min{Length,Items}`, `+k8s:max{Length,Items}`
   </td>
   <td>
    `min{Length,Items}`, `max{Length,Items}`
   </td>
  </tr>
  <tr>
   <td>
    numeric limits
   </td>
   <td>
    `+k8s:minimum`, `+k8s:maximum`, `+k8s:exclusiveMinimum`, `+k8s:exclusiveMaximum`
   </td>
   <td>
    `minimum`, `maximum`, `exclusiveMinimum`, `exclusiveMaximum`
   </td>
  </tr>
  <tr>
   <td>
    required fields
   </td>
   <td>
    `+k8s:optional`
<p>
    `+k8s:required`
   </td>
   <td>
    `required`
   </td>
  </tr>
  <tr>
   <td>
    enum values
   </td>
   <td>
    `+k8s:enum`
   </td>
   <td>
    `enum`
   </td>
  </tr>
  <tr>
   <td style="background-color: null">
    Union values
   </td>
   <td style="background-color: null">
    `+k8s:unionMember` \
`+k8s:unionDiscriminator`
   </td>
   <td style="background-color: null">
    `oneOf,anyOf,allOf`
   </td>
  </tr>
  <tr>
   <td style="background-color: null">
    forbidden values
   </td>
   <td style="background-color: null">
    `+k8s:forbidden`
   </td>
   <td style="background-color: #null">
   </td>
  </tr>
  <tr>
   <td style="background-color: null">
    feature gate is enabled
   </td>
   <td style="background-color: null">
    `+k8s:ifOptionEnabled(FeatureX)=&lt;if-enabled-validator-tag>`
   </td>
   <td style="background-color: null">
    N/A
   </td>
  </tr>
  <tr>
   <td style="background-color: null">
    feature gate is disabled
   </td>
   <td style="background-color: null">
    `+k8s:ifOptionDisabled(FeatureX)=&lt;if-disabled-validator-tag>`
   </td>
   <td style="background-color: null">
    N/A
   </td>
  </tr>
  <tr>
   <td style="background-color: null">
    validate each key
   </td>
   <td style="background-color: null">
    `+k8s:eachKey=&lt;eachKey-validator-tag>`
   </td>
   <td style="background-color: null">
    N/A
   </td>
  </tr>
  <tr>
   <td style="background-color: null">
    validate each value
   </td>
   <td style="background-color: null">
    `+k8s:eachVal=&lt;eachVal-validator-tag>`
   </td>
   <td style="background-color: null">
    N/A
   </td>
  </tr>
  <tr>
   <td style="background-color: null">
    uniqueness
   </td>
   <td style="background-color: null">
    `+k8s:listType=&lt;type>`
   </td>
   <td style="background-color: null">
    `x-kubernetes-list-type`
   </td>
  </tr>
  <tr>
   <td style="background-color: null">
    shared struct fields (subfield)
   </td>
   <td style="background-color: null">
    `+k8s:subfield(subField-json-name)=&lt;subfield-validator-tag>`
   </td>
   <td style="background-color: null">
    N/A
   </td>
  </tr>
</table>

The below rules are not currently implemented in the [validation-gen prototype](http://prototype ) so the exact syntax is still WIP

<table>
  <tr>
   <td>
    <strong>Type of validation</strong>
   </td>
   <td>
    <strong>IDL tag</strong>
   </td>
   <td>
    <strong>Relative OpenAPI validation field</strong>
   </td>
  </tr>
  <tr>
   <td>
    regex matches
   </td>
   <td>
    `+k8s:pattern`
   </td>
   <td>
    `pattern`
   </td>
  </tr>
  <tr>
   <td>
    cross field validation
   </td>
   <td>
    `TBD
   </td>
   <td>
    `x-kubernetes-validations`
   </td>
  </tr>
  <tr>
   <td>
    <a href="https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#transition-rules">transition rules</a>
   </td>
   <td>
    `TBD
   </td>
   <td>
    `x-kubernetes-validations`
   </td>
  </tr>
</table>


### Supporting Declarative Validation IDL tags On Shared Struct Fields

IDL tags may be used directly on type declarations and indirectly on field and type aliases. For example:

```go

type ObjectMeta struct {
	// ISSUE: we can't add both IDL tags to the shared struct field directly
    // +k8s-format=dns-label
    // +k8s-format=ip
	Name string
}

type Foo struct {
    // Foo.Name should be a DNS label
  metav1.ObjectMeta
}

// Foo wants Foo.Name to be ...
type Bar struct {
  // Bar.Name should be an IP address
  metav1.ObjectMeta
}

```

Shared types present a challenge. For example, different Kubernetes resources have different validation rules for `metadata.name` and `metadata.generateName`. But all resources share the `ObjectMeta` type.

#### `subfield` IDL Tag

To handle this case, we provide an IDL tag - `k8s:subfield(&lt;field-json-name>)` which can be used to specify a `subfield` validation to add to parent which validates against the the `subfield` value:

```go
type Struct struct {
  // +k8s:subfield(name)=+k8s:format=dns-label
  metav1.ObjectMeta
}
```

This will also support chaining of subfield calls with other validators (including subfield) which allows for setting subfield validations on arbitrarily deep nested fields of shared structs.  An exaggerated example showcasing this is below: \

```go
// +k8s:subfield(sliceField)=+k8s:eachVal=+k8s:subfield(stringField)=+k8s:<desired-validaton-rule>
```


#### `validation-gen` One-deep typedef support

<<[UNRESOLVED is it ok that `validation-gen`'s currently design only supports one-deep typedef support?  There are potential solutions to extend this, see below]>>

In `validation-gen` simple one-deep typedefs work, but not two-deep.  

Given:
```go
// +k8s:minLength=4
type Foo string

// +k8s:maxLength=16
type Bar Foo

type Struct struct {
    // +k8s:format=dns-label
    FooField Foo

    // +k8s:format=dns-label
    FooField Bar
}
```

In the above example, FooField would be validated as a DNS label and have at least 4 characters which is expected.  What might also be expected though is that BarField would be validated as a DNS label, have at least 4 characters, and have no more than 16 characters. INCORRECT!  Due to Go's type system, the relationship of type Foo -> string is represented, but type Bar -> type Foo -> string is flattened to type Bar -> string.  This leads to a currently open question around the severity of this potential UX issue as well potential solutions on how this could be mitigated if needed.

##### Proposed Solutions

1. **Have it well documented that `validation-gen`only support one-deep typedef:
    *   Pro: Doesn't require any additional architectural changes to `validation-gen`.
    *   Con: Opens the door for potential UX issues for users and validations generally as there is the potential for unintended outcomes when adding IDL tags.

2.  Re-discover the chain of typedefs to support and implement nested typedef for IDL tags 
    *   Pro: Better UX for users as it reduces chance users get unintended outcomes when adding IDL tags.
    *   Pro: More intuitive, allows for supporting nested typedef for IDL tags.
    *   Con: Different than Go's "usual" semantic.
    *   Con: Requires implementing logic for re-discovering the chain of typedef.

3.  Implement the chain of typedefs logic BUT only use it to issue warnings/errors to not do that
    *   Pro: Better UX for users as they are notified not to use IDL tags that might lead to unintended outcomes when adding IDL tags
    *   Con: Requires implementing the chain of typedefs logic.

<<[/UNRESOLVED]>>

### Migration Plan

This plan outlines the steps involved in migrating Kubernetes API validation from handwritten code to a declarative approach using validation tags and code generation. The process of migration will be incremental and community-driven. 

We should be able to start the migration when:

*   `validation-gen` is functional with the required set of tags implemented from the list above for Beta and then GA
*   `DeclarativeValidation` feature gate is introduced.
*   A linter is available (`validation-gen --lint`).

#### Phase1: Initialization (Responsibility of Contributors Implementing the KEP)

1. Implement the test plan: [Validation Test Framework](https://docs.google.com/document/d/143KUaYcrEnxEWwPWrQH5P0MsF_V2Yrr8bsXJY76g86I/edit?resourcekey=0-K-n-yyITXtHbKVZnT9kAtw&tab=t.0#heading=h.hkbvgeat5xy8)
2. Prototype and Initial API Selection (Core Team)
*   Select a small set of representative Kubernetes API resources. (core/v1/replicationcontroller)
*   Implement a working prototype by applying the entire process (adding IDL tags, generating code, updating tests) to these selected resources.
3. Documentation and Contribution Guide (Core Team):
*   Write documentation explaining how to:
    *   Add validation tags (IDL tags).
    *   Run validation-gen.
    *   Update unit and E2E tests.
*   Publish a contribution guide for the declarative validation migration.

#### Phase2: Scaling the Migration (Responsibility of Contributors Implementing the KEP and broader community)

1. Tracking Issue and Progress Management:
*   Create a central tracking issue on GitHub.
*   Break down the migration into smaller, manageable tasks. There are couple of options:
    *   Per validation rule (recommended): Migrate a single validation rule for a specific field. E.g., `+k8s:minimum=0` for field `ReplicationControllerSpec.Replicas`
    *   Per Field (recommended): Migrate all validation rules for a single field.
    *   Per Type: Migrate all validation rules for a single API. e.g. all validation rules of `ReplicationControllerSpec`.
    *   Per Group (not recommended): Migrate the entire API group/version.
*   Label tasks appropriately.
2. Community-Driven API Migration:
*   Community:
    *   Analyze existing handwritten validation.
    *   Add appropriate IDL tags to API type definitions.
    *   Run `validation-gen` to generate validation code.
    *   Update unit tests to use the generated validation and ensure coverage.
    *   Update E2E tests to verify behavior with declarative validation.
    *   Submit pull requests (PRs) with the changes.
*   Core Team:
    *   Provide technical guidance and support to community contributors.
    *   Review PRs.
    *   Monitor the tracking issue and adjust the plan as needed.
    *   Add/extend validators to enable further progress into non-trivial cases
3. Using Schemas for Validation (Joint Effort):
*   Core Team:
    *   Enable validation through generated schemas for migrated resources (controlled by DeclarativeValidation feature gate).
    *   Implement logic to populate default values from schemas.
*   Community:
    *   Run E2E tests with declarative validation enabled.

#### 	Phase3: Finalization and GA (Core Team and community)

1. After DeclarativeValidation reaches GA
2. Deprecation of Legacy Validation is Announced
3. ...Deprecation wait period passes (period adhering to community policy)…
4. Legacy validation code that is being validated declaratively can safely be deleted


### Tagging and Validating Against Internal vs Versioned Types
<<[UNRESOLVED what option to use for tagging and validating against internal vs versioned types ]>>

In Kubernetes there are internal schema representations and versioned schema representations for k8s types. The IDL tags can be added to either set of types (and plumbed to validate against that type).  Below are the options and associated tradeoffs related to `validation-gen` using internal, versioned, or relevant combinations of the two:

1. **Use versioned-type validation (Recommended)**.  Tags would be added on every version’s definition of a given type or field.
    *   Pro: Validation is obvious for a given version by looking at types.go.
    *   Con: Tags across versions need to stay in sync.
        *   Rebuttal: This can be easily solved with tests.
    *   Pro: Field-path differences between API versions are handled.
    *   Con: Requires we deal with the performance implications up front.
        *   Don’t want to convert internal &lt;-> versioned over and over.
    *   Pro: other tags, including closely-related tags like +default are already defined on versioned types.
2. **Use internal-type validation**.  Tags would be added on the “unversioned” definition of a given type or field.
    *   Pro: No immediate performance implications.
    *   Con: Need to also add ~equivalent tags to versioned types, in order to generate openapi. Tags across versions need to stay in sync (manually or with automation).
        *   Harder to automate cross-checking.
    *   Con: Need to handle field-path differences between versions (e.g. with post processing).
    *   Con: closely-related tags like +default are already defined on versioned types, which means we need to "thread together" fields across internal and versioned, even across renames or refactoring.
3. **Phased approach**:
    *   Milestones:
        *   1: Add Declarative validation using tags on internal type
        *   2: Add tags on versioned types, enabling OpenAPI
        *   3: Migrate declarative validation to versioned types
    *   Pro: Isolates the critical, high risk work to the first milestone, allowing everyone to focus purely on achieving that with minimal distractions.
    *   Con: Risk that we never get around to finishing all milestones.
        *   Finishing only milestone 1 means we don't get any OpenAPI publication of validation tags
        *   Finishing milestone 1 and 2 means we have tags on internal and versioned types that must be kept in sync, _increasing_ reviewer burden
    *   Con: Increases total work by taking less direct path toward goals

<<[/UNRESOLVED]>>

### Handling Zero Values in Declarative Validation

<<[UNRESOLVED what option to use for handing zero values in Declarative Validation ]>>

Declarative validation has challenges dealing with zero values in Go types. The core issue stems from the fact that zero values can be treated both as valid inputs and equivalent to unspecified or unset values.. This creates discrepancies between how Go code handles validation and how declarative validation, based on the schema, would interpret the same data.  Ex: `ReplicationControllerSpec.MinReadySeconds` might legitimately be set to `0`, indicating that a pod is considered available immediately. This challenges the general assumption in some contexts that zero values for optional fields are inherently invalid, as Kubernetes can treat in some cases as set values or defaults.

#### Difficulties with `+required` and `+default`

The straightforward approach of using the `+required` tag to enforce the presence of a field fails when the zero value is valid. Applying `+required` can incorrectly reject legitimate zero values. Similarly, using `+default` to explicitly document the default value (even if it's the zero value) creates problems because `+default` implies requiredness on the server side.

#### Proposed Solutions

1. <strong>Tri-State mutually exclusive options (Recommended): `+optional`, `+default`, `+required`:</strong>
    *   Treat `+optional`, `+default`, and `+required` as mutually exclusive options.
    *   Fields that allow valid zero values and have defaults would be explicitly tagged with neither `+optional` nor `+required`.
    *   Validation logic would need to be aware of this and handle zero values appropriately for such fields.
    *   <strong>Drawback:</strong> This approach requires a linter to enforce the tri-state rule and prevent invalid combinations.
    *   <strong>Benefit:</strong> Simplifies the mental model by making the relationship between optionality, defaults, and requiredness explicit.
2. <strong>`optional-default: zero-allowed` Tag:</strong>
    *   A new tag could be introduced to signify that a zero value is permissible, even with a default.
    *   <strong>Drawback:</strong> Adds complexity by introducing another tag and complicates the mental model.
3. <strong>Compile-Time or Runtime Default Value Check:</strong>
    *   <strong>Compile-Time Check:</strong> During code generation, the `+default` tag could be parsed, and if it refers to a zero value, validation logic could be adjusted accordingly.
    *   <strong>Drawback:</strong> Complex implementation, requires more information to be available during code generation.
    *   <strong>Runtime Check:</strong> Validation logic could check if the provided default value is a zero value and skip certain checks.
    *   <strong>Drawback:</strong> Considered overly-complicated ("gross") and potentially impacts performance.
    *   <strong>Benefit:</strong> Closest to correct.
4. <strong>Make `+optional` on non-pointer fields be advisory:</strong>
    * If we find an optional string field, the optional tag can be used as documentation, but the actual validation will rely on the format-checking (e.g. dns-label). To an end user this means that what used to be a "field is required" error now becomes a "not a dns-label" error. Only slightly worse.

#### Addressing the Problem with Valid Zero Values Using the Linter

The linter, as previously described, will enforce rules to address valid zero-value challenges. Specifically, it will:

*   Enforce the chosen zero-value handling strategy
    *   Tri-state solution: Ensure `+optional`, `+required`, and `+default` are mutually exclusive.
    *   `optional-default: zero-allowed solution`: Verify correct usage of this tag.
*   Validate `+default` values
    *   Check compatibility of `+default` values with field type and other validation rules (where applicable).
    *   Perform checks on other tag values based on any `+default` tag value (where applicable).

The linter will flag any violations of these rules, ensuring consistent zero-value handling and preventing related errors. This automated enforcement is crucial for catching issues early in the development process.

<<[/UNRESOLVED]>>

### Ratcheting

As `validation-gen`‘s go validation code has old object access we can write any transition rule we want for validation ratcheting. `validation-gen` has access to the old and new state of obj for any field.  This gives us a building block to make any flavor of ratcheting we would need in theory.  We can think the basic form of ratcheting as - “allow old value to be written in updates even if now not valid as long as it doesn’t change -> easy to test”.  If old = new -> short circuit validation (don’t care if it failed).  If certain ratcheting needs syntactic sugar, we can add that as well based ont the current `validation-gen` design. An example of what ratcheting validation logic might look like for `validation-gen` is below:

```go
// Tightened validation (also known as ratcheting validation) would be supported by
// defining a new validation function. For example:
//
        func TightenedMaxLength(opCtx operation.Context, fldPath *field.Path, value, oldValue *string) field.ErrorList {
          if oldValue != nil && len(MaxLength(opCtx, fldPath, oldValue, nil)) > 0 {
            // old value is not valid, so this value skips the tightened validation
            return nil
          }
          return MaxLength(opCtx, fldPath, value, nil)
        }

```

### Test Plan

    [x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates

 None

##### Unit tests

###### `validation-gen` Framework And Validator Tests

For the `validation-gen` code generator and framework, unit testing will be done for all core functionality and validators in `validation-gen`.  An example of the extensive units test for `validation-gen`’s validators can be seen here in the `output_tests` directory in the current prototype:

https://github.com/jpbetz/kubernetes/tree/validation-gen/staging/src/k8s.io/code-generator/cmd/validation-gen/output_tests \
 \
`validation-gen` validators will also use test fixtures for validator testing which allows for additional test coverage.  Example of such a generated test (`zz_generated.validations_test.go`) from the prototype is here:  \
https://github.com/jpbetz/kubernetes/blob/validation-gen/staging/src/k8s.io/code-generator/cmd/validation-gen/output_tests/pointers/zz_generated.[validations_test.go](https://github.com/jpbetz/kubernetes/blob/validation-gen/staging/src/k8s.io/code-generator/cmd/validation-gen/output_tests/pointers/zz_generated.validations_test.go)

##### Integration tests

###### 	Migration Equivalency Tests

When migrating to Declarative Validation each migrated schema will be plumbed through for an equivalency test against the current unit tests (for the handwritten-validation) in the associated `validation_test.go` file.  These tests will be modified to have additional logic to run against the replacement declarative validation logic and verify equivalence of outputs for the same test logic.  The current prototype of this works by running the relevant hand-written test logic with the featuregate `DeclarativeValidation` enabled/disabled and then verifying the logic and associated expected validation, errors, etc. are identical.  Below is a snippet of how this would work for successCases (errorCases are similarly added as well just not in the snippet).  A full diff of the changes made against `validation_test.go`  can be found in the prototype PR here. \

PR: https://github.com/jpbetz/kubernetes/pull/61 \
Github gist of changes: https://gist.github.com/aaron-prindle/b106b24f74770218b3e84005ddeb1bca

```go
        for _, tc := range successCases { // <--- errorCases done similarly
                for _, gateVal := range []bool{false, true} {
                        gate := features.DeclarativeValidation
                        featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, gate, gateVal)

                        errs := ValidateReplicationController(&tc, PodValidationOptions{})
                        if utilfeature.DefaultFeatureGate.Enabled(gate) {
                                // If declarative validation is enabled, it's the union of
                                // managed and declarative validation that we are testing.
                                versioned := v1.ReplicationController{}
                                if err := v1util.Convert_core_ReplicationController_To_v1_ReplicationController(&tc, &versioned, nil); err != nil {
                                        t.Fatalf("failed to convert to v1: %v", err)
                                }
                                ctx := request.WithRequestInfo(context.Background(), &request.RequestInfo{
                                        APIGroup:   "",
                                        APIVersion: "v1",
                                })
                                errs = append(errs, rest.ValidateDeclaratively(ctx, nil, legacyscheme.Scheme, &versioned)...)
                        }
                        if len(errs) != 0 {
                                t.Errorf("expected success: %v", errs)
                        }
                }

                verifyVersionedValidationEquivalence(t, &tc, nil)

```

###### Fuzz testing

The current `validation_test.go` tests may have gaps such that using the current `validaton_test.go` tests for 1:1 functional equivalence is not always guaranteed.  For example, in the current Declarative Validation prototype migration experiment attempting to migrate ReplicationSpec ([here](https://github.com/jpbetz/kubernetes/pull/61)) it was discovered that the current tests did not validate all the validation logic of all fields properly.  In order to enhance the equivalency checks being done there will also be new fuzz tests added to `validation_test.go` with similar equivalency validations to those mentioned above.  This way we enhance the coverage of our equivalency testing and be more confident in our assumption that the hand-written vs declarative validation is logically identical.  An example of what this might look like can be seen in the snippet below:

```go
func TestVersionedValidationByFuzzing(t *testing.T) {
        for i := 0; i < *roundtrip.FuzzIters; i++ {
                gv := schema.GroupVersion{Group: "", Version: "v1"}
                f := fuzzer.FuzzerFor(apitest.FuzzerFuncs, rand.NewSource(rand.Int63()), legacyscheme.Codecs)
                for kind := range legacyscheme.Scheme.KnownTypes(gv) {
                        obj, err := legacyscheme.Scheme.New(gv.WithKind(kind))
                        if err != nil {
                                t.Fatalf("could not create a %v: %s", kind, err)
                        }
                        f.Fuzz(obj)
                        verifyVersionedValidationEquivalence(t, obj, nil)

                        old, err := legacyscheme.Scheme.New(gv.WithKind(kind))
                        if err != nil {
                                t.Fatalf("could not create a %v: %s", kind, err)
                        }
                        f.Fuzz(old)
                        verifyVersionedValidationEquivalence(t, obj, old)
                }
        }
}
```

###### strategy_test.go vs validation_test.go
<<[UNRESOLVED should we move current validation_test.go logic to strategy_test.go as it makes more sense generally and aids this project? ]>>


Currently, validation logic and associated tests are logically split across `validation.go` and `strategy.go`. The hand-written validation functions and associated tests reside in `validation.go` and `validation_test.go`  while `strategy.go` determines when to invoke these functions during API object creation and updates. With the introduction of declarative validation (controlled by the `DeclarativeValidation` feature gate)  the current logic split is worth considering moving from `validation_test.go` to `strategy_test.go` as the current logic split in the test is unfavorable for the migration as it requires additional plumbing work.

The current approach in the prototype experiment to migrate ReplicationController involves directly injecting declarative validation and equivalency tests into `validation_test.go`. This is achieved by conditionally appending calls to `rest.ValidateDeclaratively` within the existing test cases, based on the `DeclarativeValidation` feature gate's status. This allows for a direct comparison of the outputs between hand-written and declarative validation within the same test framework.

**Moving the feature gate check and declarative validation logic to `strategy_test.go` could offer several advantages</strong>:

*   **Reduced Test Duplication:** `validation_test.go` could be simplified, as it would no longer need to handle both hand-written and declarative validation paths.
*   **Clearer Separation of Concerns:** `strategy.go` would be responsible for determining _when_ to validate, while `validation.go` would handle _how_ to validate.
*   **Easier Migration:**  Transitioning to a fully declarative model would be smoother, as the core validation logic would already be invoked through the strategy.

However, this approach also has potential drawbacks:

*   **Additional PRs for Initial Migration:** Moving existing hand-written validation logic from `validation.go` to `strategy.go` would result in additional work via a PR to move the related code but it would be straightforward (only moving files).

Given the low complexity of moving this code prior to the changes, the enhanced logic split of moving the code, and the reduced work for the migration that moving this code would have currently the plan for Declarative Validation is that the current `validation_test.go` tests are moved to `strategy_test.go` in a PR prior to the migration PR.  

<<[/UNRESOLVED]>>

###### Error Message Equivalence

Some error messages will be phrased differently but preserve the same semantic meaning when converted to the declarative validation system. For our testing to check if errors are changed we have two options:

1. Write trusted regexes to match equivalence that work pretty well but have no guarantee of being complete.
    * We may choose to do this for cases where the probability of false equivalence is near-zero. e.g. `field.Required(fldPath, "")` might have a slightly different error from the standard openapi `required`, but they should be seen as equivalent.

2. For any other API field that might have a changed error message, we note the pairing in an exception file somewhere reachable from the comparison code.

##### e2e tests

We will instrument as many e2e tests as necessary  comparing enablement/disablement of the `DeclarativeValidation` feature gate across, identical test inputs, and identical fuzz test inputs.  Any differences will result in a test failure. These tests will be integrated into the current testing infrastructure such that k8s CI will be able to notify and prevent any validation differences over time.

###### Benchmark Tests

To ensure that Declarative Validation and some of the identified potential performance differences (eg: additional conversion for validation) do not meaningfully impact performance, an E2E benchmark test will be created to evaluate the performance of declarative validation.  This way we will be able to ensure that declarative validation meets the performance criteria necessary for GA.

### Graduation Criteria

#### Beta

##### DeclarativeValidation

*   Validation Logic Unit tests (See “unit tests” section above).  Includes validation_test.go equivalence testing, Fuzzing tests, and Test Fixture generated tests.
    *    `validation-gen` code generator is implemented with the necessary subset of supported IDL tags for fully migrating a small number of fields from handwritten-validation to declarative validation.
    *    `validation-gen` Unit tests
*   Initial e2e tests completed and enabled
*   `DeclarativeValidation` feature gate configured and appropriately plumbed through to enable proper toggling of validation logic
*   Have annotated and enabled declarative validation initially for one full schema of a native groupversion
*   Linter with necessary linter rules for handling zero values

#### GA

##### DeclarativeValidation

*   `validation-gen` code generator supports the full set of necessary IDL tags for 1:1 porting of handwritten validation to declarative validation
*   Have plumbed all previous validation_test.go unit tests to run against declarative validation schemas.
*   All Unit and Integration tests pass with no errors or only well-understood exceptional errors sourced from a file in the repository
*   Linter finalized with complete set of linter rules
*   All declarative validation rules documented and published

### Upgrade / Downgrade Strategy

N/A. This feature replaces the implementation of validation and validation equivalence can be directly tested and guaranteed prior to rollout.

### Version Skew Strategy

N/A. This change does not affect any communications going out of the apiserver. This feature replaces the implementation of validation and validation equivalence can be directly tested and guaranteed prior to rollout.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

1. Feature gate (also fill in values in `kep.yaml`)
    *   Feature gate name: DeclarativeValidation
    *   Components depending on the feature gate: kube-apiserver
2. Other
    *   Describe the mechanism:
    *   Will enabling / disabling the feature require downtime of the control plane?
    *   Will enabling / disabling the feature require downtime or reprovisioning of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

*   `DeclarativeValidation`
    *   Beta: Swaps a portion of the validations from using native validations to generated validation code (from IDL tags).
    *   GA: Uses entirely Declarative Validations for user errors. Legacy hand-written validations removed over time.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

*   DeclarativeValidation:
    *   Beta: Since validation is used to persist objects to storage, if there was a mistake and it permits objects that should not be persisted, rollback may be impacted by preventing updates on that object until the error is fixed.
    *   GA: Since validation is used to persist objects to storage, if there was a mistake and it permits objects that should not be persisted, rollback may be impacted by preventing updates on that object until the error is fixed.

###### What happens if we reenable the feature if it was previously rolled back?

*   DeclarativeValidation: If the port to declarative validation is not accurate, during an update, old parts of an object may be seen as invalid and need to be ratcheted.

###### Are there any tests for feature enablement/disablement?

For targeting beta, this enhancement will have tests for:

*   Yes, this enhancement specifically tests that all of the current validation tests have no differences when with `DeclarativeValidation` enabled or disabled.  In this way we can be confident that when `DeclarativeValidation` is enabled there is no functional difference in how objects are validated.

### Rollout, Upgrade and Rollback Planning

        This section must be completed when targeting beta to a release.

###### How can a rollout or rollback fail? Can it impact already running workloads?

Beta & GA:

A rollout can fail by being too strict with updates. If the ported declarative validations are stricter than native validations then workloads may not be able to execute their update operations.

A rollback can fail if our declarative validations are too loose. Workloads wont be able to update objects with an invalid field until the object is corrected.

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

        This section must be completed when targeting beta to a release.

        For GA, this section is required: approvers should be able to confirm the previous answers based on experience in the field.

###### How can an operator determine if the feature is in use by workloads?

```bash
    $ kubectl get --raw /metrics | grep DeclarativeValidation
    # look for if the featuregate is enabled from /metrics
kubernetes_feature_enabled{name="DeclarativeValidation",stage=""} 1
```

###### How can someone using this feature know that it is working for their instance?

1. Events
    *   Event Reason:
2. API .status
    *   Condition name:
    *   Other field:
3. Other (treat as last resort)
    *   Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

4. Metrics
    *   Metric name: apiserver_request_duration_seconds

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies

        This section must be completed when targeting beta to a release.

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

        For alpha, this section is encouraged: reviewers should consider these questions and attempt to answer them.

        For beta, this section is required: reviewers must answer these questions.

        For GA, this section is required: approvers should be able to confirm the previous answers based on experience in the field.

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

DeclarativeValidation: Yes, when enabled the feature may impact validation time. Benchmarks will be taken to measure impact.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

DeclarativeValidation: Yes, when enabled the feature may impact validation CPU and RAM usage. Benchmarks will be taken to measure impact.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

        This section must be completed when targeting beta to a release.

        For GA, this section is required: approvers should be able to confirm the previous answers based on experience in the field.

###### How does this feature react if the API server and/or etcd is unavailable?

No change in behavior.

###### What are other known failure modes?

*   Since validation is used to persist objects to storage, if there was a mistake and it permits denies objects that should be persisted a cluster might end up in an unusable state.  As mentioned above we have test mitigations present to prevent this s.t. the validations are identical

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

## Drawbacks

## Alternatives

### Use CEL and OpenAPI libraries directly for K8s Native Types ([KEP-4153](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/4153-declarative-validation))

## Infrastructure Needed (Optional)
