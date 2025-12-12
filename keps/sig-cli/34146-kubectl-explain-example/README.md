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
# KEP-34146: kubectl example - kubectl explain example: practical output that can be applied | trialed by new user to advanced

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
  - [Basic Usage](#basic-usage)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
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
- [Future Work](#future-work)
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

This KEP proposes adding a new kubectl subcommand `kubectl example` that provides generic seed YAML for different resources, complementing the existing `kubectl explain` command. Think of it as `curl cht.sh/kubectl` but distributed at the CLI level.

The idea is to provide a `kubectl explain` then `kubectl example` flow, where users can get detailed explanations and then practical YAML examples.

## Motivation

Users often need practical, applicable YAML examples for Kubernetes resources. While `kubectl explain` provides detailed schema information, it doesn't give users ready-to-use YAML snippets. This creates a gap where users must manually construct YAML from documentation, which can be error-prone and time-consuming, especially for beginners.

This KEP addresses that gap by introducing `kubectl example`, which outputs generic seed YAML for resources. The examples are designed to be:

- **Immediately applicable**: Can be piped directly to `kubectl apply` for testing.
- **Best-practice oriented**: Include common configurations like resource limits, labels, and annotations.
- **Educational**: Serve as templates that users can modify for their needs.
- **Comprehensive**: Cover a wide range of Kubernetes resources.

For instance:

- `kubectl explain pod` -- detailed explanation of resource

- `kubectl example pod` -- generic YAML output for a standard pod with linux - alpine image

This enhances the user experience, especially for new users learning Kubernetes, by providing immediate practical output that can be applied or trialed.

Additionally, this could socialize further the use of `kubectl get --raw` on APIs and potentially automate ingestion of that to explain what the API controls in flight within a cluster at a version.

### Goals

1. Provide a new `kubectl example` subcommand that outputs generic seed YAML for Kubernetes resources.
2. Complement `kubectl explain` by offering practical, applicable examples.
3. Support common resources with sensible defaults.
4. Potentially integrate with `kubectl get --raw` to enhance API understanding.

### Non-Goals

1. Replace or modify `kubectl explain`.
2. Provide exhaustive examples for all possible configurations.
3. Generate examples dynamically from cluster state.


## Proposal

### Basic Usage

The following user experience should be possible with `kubectl example`:

```shell
kubectl example pod
```

This would output a generic YAML for a Pod resource, e.g.:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: example-pod
spec:
  containers:
  - name: example-container
    image: alpine:latest
    command: ["sleep", "3600"]
    resources:
      requests:
        memory: "64Mi"
        cpu: "250m"
      limits:
        memory: "128Mi"
        cpu: "500m"
```

For a PersistentVolumeClaim:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: example-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
```

Similarly for other resources like deployments, services, etc.

### Advanced Usage

- `kubectl example deployment --replicas=3` - Generate example with custom parameters
- `kubectl example --list` - List all available example resources
- `kubectl example pod | kubectl apply -f -` - Apply the example directly

### Risks and Mitigations

#### No Examples Available for a Resource

##### Risk

The requested resource does not have a predefined example.

##### Mitigation

Return an error message suggesting to use `kubectl explain` for schema information or check available examples with `kubectl example --list`.

#### Outdated Examples

##### Risk

Examples may not reflect the latest best practices or API changes.

##### Mitigation

Examples will be maintained as part of kubectl releases, with community contributions encouraged. Version-specific examples can be added if needed.


## Design Details

The new `kubectl example` command will be implemented as a new subcommand in kubectl, similar to `kubectl explain`.

High-level Approach:

1. User types `kubectl example <resource>`
2. kubectl resolves the resource type using discovery (similar to `kubectl explain`)
3. kubectl looks up a predefined YAML template for that resource
4. kubectl outputs the YAML to stdout

Templates will be hardcoded in the kubectl binary for common resources like pods, deployments, services, etc. Examples should use sensible defaults, such as alpine images for containers.

For extensibility, future versions could allow loading examples from files or online repositories, but initially, examples will be built-in.

### Example Storage and Management

Examples will be stored as Go string literals or embedded files in the kubectl codebase. Each example will include:

- Valid YAML structure
- Sensible default values
- Comments explaining key sections
- Resource limits and requests where applicable
- Common labels and annotations

### Adding New Examples

To add a new example:

1. Create the YAML template
2. Add it to the examples map in the code
3. Update tests
4. Update documentation

### Parameterization

Future enhancements could support basic parameterization, such as custom names, replica counts, or image versions, using Go templates or simple string replacement.

### Test Plan

##### Prerequisite testing updates

None required.

##### Unit tests

Unit tests will verify that the correct YAML is output for supported resources and appropriate errors for unsupported ones.

##### Integration tests

Integration tests will ensure the command integrates well with kubectl's existing infrastructure, such as resource discovery.

##### e2e tests

E2E tests will validate that the output YAML can be applied to a cluster (e.g., `kubectl example pod | kubectl apply -f -` creates a running pod).

### Graduation Criteria

#### Alpha

- Basic `kubectl example` command implemented with examples for core resources (pod, deployment, service).
- Unit and integration tests in place.

#### Beta

- Expanded set of examples for more resources.
- User feedback incorporated.
- E2E tests passing.

#### GA

- Comprehensive examples for commonly used resources.
- Documentation updated.
- No breaking changes.

### Upgrade / Downgrade Strategy

N/A - This is a new command, no upgrades needed.

### Version Skew Strategy

The command relies on kubectl's resource discovery, which should work across versions. Examples are static, so no skew issues.

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

Unit tests will verify that the correct YAML is output for supported resources, appropriate errors for unsupported ones, and that the YAML is valid.

##### Integration tests

Integration tests will ensure the command integrates well with kubectl's existing infrastructure, such as resource discovery, and that examples are consistent with cluster capabilities.

##### e2e tests

E2E tests will validate that the output YAML can be applied to a cluster successfully (e.g., `kubectl example pod | kubectl apply -f -` creates a running pod), and that examples work across different cluster configurations.

### Graduation Criteria

#### Alpha

- Basic `kubectl example` command implemented with examples for core resources (pod, deployment, service, persistentvolumeclaim).
- Unit and integration tests in place.
- Command available in kubectl builds.

#### Beta

- Expanded set of examples for more resources (configmap, secret, job, etc.).
- User feedback incorporated from alpha usage.
- E2E tests passing in CI.
- Documentation updated with examples.

#### GA

- Comprehensive examples for commonly used resources.
- Examples validated against multiple Kubernetes versions.
- No breaking changes in output format.
- Feature promoted as stable in kubectl documentation.

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

N/A

### Version Skew Strategy

The command relies on kubectl's resource discovery, which should work across versions. Examples are static YAML templates, so no version skew issues with the output itself. However, the applicability of examples may vary based on cluster capabilities (e.g., newer API versions). The command will use the latest available API versions for resource discovery.

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

###### How can this feature be enabled / disabled in a live cluster?

- [x] Other
  - Describe the mechanism: This is a new kubectl subcommand. It is enabled by building kubectl with the new code. No feature gate.
  - Will enabling / disabling the feature require downtime of the control plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning of a node? No

###### Does enabling the feature change any default behavior?

No, it's a new command.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by using an older version of kubectl without the command.

###### What happens if we reenable the feature if it was previously rolled back?

Normal operation.

###### Are there any tests for feature enablement/disablement?

Unit tests for the command presence.


### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

No, this is a new CLI command. No impact on workloads.

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

N/A

###### How can someone using this feature know that it is working for their instance?

Run `kubectl example pod` and verify YAML output.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Other (treat as last resort)
  - Details: N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

None

### Scalability

###### Will enabling / using this feature result in any new API calls?

Potentially, also happy to make it a subcommand of explain if that's logically neater:
```
kubectl explain example pod
```

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The command doesn't require API server access, as examples are static.
