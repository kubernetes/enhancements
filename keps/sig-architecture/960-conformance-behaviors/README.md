# Behavior-driven Conformance Testing

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Role: Developer](#role-developer)
      - [Promote a Non-Optional Feature to GA](#promote-a-non-optional-feature-to-ga)
      - [Creating a brand new feature, either required or optional](#creating-a-brand-new-feature-either-required-or-optional)
    - [Role: Kubernetes Vendor](#role-kubernetes-vendor)
      - [Evaluating a distribution for conformance](#evaluating-a-distribution-for-conformance)
      - [Identifying the profiles supported by a distribution](#identifying-the-profiles-supported-by-a-distribution)
    - [Role: CNCF Conformance Program](#role-cncf-conformance-program)
      - [Evaluate a Vendor Submission](#evaluate-a-vendor-submission)
    - [Role: CI Job](#role-ci-job)
      - [Identify a PR as requiring conformance review](#identify-a-pr-as-requiring-conformance-review)
      - [Evaluating a PR for conformance coverage](#evaluating-a-pr-for-conformance-coverage)
    - [Role: Behavior Approver](#role-behavior-approver)
      - [Review / approve new suites and behaviors](#review--approve-new-suites-and-behaviors)
      - [Verify behaviors follow the rules](#verify-behaviors-follow-the-rules)
    - [Role: Test Approver](#role-test-approver)
      - [Review / approve new tests](#review--approve-new-tests)
      - [Verify behavior coverage](#verify-behavior-coverage)
      - [Verify non-flakiness of tests](#verify-non-flakiness-of-tests)
      - [Verify test follows the rules](#verify-test-follows-the-rules)
    - [Role: SIG](#role-sig)
      - [Define expected behaviors for their area of responsibility](#define-expected-behaviors-for-their-area-of-responsibility)
  - [Solution Overview](#solution-overview)
    - [Representation of Behaviors (Phase 1)](#representation-of-behaviors-phase-1)
    - [Generating Lists of Behaviors (Phase 1)](#generating-lists-of-behaviors-phase-1)
    - [Coverage Tooling (Phase 2)](#coverage-tooling-phase-2)
    - [Developer and CI Support (Phase 2)](#developer-and-ci-support-phase-2)
  - [Generating Test Scaffolding (Phase 3)](#generating-test-scaffolding-phase-3)
    - [Handwritten Behaviour Scenarios](#handwritten-behaviour-scenarios)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Phase 1](#phase-1)
  - [Phase 2](#phase-2)
  - [Phase 3](#phase-3)
  - [Graduation Criteria](#graduation-criteria)
  - [Future development](#future-development)
    - [Complex Storytelling combined with json/yaml](#complex-storytelling-combined-with-jsonyaml)
    - [Example patch test scenario](#example-patch-test-scenario)
    - [Generating scaffolding from Gherkin .feature files](#generating-scaffolding-from-gherkin-feature-files)
    - [Autogeneration of Test Scaffolding](#autogeneration-of-test-scaffolding)
    - [Combining gherkin with existing framework](#combining-gherkin-with-existing-framework)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Annotate test files with behaviors](#annotate-test-files-with-behaviors)
  - [Annotate existing API documentation with behaviors](#annotate-existing-api-documentation-with-behaviors)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary


**NOTE**

This KEP is now withdrawn. After initial attempts at implementation it
became clear that the effort to rework existing tests to follow this methodology
was more extensive than expected. Additionally, the problems this KEP intended
to solve (particularly separate reviewer pools) have not proven to be the
bottleneck in test creation and promotion.

**END NOTE**

This proposal modifies the conformance testing framework to be driven by a list
of agreed upon behaviors. These behaviors are identified by processing of the
API schemas, documentation, expert knowledge, and code examination. They are
explicitly documented and tracked in the repository rather than in GitHub
issues, allowing them to be reviewed and approved independently of the tests
that evaluate them. Additionally it proposes new tooling to generate tests, test
scaffolding, and test coverage reports.

## Motivation

It has proven difficult to measure how much of the expected Kubernetes behavior
the current conformance tests cover. The current measurements are based upon
identifying which API endpoints are exercised by the tests. The Kubernetes API
is CRUD-oriented, and most of the client’s desired behavior is encapsulated in
the payload of the create or update calls, not in the simple fact that those
endpoints were hit. This means that even if a given endpoint is shown as
covered, it’s impossible to know how much that tests the actual behavior.

Coverage is measured this way because there is no single, explicit list of
behaviors that comprise the expected behavior of a conformant cluster. These
expectations are spread out across the existing design documents, KEPs, the user
documentation, a subset of the e2e test, and the code itself. This makes it
impossible to identify if the conformance suite provides a meaningful test of a
cluster’s operation.

Additionally, progress in writing and promoting tests has been slow and too much
manual effort is involved. As a starting point, this proposal includes new
tooling that uses the API schemas to identify expected behaviors and produce
tests and test scaffolding to quickly cover those behaviors.

### Goals

* Enable separate review of behaviors and tests that evaluate those behaviors.
* Provide a single location for defining conforming behavior.
* Provide tooling to generate as many of the behaviors as possible from API
  schemas. This will be a seed for the behavior lists, which will in turn be
  human curated. Refinements can improve the quality of the seed over time.
* Provide tooling to generate tests and test scaffolding for validating
  behaviors.
* Provide tooling to measure the conformance test coverage of the behaviors.
* Provide an incremental migration path from current conformance testing to the
  updated framework.

### Non-Goals

* Develop a complete set of behaviors that define a conforming cluster. This is
  an ongoing effort, and this proposal is intended to make that more efficient.
* Add new conformance tests. It is expected that during this effort new
  tests may be created using the proposed tooling, but it is not explicitly part
  of this proposal.
* Provide tooling that perfectly populates the behaviors from the API schema.
  Not enough information is present in the schema to achieve this. The tooling
  is only intended to produce a seed for human curation.

## Proposal

### User Stories

#### Role: Developer

##### Promote a Non-Optional Feature to GA

Conformance tests are required when promoting a non-optional feature to GA.

Today, the desired process consists of writing the tests as ordinary e2e tests,
making sure they are not flaky by having them run for several weeks without
flakes, and then including the promotion of those tests in the PR that promotes
the feature. However, even without the test promotion, PRs that promote
features are already large; for example:

* [Promote PodDisruptionBudget to
  GA](https://github.com/kubernetes/kubernetes/pull/81571) (91 files changed)
* [Promote block volumes to
  GA](https://github.com/kubernetes/kubernetes/pull/88673) (46 files changed)
* [Promote node lease to
  GA](https://github.com/kubernetes/kubernetes/pull/84351) (17 files changed)

Thus, today developers typically submit the test promotions in a separate PR, in
order to avoid adding more changes, along with an additional review team that
further slows the merge. This makes it difficult to develop a CI job that
prevents features from going to GA without conformance tests.

With the separation of behaviors and tests, the tasks a developer needs to
complete are:

1. Define expected behaviors
1. Get behaviors approved by the conformance-behavior-approvers
1. Write tests to cover those behaviors
1. Get tests approved by the conformance-test-approvers
   * Prove that the tests that will be conformance are not flaky
   * Promote the tests to conformance in that PR
1. Create a PR that promotes my feature to GA

As a developer, I would like to be able to have as much of this completed and
merged prior to the PR that promotes the feature to GA, in order to avoid
additional reviews on that PR.

<<[UNRESOLVED context and discussion around solutions this use case ]>>
@johnbelamaric
One option: We could get it all to a state where it's all done, but behaviors are
marked as "PENDING". The promo to GA would still require touching the behaviors,
to flip the status from PENDING to ACTIVE, but it should be a formality at that
point. Promo to "conformance" for the tests could have already been done just
with the PENDING status so it won't count yet. Other ideas?
@jefftree
I was thinking something along the same lines. One thing to note is that this is
promoting a set of tests (that cover a set of behaviors) rather than a set of
behaviors themselves. Tests could cover existing behaviors (is this a correct
assumption?) so it might make more sense to have the switch on the tests rather
than the behaviors side.
<<[/UNRESOLVED]>>

<<[UNRESOLVED @spiffxp: should this capture preconditions for testing: ]>>
* All behaviors present
* All behaviors covered by tests
* Tests should have been around to verify non-flakiness
* Tests should have been reviewed by conformance reviewer to make sure they meet
  the criteria - can we front load this?
<<[/UNRESOLVED]>>

##### Creating a brand new feature, either required or optional
During creation of an alpha or beta feature, conformance tests are not required,
nor are conformance behaviors. However, at the beta stage, the expectation
should be to have some quality end-to-end tests, and so we may want to allow the
definition of the behaviors at that time too. Tasks then would be similar to
some of those for GA promotion:
1. Define expected behaviors
1. Get provisional behaviors approved by the conformance-behavior-approvers
1. Write tests to cover those behaviors

<<[UNRESOLVED]>>
@johnbelamaric
Ideally we could avoid the provisional behavior approval. Maybe we can have a
way to have a separate behaviors area for beta stuff? Or maybe we just don't
have this at all for beta, and it waits till GA. The reason I bring up making
bahaviors now is because the initial idea of `kubetestgen` was to support these
steps: creation of behaviors, and creation of standard e2e tests for those
behaviors.
@jefftree
Getting this list of behaviors approved is something that needs to eventually be
done before hitting GA. I don't know how much these behaviors would change
between beta and GA, but if they're relatively stable and mainly additive,
starting the process early seems fine. Similar to your previous point, we should
look to move some of these behavior approvals earlier in the process to avoid
the chaos of reviews when a feature is going to GA.
<<[/UNRESOLVED]>>

#### Role: Kubernetes Vendor

##### Evaluating a distribution for conformance
* Must set up test cluster and run sonobuoy conformance tests
* If successful, submit PR to CNCF. If failures exist, debug them

##### Identifying the profiles supported by a distribution
* Must run a set of conformance tests for each profile supported

#### Role: CNCF Conformance Program

##### Evaluate a Vendor Submission
* Must confirm that the version of the tests being run matches the version being
  certified
* Must confirm the set of tests being run matches the set of tests for the
  version (+ profile(s)) being certified
* Must confirm that all behaviors are covered by a test that executes, and that
  no tests fail (This isn’t done today: verify skew policy - confirm a cluster
  being certified for version 1.x also passes conformance tests for version
  1.x-1)

#### Role: CI Job

##### Identify a PR as requiring conformance review
PR must touch file in conformance-specific directory
* eg: update test/conformance/behaviors/..
* eg: mv from test/e2e to test/conformance

##### Evaluating a PR for conformance coverage
* Must be able to confirm for each behavior that at least one test exercises a
  given behavior
* Must be able to list all expected behaviors for conformance
* Coverage is defined by (exercised behaviors) / (expected behaviors)
* May be able to list set of tests that exercise a given behavior
* Should not bother gating or paying attention too closely to coverage until we
  have locked (expected behaviors) in place;

#### Role: Behavior Approver

##### Review / approve new suites and behaviors
* Must verify that the listed behaviors are common across cluster providers and
  can be supported in new cluster providers.
* Must be able to identify if all of the expected behaviors are listed; this may
  mean seeing API definitions and configuration parameters, if those are
  expected to be part of the defined behaviors.
* Must be able to identify if any behaviors are LinuxOnly

##### Verify behaviors follow the rules
* The minimal set of behaviors for a given resource must include the basic
  functioning of the API CRUD operations, and of the resulting changes in
  cluster / data plane state.
* Must be able to verify behaviors do not rely on features that are deprecated
  (or pending deprecation, eg: componentstatus)
* May strive to minimize the number behaviors that rely on a specific NodeOS

#### Role: Test Approver

##### Review / approve new tests
* Should be able to reject addition of a new test if there is no associated
  behavior
* Should require a behavior approver if a new behavior is added at the same a
  test is added

##### Verify behavior coverage
* Must be able to confirm the test in question actually exercises the
  described/linked behavior(s)
* Should NOT require all test code maps directly to behavior(s) (eg: “it looks
  like you’re exercising the Foo api, is there a Foo behavior that should be
  associated with this test?”).
* When promoting to Conformance, must be able to verify no feature flags or
  additional configuration is necessary to enable the feature
* Must verify there should be at most one test linked to a given behavior. That
  is, implicitly covered behaviors should NOT be listed as covered behaviors.
  There should be a single explicit test for any given behavior.
* One test may cover multiple behaviors

##### Verify non-flakiness of tests
* May be able to identify known anti-patterns in the test code (eg: watches that
  break down at scale, arbitrary sleeps)
* When promoting to Conformance, test MUST have sufficient history to prove
  non-flakiness (eg: today, we link to testgrid and confirm that it looks good…
  we don’t mandate specific thresholds, and we don’t mandate specific cluster
  configurations)

##### Verify test follows the rules
* Must be able to confirm all associated Behavior(s) are eligible for
  Conformance
* Must be able to confirm the test(s) in question exercise only GA APIs
* Must be able to confirm the test(s) in question do NOT require access to
  kubelet APIs to pass
* Must not depend on specific Events (nor their contents) to pass
* Must not depend on optional Condition fields
* etc.

#### Role: SIG

##### Define expected behaviors for their area of responsibility
* Should be able to enumerate list of behaviors for a given API/resource
* Should be able to enumerate list of behaviors for a given feature (eg:
  [Feature:Foo] suite of tests)
* Should be able to enumerate list of behaviors for a given set of e2e tests
  owned by the SIG

### Solution Overview
The proposed solution consists of these deliverables:
* A machine readable format to define conforming behaviors. (Phase 1)
* Tooling to generate lists of behaviors from the API schemas. (Phase 1)
* Tooling to compare the implemented tests to the list of behaviors and
  calculate coverage. (Phase 2)
* Process documentation and supporting tooling to enable developers to move
  their features to GA and add conformance behaviors and tests with minimal
  review burden and entanglements. (Phase 2)
* Migration of the existing tests to use behaviors, and to updates to the
  CNCF conformance program validation of tests to use the behaviors rather
  than simply the test results. (Phase 2)
* Tooling to generate tests and test scaffolding to evaluate those behaviors.
  (Phase 3)


#### Representation of Behaviors (Phase 1)

Behaviors will be captured in prose, which is in turn embedded in a YAML file
along with meta-data about the behavior. More details on exactly what defines
a behavior is documented in the [behaviors
README](https://git.k8s.io/kubernetes/test/conformance/behaviors/README.md).

Behaviors must be captured in the repository and agreed upon as required for
conformance. Behaviors are broken out by owning SIG, and there are multiple
test suites (named sets of tests) for each SIG. Some of these suites
may be machine-generated based upon the API schema, whereas others are
handwritten. Keeping the generated and handwritten suites in separate files
allows regeneration of the auto-discovered behavior suites. Some areas may be
defined by API group and Kind, while others will be subjectively defined by
subject-matter experts.

The grouping at the suite level should be defined based upon subjective
judgement of how behaviors relate to one another, along with an understanding
that all behaviors in a given suite may be required to function for a given
cluster to pass conformance for that suite.

Typical suites defined for any given feature will include:
 * API spec. This suite is generated from the API schema and represents
   the basic field-by-field functionality of the feature. For features
   that include provider-specific fields (for example, various VolumeSource
   fields for pods), those must be segregated into separate suites.
 * Internal interactions. This suite tests interactions between settings
   of fields within the API schema for this feature.
 * External interactions. This suite tests interactions between this feature
   and other features.

Each suite may be stored in a separate file in a directory for the specific
SIG. For example, a `sig-node` directory may have files such as:
these files:
 * `api-generated.yaml` describing the set of behaviors auto-generated from the
   API specification.
 * `lifecycle.yaml` describing the set of behaviors expected from the Pod
   lifecycle.
 * `readiness-gates.yaml` describing the set of behaviors expected for Pod
   readiness gates functionality.

Behavior files are reviewed separately from the tests themselves, with separate
OWNERs files corresponding to those tests. This may be captured in a directory
structure such as:

```
test/conformance/behaviors
│── OWNERS # no-parent: true, approvers: behavior-approvers
│── {sig}
│   ├── OWNERS # optional: reviewers: area-experts
│   └── {suite}.yaml
```

The relationship between tests and behaviors is captured in the conformance test
metadata, which contains a list of behavior IDs covered by the test.

The structure of the behavior YAML files is described by these Go types, which
have been updated for Phase 2 to add the `Status` field. Use of this field is
described later in this KEP. The actual implementation of these types can be
found in
[types.go](https://git.k8s.io/kubernetes/test/conformance/behaviors/types.go).

```go
// Area defines a general grouping of behaviors
type Area struct {
        // Area is the name of the area or SIG.
        Area   string  `json:"area,omitempty"`

        // Suites is a list containing each suite of behaviors for this area.
        Suites []Suite `json:"suites,omitempty"`
}

type Suite struct {
        // Suite is the name of this suite.
        Suite       string     `json:"suite,omitempty"`

        // Description is a human-friendly description of this suite, possibly
        // for inclusion in the conformance reports.
        Description string     `json:"description,omitempty"`

        // Behaviors is the list of specific behaviors that are part of this
        // suite.
        Behaviors   []Behavior `json:"behaviors,omitempty"`
}

type Behavior struct {
        // ID is a unique identifier for this behavior, and will be used to tie
        // tests and their results back to this behavior. For example, a
        // behavior describing the defaulting of the PodSpec nodeSelector might
        // have an id like `pods/spec/nodeSelector/default`.
        ID          string `json:"id,omitempty"`

        // APIObject is the object whose behavior is being described. In
        // particular, in generated behaviors, this is the object to which
        // APIField belongs. For example, `core.v1.PodSpec` or
        // `core.v1.EnvFromSource`.
        APIObject   string `json:"apiObject,omitempty"`

        // APIField is filled out for generated tests that are testing the
        // behavior associated with a particular field. For example, if
        // APIObject is `core.v1.PodSpec`, this could be `nodeSelector`.
        APIField    string `json:"apiField,omitempty"`

        // APIType is the data type of the field; for example, `string`.
        APIType     string `json:"apiType,omitempty"`

        // Status is used to create provisional behaviors prior to them becoming
        // part of the actual conformance criteria, as well as to deprecate
        // prior behaviors.
        Status      string `json:"status,omitempty"`

        // Description specifies the behavior. For those generated from fields,
        // this will identify if the behavior in question is for defaulting,
        // setting at creation time, or updating, along with the API schema field
        // description.
        Description string `json:"description,omitempty"`
}
```

#### Generating Lists of Behaviors (Phase 1)

In Phase 1, a `kubetestgen` tool was created that can be used to generate
candidate behaviors from the API spec, as described below. In Phase 2 this tool
is combined into `kubeconform`, which will handle all of the conformance-related
tooling for developers and CI.

#### Coverage Tooling (Phase 2)

In Phase 1, conformance test meta-data was updated to include a `Behaviors:`
list, which can contain each of the behavior IDs for behaviors explicitly
covered by that test. Implicit behaviors should not be included in the list.

In Phase 2, existing tests will be updated to properly set this field, and
the `kubeconform` tool will be updated to calculate behavior coverage.

#### Developer and CI Support (Phase 2)

Phase 2 will define the process and tooling used by the developer to move a
feature through to GA. This will include documenting the process and producing
any supporting tooling as part of `kubeconform`, to be described in the Design
Details below.

A CI job, and necessary supporting tooling, to verify coverage for features
going to GA will also be implemented.

### Generating Test Scaffolding (Phase 3)

Some sets of behaviors may be tested in a similar, mechanical way. Basic CRUD
operations, including updates to specific fields and constraints on immutable
fields, operate in a similar manner across all API resources. Given this, it is
feasible to automate the creation of simple tests for these behaviors, along
with the behavior descriptions in the `api-generated.yaml`. In some cases a
complete test may not be easy to generate, but a skeleton may be created that
can be converted into a valid test with minimal effort.

For these tests, the input is a set of manifests that are applied to the
cluster, along with a set of conditions that are expected to be realized within
a specified timeframe. The test framework will apply the manifests, and monitor
the cluster for the conditions to occur; if they do not occur within the
timeframe, the test will fail.

For each Spec object, scaffolding can be defined to include the following tests:

* Creation and read of the resource with only required fields specified.
  * API functions as expected: Resource is created and may be read, and defaults
    are set. This is mechanical and can be completely generated.
  * Cluster behaves as expected. This cannot be generated, but a skeleton can be
    generated that allows test authors to evaluate the condition of the cluster
    to make sure it meets the expectations.
* Deletion of the resource. This may be mostly mechanical but if there are side-
  effects, such as garbage collection of related resources, we may want to have
  manually written evaluation here as well.
* Creation of resource with each field set, and update of each mutable field.
  * For each mutable field, apply a patch to the based resource definition
    before creation (for create tests), or after creation (for update tests).
  * Evaluate that the API functions as expected; this is mechanical and
    generated.
  * Evaluate that the cluster behaves as expected. In some cases this may be
    able to re-use the same evaluation function used during the creation tests,
    but often it will require hand-crafted code to test the conditions.
    Nonetheless, the scaffolding can be generated, minimizing the effort needed
    to implement the test.

As an example, the tooling would generate scaffolding which creates a Pod. It
would still be necessary to fill in the values used for the base Pod fixture.
The tooling would also generate a test case that includes a change to `image:`
field of the container spec. It would still be necessary for a human to fill in
the what new value to use for the image. The scaffolding would also generate the
entire test evaluation as described, except that the true/false condition for
whether the desired state is achieved would be an empty function that needs to
be implemented. In this case, the function would wait for the condition that the
Pod's container has been restarted with the new image.  While there is still
human involvement here, much of the toil is removed, with the only necessary
intervention being specifying the specific image values and content of the
function.

This example does illustrate the need for some logic in the generation to avoid
overwriting the specified image values and function content. One option is to
put the fixtures in separate files, rather than embedding them in the generated
files. However that extra indirection and all the extra files can make the tests
difficult to follow. Some more investigation is necessary here.

#### Handwritten Behaviour Scenarios

Additional, handwritten tests will be needed that modify the resource in
multiple ways and evaulate the behavior. The scaffolding must be built such that
the same process is used for these tests. The test author must only need to
define:
* A patch to apply to create or update the resource.
* A function to evaluate the effect of the API call.

With those two, the same creation and update scaffolding defined for individual
field updates can be reused.

### Risks and Mitigations

The behavior definitions may not be properly updated if a change is made to a
feature, since these changes are made in very different areas in the code.
However, given that the behaviors defining conformance are generally stable,
this is not a high risk.

## Design Details
<<[UNRESOLVED]>>
@johnbelamaric: note this section is still not updated for Phase 2
<<[/UNRESOLVED]>>

Delivery of this KEP shall be done in the following phases:

### Phase 1

Phase 1 defined the basic behavior structure and enabled the generation of
behaviors from API schemas.
In Phase 1, we will:
* Implement the behavior formats and types described above. This will include
  separate suites for tooling-generated behaviors and handcrafted behaviors.
* Implement the directory structure described above to contain the behavior
  lists, including how to tie tests back to behaviors.
* `kubetestgen`, a tool which reads the OpenAPI schema and generates the list of
  behaviors.
* Migrate existing conformance tests to work with the new tooling. Existing
  tooling around generation of conformance reports will not be changed in this
  phase.

### Phase 2

In Phase 2, we will:
* Migrate existing tooling for conformance report generation to the new method,
  and remove older tooling. This will eliminate the need to maintain conformance
  tests in both the new and old manner.
* Add test scaffolding generation in parallel with the behavior list generation.
* Implement coverage metrics comparing behavior lists to the coverage captured
  by existing conformance tests.

### Phase 3

### Graduation Criteria
As this is a tooling component and is not user facing, it does not follow the
ordinary alpha/beta/GA process. In 1.17, the intent is to implement Phase 1,
without disruption to any feature development. The acceptance criteria here
are that the deliverables described in Phase 1 are complete, and that no
developers other than those writing or promoting conformance tests are
affected by the changes introduced in this KEP.

### Future development

The description above achieves the basic goals of the KEP. However, in the same
timeframe as implementation of this KEP, we also plan to explore some future
refinements. In particular, we will explore the use of an existing behavior-
driven testing language to refine our *prose* behavior descriptions into
*machine-readable* behavior descriptions.

One such language is [Gherkin](https://cucumber.io/docs/gherkin/). In Gherkin,
specifications are defined around Features, which are collections of Scenarios.

#### Complex Storytelling combined with json/yaml

Inline json or yaml as CRUD input/output can be autogenerated for verification. The
json or yaml can also be contained in external files. The functions matching the
step definitions would be re-used for all matching scenarios as needed.

```feature
Feature: Intrapod Communication
  Pods need to be able to talk to each other, as well as the node talking to the Pod.
  @sig-node @sig-pod
  Scenario: Nodes can communicate to each other
    Given a pods A and B
    When pod A says hello to pod B
    Then pod B says hello to pod A
  @wip @tags-are-no-longer-part-of-test-names
  Scenario: Pods can can communicate to Nodes
    Given a pod A on a node
    When the node says hello to pod A
    Then pod A says hello to the node
    And this is fine
```

#### Example patch test scenario

```feature
Feature: Manually using Manifests to CRUD and evaluate effects
  Pods need to be able to talk to each other, as well as the node talking to the Pod.
  Scenario: Pods can can communicate to Nodes
    Given I create pod A with this yaml spec
      """
      yaml: [
         values
      ]
      """
    And I create pod B with this json spec
      """
      {
        json: values
      }
      """
    When I request pod A and pod B talk to each other
    Then I can observe a v1.PodCommunication matching this json spec
      """
      {
        "node a": "talked to node b"
      }
      """
    And this is fine
```
#### Generating scaffolding from Gherkin .feature files

A Gherkin **feature** is synonymous with our definition of **behaviour**, and
tagging can be used for **@conformance** or **@release-X.Y** metadata.

```feature
Feature: Structured Metadata allowing Behaviour Driven tooling automation
  In order to auto-generate testing scaffolding
  As a sig-X member
  I want to describe the behaviour of X

  @sig-X
  Scenario: Behaviour X
    Given a well formed file describing the behaviour X
    When I run the automation
    Then I am provided with the basic structure for a corresponding test
    And this is fine
  @sig-Y
  Scenario: Behaviour Y
    Given a well formed file describing the behaviour Y
    When I run the automation
    Then I am provided with the basic structure for a corresponding test
    And this is fine
  @sig-Y @sig-X
  Scenario: Behaviour X+Y
    Given a well formed file describing the behaviour X
    And a well formed file describing the behaviour Y
    When I run the automation
    Then I can reuse existing step definitons on multiple tests
    And this is fine
```

#### Autogeneration of Test Scaffolding

```shell
~/go/bin/godog --no-colors
```

```feature
Feature: Structured Metadata allowing Behaviour Driven tooling automation
  In order to auto-generate testing scaffolding
  As a sig-X member
  I want to describe the behaviour of X

  Scenario: Behaviour X                                                  # features/behaviour.feature:7
    Given a well formed file describing the behaviour X
    When I run the automation
    Then I am provided with the basic structure for a corresponding test
    And this is fine

  Scenario: Behaviour Y                                                  # features/behaviour.feature:13
    Given a well formed file describing the behaviour Y
    When I run the automation
    Then I am provided with the basic structure for a corresponding test
    And this is fine

  Scenario: Behaviour X+Y                                       # features/behaviour.feature:19
    Given a well formed file describing the behaviour X
    And a well formed file describing the behaviour Y
    When I run the automation
    Then I can reuse existing step definitons on multiple tests
    And this is fine

3 scenarios (3 undefined)
13 steps (13 undefined)
1.253405ms

You can implement step definitions for undefined steps with these snippets:

func aWellFormedFileDescribingTheBehaviourX() error {
  return godog.ErrPending
}

func iRunTheAutomation() error {
  return godog.ErrPending
}

func iAmProvidedWithTheBasicStructureForACorrespondingTest() error {
  return godog.ErrPending
}

func thisIsFine() error {
  return godog.ErrPending
}

func aWellFormedFileDescribingTheBehaviourY() error {
  return godog.ErrPending
}

func iCanReuseExistingStepDefinitonsOnMultipleTests() error {
  return godog.ErrPending
}

func FeatureContext(s *godog.Suite) {
  s.Step(`^a well formed file describing the behaviour X$`, aWellFormedFileDescribingTheBehaviourX)
  s.Step(`^I run the automation$`, iRunTheAutomation)
  s.Step(`^I am provided with the basic structure for a corresponding test$`, iAmProvidedWithTheBasicStructureForACorrespondingTest)
  s.Step(`^this is fine$`, thisIsFine)
  s.Step(`^a well formed file describing the behaviour Y$`, aWellFormedFileDescribingTheBehaviourY)
  s.Step(`^I can reuse existing step definitons on multiple tests$`, iCanReuseExistingStepDefinitonsOnMultipleTests)
}

```

These functions and the Suite.Step matchers that tie them to Gherkin steps can
be pasted into a `test_steps.go` file as a initial scaffolding.

#### Combining gherkin with existing framework

Our current tests are not super easy to write, read, or review. BDD in go was in
it's early days when k8s started integration testing with a closely coupled
component testing approach. Our Ginko based e2e framework evolved based upon
those tightly coupled assumptions. This approach unfortunately lacks the
metadata, tags, and descriptions of the desired behaviours required for clear
separation of acceptance behaviors and tests.

Documenting and discovering of all our behaviours will require a combination of
automated introspection and well as some old fashioned human storytelling.

To do so need to standardize the business language that our bottlenecked people
can use to write these stories in a way can be assisted with some automation.
This would reduce complexity for articulating concrete requirements for
execution in editors, humans, and automation workflows.

Defining our Behaviours in Gherkin would allow us to leverage our existing
conformance framework and test mechanisms to allow incremental adoption of this
proposal.

Scenarios could be defined for existing tests using the form:

```feature
Scenario: Use existing ginkgo framework
  As a test contributor
  I want to not throw away all our old tests
  In order to retain the value generated in them
  @sig-node @sig-pod @conformance @release-1.15
  Feature: Map behaviours to existing ginkgo tests
    Given existing test It('should do the right thing')
    And I optionally tag it with @conformance
    When I run the test
    Then we utilize our existing test via our new .feature framework
    And this is fine
```

Thus, test authors will indicate the behaviors covered by adding a
**@conformance** tag to Feature/Behaviours using `Given an existing test
It('test string')`


## Implementation History

- 2019-04-12: Created
- 2019-06-11: Updated to include behavior and test generating from APIs.
- 2019-07-08: Updated to include Gherkin / godog as possible behaviour workflow
- 2019-07-24: Updated to add reviewers and an example on generated scaffolding
- 2019-07-30: Updated to separate Gherkin / godog into second phase, include
  directory structure for showing behavior/test separation
- 2019-10-01: Added detailed design; marked implementable
- 2020-03-26: Reformat for new KEP structure
- 2020-04-17: Updated to add use cases, reflect what was implemented in Phase 1
  and add design details for Phase 2

## Drawbacks

* Separating behaviors into a file that is not directly part of the test suite
  creates an additional step for developers and could lead to divergence.

## Alternatives

### Annotate test files with behaviors

This option is essentially an extension of the existing tagging of e2e tests.
Rather than just tagging existing tests, we can embed the list of behaviors in
the files as well. The same set of metadata that is described in Option 1 can be
embedded as specialized directives in comments.


*Pros*
* Keeps behaviors and tests together in the same file.

*Cons*
* All of the same features may be met, but the tooling needs to parse the Go
  code and comments, which is more difficult than parsing a YAML.
* Behaviors are scattered throughout test files and intermingled with test code,
  making it hard to review whether the list of behaviors is complete (this
  could be mitigated with tooling similar to the existing tooling that extracts
  test names).
* Adding or modifying desired behaviors requires modifying the test files, and
  leaving the behaviors with a TODO or similar flag for tracking what tests are
  needed.

### Annotate existing API documentation with behaviors
The current API reference contains information about the meaning and expected
behavior of each API field. Rather than producing a separate list, the metadata
for conformance tests can be attached to that documentation.

*Pros*
* Avoids adding a new set of files that describe the behavior, leveraging what
  we already have.
* API reference docs are a well-known and natural place to look for how the
  product should behave.
* It is clear if a given API or field is covered, since it is annotated directly
  with the API.

*Cons*
* Behaviors are spread throughout the documentation rather than centrally
  located.
* It may be difficult to add tests that do not correspond to specific API
  fields.
