# KEP-4369: Allow special characters in environment variables

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
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
    - [Upgrade](#upgrade)
    - [Downgrade](#downgrade)
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Allows all printable ASCII characters except "=" to be set as environment variables, the range of printable ASCII characters is 32-126.

## Motivation

Kubernetes should not restrict which environment variable names can be used, because it has no way of knowing what the application may need, and people can't always choose their own variable names, which may limit the adoption of Kubernetes.

### Goals

* Allows users to set all ASCII characters with serial numbers in the range of 32-126 except "=" as environment variables.

### Non-Goals

## Proposal

* Implements relaxed validation at the top level validation method when validating API create requests, all ASCII characters in the range 32-126 except "=" can be verified.
* Allow users to set `Configmap` keys and secret keys outside the `C_IDENTIFIER` scope as environment variables using EnvFrom
* Document rules for setting environment variables.

### User Stories (Optional)

#### Story 1

I am a .NET Core development engineer, .Net Core applications are using ":" when working with application settings loaded from appsettings.json file. When running .net core app in containers typically overwrite this settings by specifying environmental variable.
such as: 
`"Logging": { "IncludeScopes": false, "LogLevel": { "Default": "Warning" } }`    
override like this `-e Logging:LogLevel:Default=Debug`    

### Risks and Mitigations

Relaxed validation can break upgrade and rollback scenarios, but our use of feature gate to control whether it's enabled or not will make it a manageable risk, with the user having the autonomy to choose whether or not to enable it.

## Design Details

- A feature gate name `RelaxedEnvironmentVariableValidation` controlling the loosening of the envvar name validation, initially in alpha state and defaulting to false

- Two sets of validation logic for envvar names:
  
  * Strict validation
    * Strict validation follows the current design, which only allows envvar names passed the regular expression `[-._a-zA-Z][-._a-zA-Z0-9]*`.
  
  - Relaxed validation
    * Relaxed verification allows all ASCII characters in the range 32-126 as envvar name, and its regular expression is `^[ -<>-~]+$`,  matches a string containing ASCII characters from `space` to `<` and from `>` to `~`, ignore `=`, and has a length of at least 1.
  
- Everywhere we validate envvar names in API objects, plumbing a parameter whether we want the strict or relaxed validation
  - At the top level validation method when validating API create requests, use the strict validation if the feature gate is off
  - At the top level validation method when validating API update requests, use the strict validation if the feature gate is off and the old object passes strict envvar name validation

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

Currently coverages:

* pkg/apis/core/validation/validation_test.go: `2023-12-21` - `83.9%`
* pkg/kubelet/kubelet_pods_test.go: `2023-12-21` - `67.2%`
* staging/src/k8s.io/apimachinery/pkg/util/validation/validation_test.go: `2023-12-21` - `94.8%`

These tests will be added:

* New tests will be added to ensure environment variable fields can be correctly validated `pkg/apis/core/validation/validation_test.go`
* Add a new test that sets special character environment variables for pods in a given namespace `pkg/kubelet/kubelet_pods_test.go`
* A new test will be added to ensure that the environment variable name field is valid `staging/src/k8s.io/apimachinery/pkg/util/validation/validation_test.go`

##### Integration tests

- N/A

##### e2e tests

* Add a test to `test/e2e/common/node/configmap.go` to test that the special characters in configmap are consumed by the environment variable.

* Add a test to `test/e2e/common/node/secret.go` to test that the special characters in secret are consumed by the environment variable.

* Add a test to `test/e2e/common/node/expansion` to test environment variable can contain special characters.

### Graduation Criteria

#### Alpha

- Created the feature gate and implement the feature, disabled by default.
- Add unit and e2e tests for the feature.

#### Beta

- Solicit feedback from the Alpha.
- Ensure tests are stable and passing.
- Add monitor for pods that fail due to using enhancements.

#### GA

- Ensure that the time range of the beta version can cover the version skew of all components.
- Add troubleshooting details on how to deal with incompatible kubelet/CRI implementations based on issues found in beta releases.

### Upgrade / Downgrade Strategy

#### Upgrade

Environment variables previously set by the user will not change. To use this enhancement, users need to enable the feature gate

#### Downgrade

users need to reset their environment variables for special characters to normal characters.

### Version Skew Strategy

kube-apiserver will need to enable feature gates to use this feature.

If kube-apiserver is not enabled feature gate will use strict validation.

If the feature gate is disabled and the existing object passes strict validation, strict validation on update will be used.


## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: RelaxedEnvironmentVariableValidation
  - Components depending on the feature gate: kube-apiserver

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

If close the feature gate, already running workloads will not be affected in any way, 
but cannot create workloads that use special characters as environment variables.

###### What happens if we reenable the feature if it was previously rolled back?

The feature should continue to work just fine.

###### Are there any tests for feature enablement/disablement?

Yes.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

When a feature gate is closed, already running workloads are not affected in any way, but update fields for workload will cause the workload to fail.

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

- We will investigate in the beta version how to monitor kubelet/CRI implementations could fail on pods using this enhancement.

### Dependencies

N/A

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

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

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

\- 2023-12-21: Initial draft KEP

## Drawbacks

If the envvar name character set is extended,  all the things currently consuming and using envvar names from the API will have an impact and may break or be unsafe.

For example: 

* If a third party uses an envvar name as a filename and assumes that it is currently safe, then if it contains characters that cannot be used as a filename (like `:`) or characters that break the assumptions of a flat directory structure (like `/`), then unexpected results will occur.

## Alternatives

- do nothing (leave it as-is)

- relax the rule, but with a long beta period where the existing rule remains the default.
  Ensure that the beta period doesn't end until ValidatingAdmissionPolicy is GA and has been for 2 minor releases.
  *Clearly* document how to use a ValidatingAdmissionPolicy to get behavior equivalent to the legacy checking,
  and signpost people to these docs when graduating the looser validation to be the Kubernetes default.

- define a label or annotation for each namespace that controls how Pod environment variables are validated in that namespace

- [more complex!]
  add an API kind to specify the validation rules for Pods

  Create a new API kind, eg PodValidationRule. It's **namespaced**. Within the `.spec` of each object, define:

  - a Pod selector
  - an optional CEL validation rule for environment variable keys
  - an optional CEL validation rule for environment variable values

  If any of the selected validation rules don't pass for a Pod, reject it at admission time. Set up a defaulting
  mechanism to
  Also, define how Pod templates interact with this new API (eg: you get a `Warning:` when you create
  a Deployment where the PodTemplate inside the Deployment wouldn't pass validation)

## Infrastructure Needed (Optional)
