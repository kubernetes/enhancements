<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

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
# KEP-5491: DRA: List Types for Attributes

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
  - [API Changes](#api-changes)
    - [Introduce typed-<code>list</code> in <code>DeviceAttribute</code>](#introduce-typed-list-in-deviceattribute)
    - [Introduce <code>.include</code> function in CEL](#introduce-include-function-in-cel)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Hardware Topological Aligned CPUs &amp; GPUs &amp; NICs](#story-1-hardware-topological-aligned-cpus--gpus--nics)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Go Type Definitions](#go-type-definitions)
    - [<code>DeviceAttribute</code>](#deviceattribute)
  - [Implementation (for evaluating constraints)](#implementation-for-evaluating-constraints)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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
  - [Just support formatted string list instead of introducing <code>list</code> type](#just-support-formatted-string-list-instead-of-introducing-list-type)
  - [Introduce <code>matchSemantics/distinctSemantics</code> field for flexible/declarative match](#introduce-matchsemanticsdistinctsemantics-field-for-flexibledeclarative-match)
    - [<code>matchSemantics</code> field](#matchsemantics-field)
    - [`distinctSemantics](#distinctsemantics)
    - [Pros/Cons](#proscons)
  - [Unified <code>semantics</code> field instead of <code>matchSemantics</code>/<code>distinctSemantics</code>](#unified-semantics-field-instead-of-matchsemanticsdistinctsemantics)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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
-->

The Device Resource Assignment (DRA) API currently allows scalar attribute values to describe device characteristics. However, many real-world device topologies require representing sets of relationships (e.g., multiple PCIe roots, NUMA nodes). This KEP introduces support for list-typed attributes in `ResourceSlice` and extends(redefine) `ResourceClaim`'s `constraints[].{matchAttribute, distinctAttribute}` semantics to fit both list-type attributes and primitive attributes supported previously.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

The `ResourceSlice` API allows users to attach scalar attributes to devices. These can be used to allocate devices that share common topology within the node. For certain types of topological relationships, scalar values are insufficient. For example, a CPU may have adjacency to multiple PCIe roots. This enhancement proposes allowing attributes to be lists. The semantics of the `MatchAttribute` and `DistinctAttribute` constraints must adapt to the possibility of lists. For example, rather than defining an attribute "match" as equality, it would be defined as a non-empty intersection, treating scalars as single-element lists. Conversely, "distinct" attributes for lists would be defined as an empty intersection.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Support typed-list in device attribute values.
- Extends(redefine) the semantics of `ResourceClaim`'s `constraints[].{matchAttribute,distinctAttribute}` fields as below so that it can work with list-type attribute values
  - `matchAttribute`: it is defined as non-empty intersection
  - `distinctAttribute`: it is defined as empty intersection
  - note: scalar values are treated as single-element lists
- Keep _monotonicity_ in constraint.
  - Currently `Allocator`'s algorithm assumes [_monotonic_ constraints](https://github.com/kubernetes/kubernetes/blob/v1.34.2/staging/src/k8s.io/dynamic-resource-allocation/structured/internal/experimental/allocator_experimental.go#L274-L276) only. Monotonic means that once a constraint returns false, adding more devices will never cause it to return true. This allows to bound the computational complexity for searching device combinations which satisfies the specified constraints. This KEP focuses to keep monotonicity of `matchAttribute/distinctAttribute` semantics.
- Maintain backward compatibility and inter-operability for scalar-only attributes.
  - `matchAttribute/distinctAttribute`: existing constraint can work because scalar values are treated as single-value list
  - CEL expressions in device selectors: when the attribute type is updated, existing CEL won't failed to compile. But, we will provide some type-agnostic helper function to achieve easier migration for users/DRA driver developers.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- Introducing generic or complex boolean logic in constraints([KEP-5254: DRA: Constraints with CEL](https://github.com/kubernetes/enhancements/issues/5254)).
- Forcing all drivers to use list attributes immediately.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->


The proposal has mainly two parts:

- Add list-types in `DeviceAttribute` so that DRA drivers can expose the attribute values in typed list(`int`, `string`, `boolean`, `version`)
- Extends the semantics of `MatchAttribute`/`DistinctAttribute` field in `DeviceConstraint`
  - For `MatchAttribute`:
    - Previously: it matches when the attribute values among candidate devices are identical (i.e. `∀i,j, v_i = v_j`)
    - This KEP: it matches when the intersection (as a set) of all the list values among candidate devices is non-empty(i.e. `(∩ v_k != ∅)`)
  - For `DistinctAttribute`
    - Previously: it matches when all the attribute values among candidate devices are distinct (i.e. `∀i,j, s.t. i != j, v_i != v_j`)
    - This KEP: it matches when the intersection (as a set) of all the list values among candidate devices is empty (i.e. `∩ v_k = ∅`)

### API Changes

#### Introduce typed-`list` in `DeviceAttribute`

```yaml
kind: ResourceSlice
spec:
  devices:
  - name: typed-list-attributes
    attributes:
      list-of-string:
        list:
          string: ["pci0000:00", "pci000:01"]
      list-of-int:
        list:
          int: [0, 1, 2]
      list-of-bool:
        list:
          bool: [true, false, true]
      list-of-version:
        list:
          version: ["1.0.0", "1.0.1"]
```
#### Introduce `.include` function in CEL

When the attribute type was changed from scalar to list. Existing CEL won't compile due to type mismatch. 

```
// This CEL won't compile if attributes["foo"] type is changed from 1 (scalar) to [1](list)
attributes["foo"] == 1
```

To maintain backward compatibility for existing CEL expressions, it _might_ be possible to override comparison operators (`==`, etc.) that allows for a list type where `attributes["foo"] == 1` is equivalent to `attributes["foo"] == [1]`. But we don't do this way because it wouldn't be idiomatic and would diverge from normal CEL type system expectations and feels confusing to anyone that already has an understanding of how the CEL type system is suppose to work.

Instead, although user needs to rewrite the existing CEL expressions, it plans to provide a helper function, say `.include`, which can work in type-agnostic way to make the CEL migration easier:

```
// assume attribute["foo"] is 1
attribute["foo"].include(1) --> true

// assume attribute["foo"] is [1]
attribute["foo"].include(1) --> true
```

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1: Hardware Topological Aligned CPUs & GPUs & NICs

Assume several DRA drivers exposed device attribute `resource.kubernetes.io/pcieRoot`:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: cpu
spec:
  driver: "cpu.example.com"
  pool:
    name: "cpu"
    resourceSliceCount: 1
  nodeName: node-1
  devices:
  - name: "cpu-0"
    attributes:
      resource.kubernetes.io/pcieRoot:
        list:
          string:
          - pci0000:01
          - pci0000:02
  - name: "cpu-1"
    attributes:
      resource.kubernetes.io/pcieRoot:
        list:
          string:
          - pci0000:03
          - pci0000:04
---
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: gpu
spec:
  driver: "gpu.example.com"
  pool:
    name: "gpu"
    resourceSliceCount: 1
  nodeName: node-1
  devices:
  - name: "gpu-0"
    attributes:
      # Assume this driver is a bit old that keeps exposing string for the attribute
      resource.kubernetes.io/pcieRoot:
        string: pci0000:01
---
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: nic
spec:
  driver: "nic.example.com"
  pool:
    name: "nic"
    resourceSliceCount: 1
  nodeName: node-1
  devices:
  - name: "nic-0"
    attributes:
      # Assume this driver is a bit old that keeps exposing string for the attribute
      resource.kubernetes.io/pcieRoot:
        string: pci0000:01
```

Then, user can create `ResourceClaim` resource which requests PCIe topology aligned CPU & GPU & NIC triple like below:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
spec:
  requests:
  - name: "gpu"
    exactly:
      deviceClassName: gpu.example.com
      count: 1
  - name: "nic"
    exactly:
      deviceClassName: nic.example.com
      count: 1
  - name: "cpu"
    exactly:
      deviceClassName: cpu.example.com
      count: 2
  constraints:
    # "gpu-0", "nic-0" and "cpu-0" above can match
    # because
    # - "pci0000:01" is common.
    # - string attribute can be treated as a single value list
  - requests: ["gpu", "nic", "cpu"]
    matchAttribute: k8s.io/pcieRoot
```

#### Story 2

T.B.D.

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
- Risk 1: Driver adoption lag
  - Mitigation: scalar is treated as single value list
- Risk 2: Scheduler performance overhead
  - bound lengths of the list-typed attribute values

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Go Type Definitions

#### `DeviceAttribute`

```go
type DeviceAttribute struct {
    ...
    // ListValue is a typed-list.
    //
    // +optional
    // +k8s:optional
    // +k8s:unionMember
    ListValue *ListAttribute `json:"list,omitempty"`
}

// ListAttribute defines typed-list value for device attributes
type ListAttribute struct {
    // IntValue is a list of numbers.
    //
    // +optional
    // +k8s:optional
    // +k8s:unionMember
    // +k8s:listType=atomic
    // +k8s:maxItems=64
    IntValue []int64 `json:"int,omitempty"`

    // BoolValue is a list of true/false values.
    //
    // +optional
    // +k8s:optional
    // +k8s:unionMember
    // +k8s:listType=atomic
    // +k8s:maxItems=64
    BoolValue []bool `json:"bool,omitempty"`

    // StringValue is a list of strings.
    // Each string must not be longer than 64 characters.
    //
    // +optional
    // +k8s:optional
    // +k8s:unionMember
    // +k8s:listType=atomic
    // +k8s:maxItems=64
    // +k8s:eachVal=+k8s:maxLength=64
    StringValue []string `json:"string,omitempty"`

    // VersionValue is a list of semantic versions according to semver.org spec 2.0.0.
    // Each version string must not be longer than 64 characters.
    //
    // +optional
    // +k8s:optional
    // +k8s:unionMember
    // +k8s:listType=atomic
    // +k8s:maxItems=64
    // +k8s:eachVal=+k8s:maxLength=64
    VersionValue []string `json:"version,omitempty"`
}
```

### Implementation (for evaluating constraints)

Since _non-empty intersection_ constraint is _monotonic_, we would not need updating [`Allocator.Allocate()` algorithm](https://github.com/kubernetes/kubernetes/blob/v1.34.2/staging/src/k8s.io/dynamic-resource-allocation/structured/internal/experimental/allocator_experimental.go#L135) and can keep using [`constraint` interface](https://github.com/kubernetes/kubernetes/blob/v1.34.2/staging/src/k8s.io/dynamic-resource-allocation/structured/internal/experimental/allocator_experimental.go#L703-L712). We will just extend the current [`matchAttributeConstraint`](https://github.com/kubernetes/kubernetes/blob/v1.34.2/staging/src/k8s.io/dynamic-resource-allocation/structured/internal/experimental/allocator_experimental.go#L721C6-L728) and [`distinctAttributeConstraint`](https://github.com/kubernetes/kubernetes/blob/v1.34.2/staging/src/k8s.io/dynamic-resource-allocation/structured/internal/experimental/constraint.go#L34-L41) instances. Or, we could introduce `constraint` instances for proposed modes (e.g., `nonEmptyIntersectionMatchAttributeConstraint`, etc.).

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

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

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
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

<!--
- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

#### Alpha

- Feature implemented behind a feature flag (`DRAListTypeAttributes`). The Feature gate is disabled by default.
- Documentation provided
- Initial unit, integration and e2e tests completed and enabled.

#### Beta

- Feature Gates are enabled by default.
- No major outstanding bugs.
- 1 example of real-world use case.
- Feedback collected from the community (developers and users) with adjustments provided, implemented and tested.

#### GA

- 2 examples of real-world use cases.
- Allowing time for feedback from developers and users.

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
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

For upgrade, existing `ResourceClaim`/`ResourceSlice` will still work as expected, as the new fields are missing there.

For downgrade, when there exists `ResourceClaim` with `matchSemantics`/`distinctSemantics` field or `ResourceSlice` with `list` type attribute values, there need to be caution. Although the already allocated claim does not affect, but when re-allocating, `matchSemantics`/`distinctSemantics` will be ignored. And, specified attribute in `matchAttribute`/`distinctAttribute` is `list` type, then allocation will be failed.

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

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DRAListTypeAttributes`
  - Components depending on the feature gate: kube-apiserver, kube-scheduler
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

Basically, no. Just introducing new API fields in `ResourceClaim` and `ResourceSlice` which does NOT change the default behavior when any device attribute type was NOT changed.

However, please note that `ResourceClaim`'s `matchAttribute/distinctAttribute` semantics are CHANGED when some device attribute type are changed from scalar to list.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes. When disabled, you can not create `DeviceAttribute` with `list`-type values. And, existing `list`-type attribute values are just ignored. But, if specified attribute in `matchAttribute`/`distinctAttribute` is `list` type, allocation will be failed.

###### What happens if we reenable the feature if it was previously rolled back?

`list`-type attribute values in `DeviceAttribute` and `matchSemantics`/`distinctAttribute` in `ResourceClaim` will be available again.

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

Yes, it will be covered by [Unit tests](#unit-tests).

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

No

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
Yes and no. It does add new fields, which increase the worst case size of `ResourceSlice` and `ResourceClaim` object. However, the increase size is bounded for most cases:
- `ResourceClaim`: linear to the number of constraints specified in the resource.
- `ResourceSlice`: linear to the number of devices defined in the resource. And, the number of list items is also bounded.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

Not expected. All the proposed constraints in this KEP are _monotonic_ constraint. Thus, worst case of computational complexity for device search is the same.

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

### Just support formatted string list instead of introducing `list` type

We could add pseudo list type support only for string type attribute (e.g. comma separated string).

- Pros:
  - Simple, no change in `DeviceAttribute`
- Cons:
  - String list only (Can't support list of int/version).
  - prone to mis-formatted string
  - extra parsing computation

### Introduce `matchSemantics/distinctSemantics` field for flexible/declarative match

Introduce `matchSemantics/distinctSemantics` fields into `constraints` field like this:

#### `matchSemantics` field

```yaml
kind: ResourceClaim
spec:
  constraints:
  - requests: [ "device1", "device2", "device3" ]
    matchAttribute: "resource.kubernetes.io/pcieRoot"

    # [NEW]
    # An optional field that defines customized "match" semantics over attribute values.
    # This field must not set when "distinctAttribute" is set
    matchSemantics:
      # mode specifies the "match" semantics
      # Identical (∀i,j, v_i = v_j):
      #   All the attribute values among candidate devices are identical,
      #   supporting both list-order-sensitive and set-equivalence comparisons via `listMode`.
      # NonEmptyIntersection (|∩ v_i| >= k (>=1)): 
      #   The intersection (as a set) of list values among candidate devices is non-empty.
      #   The required intersection size could be configurable via `minSize`.
      # For future possible cases:
      # - CommonPrefix/Suffix with customizable length
      # - Identical for aggregated values of the list items (min/max/sum/length)
      mode: Identical | NonEmptyIntersection

      options:
        nonEmptyIntersection:
          # if true, implicit cast from scalar to list will be performed. The default is false.
          coerceScalarToList: true | false
          # minSize specifies the minimum size of the intersection to evaluate as true.
          # Default is 1. The value must be positive integer.
          minSize: 1
       identical:
          coerceScalarToList: true | false    # common option
          # listMode specified the equality as a set(order/duplicates are ignored) or list (order significant). Default is List
          listMode: List | Set
```
Examples of match semantics mode:

| attribute values | `Identical` | `NonEmptyIntersection`<br/>(`coerceScalarToList=true`) |
|:--:|:--:|:--:|
| `d1="a"`, `d2="b"` | `false` | `false` |
| `d1=["a", "b"]` , `d2=["b", "a"]` | `false`(`listMode: List`)<br/>`true`(`listMode: Set`) | `true`<br/>(`d1 ∩ d2 = {"a", "b"}`) |
| `d1=["a", "b"]` , `d2=["a", "c"]`| `false` | `true`<br/>(`d1 ∩ d2 = {"a"}`) |
| `d1=["a", "b"]` , `d1=["c", "d"]` | `false` | `false`<br/>(`d1 ∩ d2 = ∅`) |

#### `distinctSemantics

```yaml
kind: ResourceClaim
spec:
  constraints:
  - requests: [ "device1", "device2", "device3" ]
    distinctAttribute: "resource.kubernetes.io/numaNode" # note: this is imaginary attribute.

    # [NEW]
    # an optional field that defines customized "distinct" semantics over attribute values
    # this field must not set when "matchAttribute" is set
    distinctSemantics:
      # mode specifies the "distinct" semantics
      # `AllDistinct`:
      #   All the values are distinct, supporting both list-order-sensitive and set-equivalence comparisons via `listMode`.
      #   (i.e. ∀i,j s.t. i ≠ j, v_i != v_j), 
      # `EmptyIntersection`:
      #   The intersection (as a set) of all the list values among candidate devices is empty. (i.e. ∩ v_k = ∅ )
      # `PairwiseDisjoint`:
      #   Every pair of the list values (as a set) of candidate devices is disjoint (i.e. completely no overlap).
      #   (i.e. ∀i,j s.t. i ≠ j, v_i ∩ v_j = ∅),
      # For future possible cases:
      # - NoCommonPrefix/Suffix, PairwiseDisjointPrefix/Suffix with customizable length
      # - AllDistinct for aggregated values of the list items (min/max/sum/length)
      mode: AllDistinct | EmptyIntersection | PairwiseDisjoint

      options:
        allDistinct:
          coerceScalarToList: true | false    # common option
          # listMode specified the equality as a set(order/duplicates are ignored) or list (order significant). Default is List
          listMode: List | Set
        emptyIntersection:
          coerceScalarToList: true | false    # common option
        pairwiseDisjoint:
          coerceScalarToList: true | false    # common option
```

Examples of distinct semantics mode:

| attribute values | `AllDistinct` | `PairwiseDistinct`<br/>(`coerceScalarToList=true`) | `EmptyIntersection`<br/>(`coerceScalarToList=true`) |
|:--:|:--:|:--:|:--:|
| `d1="a"`, `d2="b"` | `false` | `false` | `false` |
| `d1=["a", "b"]` , `d2=["b", "a"]` | `true`(`listMode: List`)<br/>`false`(`listMode: Set`) | `false`<br/>(`d1 ∩ d2={"a","b"}`) | `false`<br/>(`∩dk={"a","b"}`) |
| `d1=["a", "b"]` , `d2=["a", "c"]`, `d3=["a", "d"]` | `true` | `false`<br/>(`di ∩ dj = {"a"} ≠ ∅`) | `false` <br/>(`∩ dk = {"a"} ≠ ∅`) |
| `d1=["a", "b"]` , `d2=["b", "c"]`, `d3=["c", "a"]` | `true` | `false`<br/>(`di ∩ dj ≠ ∅`) | `true`<br/>(`∩ dk = ∅`) |
| `d1=["a", "b"]` , `d2=["c", "d"]`, `d3=["e", "f"]` | `true` | `true`<br/>(`di ∩ dj = ∅`) | `true`<br/>(`∩ dk = ∅`) |

#### Pros/Cons

- Pros:
  - Flexible
  - Declarative
  - Extensible
- Cons:
  - Too much complex even we don't have use-cases to introduce the complexity


### Unified `semantics` field instead of `matchSemantics`/`distinctSemantics`

We can consider unified `semantics` field for both `matchAttribute`/`distinctAttribute` like below:

```yaml
semantics:
  mode: NonEmptyIntersection | EmptyIntersection | Identical | AllDistinct | PairwiseDisjoint
```

- Pros:
  - Simple
- Cons:
  - Confusing which mode is valid for `matchAttribute` or `distinctAttribute`
  - Extra validation logics

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
