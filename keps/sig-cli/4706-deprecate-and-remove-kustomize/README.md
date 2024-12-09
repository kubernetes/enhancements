# KEP-4706: Deprecate and remove kustomize from kubectl

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Users relying on <code>kubectl kustomize</code>](#users-relying-on-kubectl-kustomize)
    - [Users relying on <code>--kustomize</code> flag](#users-relying-on---kustomize-flag)
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
    - [Deprecation](#deprecation)
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
  - [Help with kustomize maintenance](#help-with-kustomize-maintenance)
  - [Minimize kustomize dependencies](#minimize-kustomize-dependencies)
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

Deprecate and remove kustomize from kubectl. This will allow both tools to
be developed and maintained separately.

**NOTE**: After a discussion in sig-architecture meeting on Oct 17th, 2024 (refer to the [recording](https://www.youtube.com/watch?v=cQzS38D0asM)
and [meeting notes](https://docs.google.com/document/d/1BlmHq5uPyBUDlppYqAAzslVbAO8hilgjqZUTaNXUhKM/edit?tab=t.0#bookmark=kix.qolek4jkm5m))
it was decided not to pursue this topic further, and retain kustomize as part of
kubectl. The primary reason for this decision was the widespread adoption of
the tool by the community. Moving forward with the proposed enhancement could
potentially disrupt its established usage and jeopardize users trust.

## Motivation

Kustomize was [brought into kubectl](https://github.com/kubernetes/kubernetes/pull/70875)
shortly after its [initial release](https://github.com/kubernetes-sigs/kustomize/commit/e57010bcf641738738591eb48b4977843a494893).
The main motivation back then was to expand declarative support for kubernetes objects.
Over the past decade of kubernetes existence multiple, various tools for customization
and templating has been development. Given that, current kubectl maintainers
feel that promoting one tool over the other should not be the role of the project.

Moreover, decoupling both projects will allow them to move at separate pace. The
current kubernetes release cycle doesn't match that of kustomize, oftentimes
risking users of kubectl to work with outdated version of kustomize.

Lastly, some of the kustomize dependencies has already been problematic to the
core kubernetes project, so removing kustomize will allow us to minimize the
dependency graph and the size of kubectl binary.


### Goals

* Deprecate and remove kustomize from kubectl.

### Non-Goals

* Change kubectl or kustomize functionality.
* Change kubectl or kustomize release cycle.

## Proposal

Following the [official kubernetes deprecation policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-a-flag-or-cli)
it is required to keep a generally available element of user-facing component to
be available at minimum 12 months or 2 releases. We're proposing to proceed with
the deprecation and removal of kustomize in the following stages:

1. v1.31 (alpha) - announce the deprecation of `kubectl kustomize` subcommand,
   `--kustomize` flag from all the subcommands using it, and from `kubectl version`
   output.
2. v1.31 - v1.33 - gather feedback from users, and publish kustomize in [krew index](https://github.com/kubernetes-sigs/krew-index).
3. v1.34 - v1.35 (beta) - disable kustomize by default from kubectl.
4. Entirely remove kustomize in v1.36 (planned as first release of 2026).

### Risks and Mitigations

#### Users relying on `kubectl kustomize`

There may be a significant group of users relying on the built-in kustomize.
Those users should switch to using [kubectl plugins](../2379-kubectl-plugins/README.md),
which allows kubectl users to invoke any `kubectl-*` prefixed binary as a native
kubectl command. Additionally, we will publish kustomize in [krew index](https://github.com/kubernetes-sigs/krew-index),
which provides a central repository for kubectl plugins.

#### Users relying on `--kustomize` flag

There may be a significant group of users relying on the built-in kustomize
inside one of the subcommands. Those users will be advised to use `kustomize build`
and pipe the output to appropriate command instead, as part of the deprecation warning.

## Design Details

In version 1.31 of kubectl we will implement warnings, such that all users of
`kubectl kustomize` will be informed that the subcommand is deprecated and they
will be directed to use standalone kustomize, or switch to kustomize plugin.
Users of `--kustomize` flag will be informed that the flag is deprecated,
and will be directed to use standalone kustomize.

The above warnings will be implemented behind a `KUBECTL_LEGACY_KUSTOMIZE` environment
variable which starting from v1.31 will be on by default. Users interested in
trying out kubectl without kustomize will be able to disable the aforementioned
environment variable.

In version 1.34 we will disable the environment variable, disabling access to kustomize
by default in kubectl. At this point in time users will still have the option to
enable this mechanism back using `KUBECTL_LEGACY_KUSTOMIZE`.

In version 1.36 lock `KUBECTL_LEGACY_KUSTOMIZE` to off by default, but leaving
the kustomize code base in kubectl for one more release.

Finally, in version 1.37 entirely drop kustomize from kubectl.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `k8s.io/cli-runtime/pkg/genericclioptions`: `2024-06-10` - `22.5%`
- `k8s.io/cli-runtime/pkg/resource`:          `2024-06-10` - `71.8%`
- `k8s.io/kubectl/pkg/cmd/delete`:            `2024-06-10` - `77.6%`
- `k8s.io/kubectl/pkg/cmd/kustomize`:         `2024-06-10` - `0%`

##### Integration tests

- `run_kubectl_version_tests`:                        https://testgrid.k8s.io/sig-testing-canaries#pull-kubernetes-integration-go-canary
- `run_kubectl_apply_tests`:                          https://testgrid.k8s.io/sig-testing-canaries#pull-kubernetes-integration-go-canary
- `run_kubectl_create_kustomization_directory_tests`: https://testgrid.k8s.io/sig-testing-canaries#pull-kubernetes-integration-go-canary

##### e2e tests

None available.

### Graduation Criteria

#### Alpha

- Warning users that kustomize subcommand and flag is deprecated.
- Feature implemented behind an environment variable `KUBECTL_LEGACY_KUSTOMIZE` turned on by default.
- Gather feedback from users

#### Beta

- Turn `KUBECTL_LEGACY_KUSTOMIZE` environment variable off by default.
- Publish kustomize in [krew index](https://github.com/kubernetes-sigs/krew-index).
- Gather feedback from users

#### GA

- Lock `KUBECTL_LEGACY_KUSTOMIZE` environment variable to off by default.

#### Deprecation

- Remove kustomize from kubectl code base.

### Upgrade / Downgrade Strategy

Initially users will have the ability to test kubectl with and without kustomize
setting `KUBECTL_LEGACY_KUSTOMIZE` to on or off. Eventually, once the functionality
is removed users will have the option to switch to kustomize plugin (preferred),
or stick with older version of kubectl as long as the support window allows.

### Version Skew Strategy

kubectl supports +/- one version skew. The deprecation and later removal
of kustomize will not affect the provided support window.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

Users will have an option to test kubectl with and without kustomize by setting
`KUBECTL_LEGACY_KUSTOMIZE` environment variable. Initially, by default this option
will be on.

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [X] Other
  - Describe the mechanism:
    `KUBECTL_LEGACY_KUSTOMIZE` environment variable
  - Will enabling / disabling the feature require downtime of the control
    plane?
    No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
    No.

###### Does enabling the feature change any default behavior?

Disabling `KUBECTL_LEGACY_KUSTOMIZE` will remove access to `kubectl kustomize`,
`--kustomize` flag from appropriate subcommands and from `kubectl version` output.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, users will have `KUBECTL_LEGACY_KUSTOMIZE` environment variable to do just that.

###### What happens if we reenable the feature if it was previously rolled back?

Enabling `KUBECTL_LEGACY_KUSTOMIZE` will keep access to `kubectl kustomize`,
`--kustomize` in the appropriate subcommands and in the `kubectl version` output.

###### Are there any tests for feature enablement/disablement?

We plan to add integration tests (in `test/cmd`) exercising the functionality
with the environment variable turn on and off.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

We are deprecating `kubectl kustomize` subcommand, `--kustomize` flag from
appropriate subcommands, and from `kubectl version` output.


### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

N/A

###### How can someone using this feature know that it is working for their instance?

N/A

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

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

This is client-side functionality, so it is not affected by API server and/or etcd availability.

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

2024-06-10: Initial version of the document

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

### Help with kustomize maintenance

[Kustomize](https://github.com/kubernetes-sigs/kustomize/) is pretty significant,
the project has been around for [6 years now](https://github.com/kubernetes-sigs/kustomize/commit/e57010bcf641738738591eb48b4977843a494893).
Unfortunately, in the recent years the maintainers turnaround has been pretty
significant. Currently, the project is being overlooked by two primary contributors:
[Yugo](https://github.com/koba1t) and [Varsha](https://github.com/varshaprasad96).


### Minimize kustomize dependencies

The project over the years built a lot of functionality current users rely on.
The option to drop some of that, to be able to limit the scope of the dependencies
is thus limited by which functionality can be dropped. Further more, our goal is
not to limit the further development of kustomize, but only decoupling it from
kubectl.

## Infrastructure Needed (Optional)

N/A
