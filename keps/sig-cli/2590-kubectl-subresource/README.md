# KEP-2590: Kubectl Subresource Support

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
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Subresource support](#subresource-support)
  - [Table printer](#table-printer)
  - [Test Plan](#test-plan)
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
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This document proposes adding a new `--subresource` flag to the following kubectl
subcommands: `get`, `patch`, `edit`, `apply` and `replace`. The goal of this flag
is to simplify the process of fetching and updating `status`, `scale` and `resize`
subresources.

## Motivation

Today while testing or debugging, fetching subresources (like `status`) of API objects via kubectl
involves using `kubectl --raw`. Patching subresources using kubectl is not possible at all and
requires using `curl` directly. This method is very cumbersome and not user-friendly.

This enhancement adds subresources as a first class option in kubectl to allow users
to work with the API in a generic fashion.

### Goals

- Add a new flag `--subresource=[subresource-name]` to `get`, `patch`, `edit`, `apply`
and `replace` kubectl commands to allow fetching and updating `status`, `scale` and `resize`
subresources for all resources (built-in and custom resources) that support these.
- Display pretty printed table columns for the `status` (uses same columns as the main resource)
and `scale` subresources.

### Non-Goals

- Support subresources other than `status` and `scale`.
- Allow specifying `additionalPrinterColumns` for CRDs for the status subresource.

## Proposal

kubectl commands like `get`, `patch`, `edit`, `apply` and `replace` will now contain a
new flag `--subresource=[subresource-name]` which will allow fetching and updating
`status`, `scale` and `resize` subresources for all API resources.

Note that the API contract against the subresource is identical to a full resource.
Therefore updating the status subresource to hold new value which could potentially
be reconciled by a controller to a different value is *expected behavior*.

If `--subresource` flag is used for a resource that doesn't support the subresource,
a `NotFound` error will be returned.


### User Stories (Optional)

#### Story 1

```shell
# for built-in types
# a `get` on a status subresource will return identical information
# to that of a full resource
$ kubectl get deployment nginx-deployment --subresource=status
NAME               READY   UP-TO-DATE   AVAILABLE   AGE
nginx-deployment   3/3     3            3           43s

$ kubectl get deployment nginx-deployment --subresource=scale
NAME               DESIREDREPLICAS   AVAILABLEREPLICAS
nginx-deployment   3                 3

# for CRDS
# a `get` on a status subresource will return identical information
# to that of a full resource
$ kubectl get crontab cron --subresource=status
NAME   SPEC          REPLICAS   AGE
cron   * * * * */5   3          4m52s

$ kubectl get crontab cron --subresource=scale
NAME   DESIREDREPLICAS   AVAILABLEREPLICAS
cron   3                 0
```

Invalid subresources:

```shell
$ kubectl get pod nginx-deployment-66b6c48dd5-dv6gl --subresource=logs
error: --subresource must be one of [status scale], not "logs"

$ kubectl get pod nginx-deployment-66b6c48dd5-dv6gl --subresource=scale
Error from server (NotFound): the server could not find the requested resource
```

#### Story 2

```shell
# For built-in types
# update spec.replicas through scale subresource
$ kubectl patch deployment nginx-deployment --subresource='scale' --type='merge' -p '{"spec":{"replicas":2}}'
scale.autoscaling/nginx-deployment patched

# spec.replicas is updated for the main resource
$ kubectl get deploy nginx-deployment
NAME               READY   UP-TO-DATE   AVAILABLE   AGE
nginx-deployment   2/2     2            2           4m

# For CRDs
$ kubectl patch crontabs cron --subresource='status' --type='merge' -p '{"status":{"replicas":2}}'
crontab.stable.example.com/cron patched

$ kubectl get crontab cron --subresource=scale
NAME   DESIREDREPLICAS   AVAILABLEREPLICAS
cron   3                 2
```

### Risks and Mitigations

This feature adds a new flag which will be validated like any other flag for a limited
set of inputs. The remaining flags passed to every command will be validated as
before.

## Design Details

### Subresource support

A new field `Subresource` and method `Subresource`/`WithSubresource` is added to
the [builder], [helper] and [visitor] code to use the API object at the subresource path.

[builder]: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/cli-runtime/pkg/resource/builder.go
[helper]: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/cli-runtime/pkg/resource/helper.go
[visitor]: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/cli-runtime/pkg/resource/visitor.go

For each relevant kubectl command, a new flag `--subresource` is added. If the flag is specified,
the value is also validated to ensure that it is a valid subresource.

If the subresource does not exist for an API resource, a `NotFound` error is returned.

### Table printer

To support table view for subresources using `kubectl get`, table convertor support is added to
the scale and status subresources for built-in and custom resource types.

For built-in types, `StatusStore` and `ScaleStore` are updated to implement the `TableConvertor` interface.
`StatusStore` uses the same columns as the main resource object.

The following column definitions for the `Scale` object are added to [printers.go] to support the scale subresource:
- `Available Replicas` uses the json path `.status.replicas` of autoscalingv1.Scale object
- `Desired Replicas` uses the json path of `.spec.replicas` of autoscalingv1.Scale object

For custom resources:
- the status subresource uses the same columns as defined for the full resource, i.e., `additionalPrinterColumns` defined in the CRD.
- the scale subresource follows the same column definitions as the built-in types, and are defined in [helpers.go].

[printers.go]: https://github.com/kubernetes/kubernetes/blob/master/pkg/printers/internalversion/printers.go#L88
[helpers.go]: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/helpers.go

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Unit tests

- `k8s.io/kubernetes/pkg/printers/internalversion`: `2025-01-23` - 73.5
- `k8s.io/cli-runtime/pkg/resource`: `2025-01-23` - 71.8
- `k8s.io/kubectl/pkg/cmd/apply`: `2025-01-23` - 82
- `k8s.io/kubectl/pkg/cmd/edit`: `2025-01-23` - 100
- `k8s.io/kubectl/pkg/cmd/get`: `2025-01-23` - 80.8
- `k8s.io/kubectl/pkg/cmd/patch`: `2025-01-23` - 56.4
- `k8s.io/kubectl/pkg/cmd/replace`: `2025-01-23` - 63.8

##### Integration tests

- [kubectl get](https://github.com/kubernetes/kubernetes/blob/00fa8f119077da3c96090aa5efc5dfc9c5a78977/test/cmd/get.sh#L178-L184): https://storage.googleapis.com/k8s-triage/index.html?pr=1&job=pull-kubernetes-integration&test=test-cmd%3A%20run_kubectl_get_tests
- [kubectl apply](https://github.com/kubernetes/kubernetes/blob/00fa8f119077da3c96090aa5efc5dfc9c5a78977/test/cmd/apply.sh#L417): https://storage.googleapis.com/k8s-triage/index.html?pr=1&job=pull-kubernetes-integration&test=test-cmd%3A%20run_kubectl_server_side_apply_tests
- [TestGetSubresourcesAsTables](https://github.com/kubernetes/kubernetes/blob/00fa8f119077da3c96090aa5efc5dfc9c5a78977/test/integration/apiserver/apiserver_test.go#L1458-L1678): https://storage.googleapis.com/k8s-triage/index.html?pr=1&job=pull-kubernetes-integration&test=TestGetSubresourcesAsTables
- [TestGetScaleSubresourceAsTableForAllBuiltins](https://github.com/kubernetes/kubernetes/blob/ed9572d9c7733602de43979caf886fd4092a7b0f/test/integration/apiserver/apiserver_test.go#L1681-L1876): https://storage.googleapis.com/k8s-triage/index.html?pr=1&job=pull-kubernetes-integration&test=TestGetScaleSubresourceAsTableForAllBuiltins

##### e2e tests

- [kubectl subresource flag](https://github.com/kubernetes/kubernetes/blob/00fa8f119077da3c96090aa5efc5dfc9c5a78977/test/e2e/kubectl/kubectl.go#L2090-L2118): https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-e2e-gci-gce&include-filter-by-regex=kubectl%20subresource

### Graduation Criteria

#### Alpha

- Add the `--subresource` flag to `get`, `patch`, `edit` and `replace` subcommands.
- Unit tests and integration tests are added.

#### Beta

- Gather feedback from users.
- e2e tests are added.
- Add the `--subresource` flag to `apply` subcommand.

#### GA

Since v1.27 when the feature moved to beta, there have been no reported bugs concerning this feature.
In fact, it is reassuring to see the community use this feature quite commonly such as in bug reports:
https://github.com/kubernetes/kubernetes/issues/116311

Seeing this and given our added unit, integration and e2e tests gives us the confidence to graduate to stable.

### Upgrade / Downgrade Strategy

See [Version Skew Strategy](#version-skew-strategy).

### Version Skew Strategy

The [kube-apiserver functionality](https://github.com/kubernetes/kubernetes/pull/103516)
required for the `--subresource` flag to work correctly was introduced in Kubernetes v1.24.
The current release (v1.33) exceeds the [supported version skew policy](https://kubernetes.io/releases/version-skew-policy/).
Therefore, there are no requirements for planning the upgrade or downgrade process.
needs to be completed.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
- [x] Other
    - Describe the mechanism: A new flag for kubectl commands.
      For the alpha stage, description will be added to explicitly call
      out this flag as an alpha feature.
    - Will enabling / disabling the feature require downtime of the control
      plane? No, disabling the feature would be a client behavior.
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node?
      No, disabling the feature would be a client behavior.

###### Does enabling the feature change any default behavior?

While the feature now updates kubectl's behavior to allow updating subresources,
it is gated by the `--subresource` flag so it does not change kubectl's default
behavior.

Subresources can also be updated using `curl` today so this feature only
provides a consistent way to use the API via kubectl, but does not allow additional
changes to the API that are not possible today.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, rolling back to a previous version of `kubectl` will remove support
for the `--subresource` flag and the ability to update subresources via kubectl.

However, this does not "lock" us to any changes to the subresources that were made
when the feature was enabled i.e. it does not remove the ability to update subresources
for existing API resources completely. If a subresource of an existing API resource needs
to be updated, this can be done via `curl`.

###### What happens if we reenable the feature if it was previously rolled back?

The `--subresource` flag can be used and subresources can be updated via kubectl again.

###### Are there any tests for feature enablement/disablement?

No, because it cannot be disabled or enabled in a single release.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout fail? Can it impact already running workloads?

The feature is encapsulated entirely within the kubectl binary, so rollout is
an atomic client binary update. Subresources can always be updated via `curl`,
so there are no version dependencies.

For kube-apiserver changes see [Version Skew Strategy](#version-skew-strategy).

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This feature is completely within the client. The upgrades and rollback of cluster will not be affected by this change.
The update and downgrade of the kubectl version will only limit the availability of the `--subresource` flag and will not
change any API behavior.

For kube-apiserver changes see [Version Skew Strategy](#version-skew-strategy).

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Cluster administrator can verify [audit entries](https://kubernetes.io/docs/tasks/debug/debug-cluster/audit/)
looking for `kubectl` invocations targeting `scale`, `status` and `resize` subresources.

###### How can someone using this feature know that it is working for their instance?

N/A

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

Since this functionality doesn't heavily modify kube-apiserver I'd expected
the SLO defined [here](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/slos.md)
to apply.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

- API call type (e.g. PATCH pods):
  GET, PATCH, PUT `/<subresource>`

- Estimated throughput:
  Negligible, because it's human initiated. At maximum, each command would involve
  two calls: 1 read and 1 mutate.

- Originating component(s):
  kubectl

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

`kubectl` is not resilient to API server unavailability.

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

2021-03-01: Initial [POC PR] created
2021-04-06: KEP proposed
2021-04-07: [Demo] in SIG CLI meeting
2022-05-25: PR for alpha implementation merged
2023-01-12: KEP graduated to Beta
2023-03-15: e2e test added for KEP as part of beta graduation
2025-01-23: KEP graduated to Stable

[POC PR]: https://github.com/kubernetes/kubernetes/pull/99556
[Demo]: https://youtu.be/zUa7dudYCQM?t=299

## Alternatives

Alternatives would be to use `curl` commands directly to update subresources.
