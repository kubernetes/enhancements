# KEP-859: kubectl commands in headers

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [Anti-Goals](#anti-goals)
- [Proposal](#proposal)
  - [Kubectl-Command Header](#kubectl-command-header)
  - [Kubectl-Session Header](#kubectl-session-header)
  - [Example](#example)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
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
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Requests sent to the kube-apiserver from kubectl already include a User-Agent header with
information about the kubectl build.  This KEP proposes sending http request headers
with additional context about the kubectl command that created the request.
This information may be used by operators of clusters for debugging or
to gather telemetry about how users are interacting with the cluster.

## Motivation

Kubectl generates requests sent to the apiserver for commands such as `apply`, `delete`, `edit`, `run`, however
the context of the command for the requests is lost and unavailable to cluster administrators.  Context would be
useful to cluster admins both for debugging the cause of requests as well as providing telemetry about how users
are interacting with the cluster, which could be used for various purposes.

### Goals

- Allow cluster administrators to identify how requests in the logs were generated from
  kubectl commands.

Possible applications of this information may include but are not limited to:

- Organizations could learn how users are interacting will their clusters to inform what internal
  tools they build and invest in or what gaps they may need to fill.
- Organizations could identify if users are running deprecated commands that will be removed
  when the version of kubectl is upgraded.  They could do this before upgrading kubectl.
  - SIG-CLI could build tools that cluster admins run and perform this analysis
    to them to help with understanding whether they will be impacted by command deprecation
- Organizations could identify if users are running kubectl commands that are inconsistent with
  the organization's internal best practices and recommendations.
- Organizations could voluntarily choose to bring back high-level learnings to SIG-CLI regarding
  which and how commands are used.  This could be used by the SIG to inform where to invest resources
  and whether to deprecate functionality that has proven costly to maintain.
- Cluster admins debugging odd behavior caused by users running kubectl may more easily root cause issues
  (e.g. knowing what commands were being run could make identifying miss-behaving scripts easier)
- Organizations could build dashboards visualizing which kubectl commands where being run
  against clusters and when.  This could be used to identify broader usage patterns within the
  organization.


### Non-Goals

*The following are not goals of this KEP, but could be considered in the future.*

- Supply Headers for requests made by kubectl plugins.  Enforcing this would not be trivial.
- Send Headers to the apiserver for kubectl command invocations that don't make requests -
  e.g. `--dry-run`

### Anti-Goals

*The following should be actively discouraged.*

- Make decisions of any sort in the apiserver based on these headers.
  - This information is intended to be used by humans for the purposes of developing a better understanding
    of kubectl usage with their clusters, such as **for debugging and telemetry**.

## Proposal

Include in http requests made from kubectl to the apiserver:

- the kubectl subcommand
- which flags were specified as well as whitelisted enum values for flags (never arbitrary values)
- a generated session id
- never include the flag values directly, only use a predefined enumeration
- never include arguments to the commands, only the sub commands themselves
- if the command is deprecated, add a header including when which release it will be removed in (if known)
- allow users and organizations that compile their own kubectl binaries to define a build metadata header

### Kubectl-Command Header

The `Kubectl-Command` Header contains the kubectl sub command.

It contains the path to the subcommand (e.g. `create secret tls`) to disambiguate sub commands
that might have the same name and different paths.

Examples:

- `Kubectl-Command: kubectl apply`
- `Kubectl-Command: kubectl create secret tls`
- `Kubectl-Command: kubectl delete`
- `Kubectl-Command: kubectl get`

### Kubectl-Session Header

The `Kubectl-Session` Header contains a Session ID that can be used to identify that multiple
requests which were made from the same kubectl command invocation.  The Session Header is generated
once and used for all requests for each kubectl process.

- `Kubectl-Session: 67b540bf-d219-4868-abd8-b08c77fefeca`


### Example

```sh
$ kubectl apply -f - -o yaml
```

```
Kubectl-Command: apply
Kubectl-Session: 67b540bf-d219-4868-abd8-b08c77fefeca
```

```sh
$ kubectl apply -f ./local/file -o=custom-columns=NAME:.metadata.name
```

```
Kubectl-Command: apply
Kubectl-Session: 0087f200-3079-458e-ae9a-b35305fb7432
```

```sh
kubectl patch pod valid-pod --type='json' -p='[{"op": "replace", "path": "/spec/containers/0/image", "value":"new
image"}]'
```

```
Kubectl-Command: patch
Kubectl-Session: 0087f200-3079-458e-ae9a-b35305fb7432
```

```sh
kubectl run nginx --image nginx
```

```
Kubectl-Command: run
Kubectl-Session: 0087f200-3079-458e-ae9a-b35305fb7432
```

### Risks and Mitigations

Unintentionally including sensitive information in the request headers - such as local directory paths
or cluster names.  This won't be a problem as the command arguments and flag values are never directly
included.

## Design Details

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Unit tests

- `k8s.io/cli-runtime/pkg/genericclioptions`: `2025-10-08` - `22.9%`
- `k8s.io/kubectl/pkg/cmd/`: `2025-10-08` - `61.8%`

##### Integration tests

None are necessary, the entire functionality can be tested with unit tests.

##### e2e tests

None are necessary, the entire functionality can be tested with unit tests.

### Graduation Criteria

#### Alpha

- Initial implementation behind `KUBECTL_COMMAND_HEADERS` environment variable.

#### Beta

- Change feature gate from default **off** to default **on**
- Documentation in kubectl section of [kubernetes.io] describes the extra headers sent and how to disable these headers if necessary.
- kubectl headers remove the `X-` prefix to align with [IETF guidance](https://datatracker.ietf.org/doc/html/rfc6648). Example: `X-Kubectl-Command` header is changed to `Kubectl-Command`.
- All other issues raised during alpha use of feature are resolved.
- Completion of the test plan

#### GA

- Address feedback.

### Upgrade / Downgrade Strategy

Not applicable. There are no cluster components affected by this feature.

### Version Skew Strategy

Not applicable. There is nothing required of the API Server, so there
can be no version skew.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:

- [X] Other
  - Describe the mechanism:  The environment variable `KUBECTL_COMMAND_HEADERS=true`.
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime of the control
    plane? No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No.

###### Does enabling the feature change any default behavior?

No. This feature is not user facing, so it does not change behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. This feature can be disabled by simply setting the `KUBECTL_COMMAND_HEADERS`
environment variable to false on the command line.

###### What happens if we reenable the feature if it was previously rolled back?

Re-enabling this feature is simply accomplished by removing the feature
environment variable on the client command line. There is no state, and there
is no consequence for re-enabling the feature.

###### Are there any tests for feature enablement/disablement?

Yes, there is a [unit test](https://github.com/kubernetes/kubernetes/blob/2e2c63ef731ff1526321b4f81508734d68df2872/staging/src/k8s.io/kubectl/pkg/cmd/cmd_test.go#L364-L405),
exercising the `KUBECTL_COMMAND_HEADERS` environment variable.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A danger in the rollout of this feature is adding too much data (too many
headers) to each of the API Server calls. We intend to mitigate this risk
by defining a MAX_HEADERS concept to ensure the headers to not grow above a
certain size. MAX_HEADERS will provide a mechanism to ensure the headers data
does not grow without bound. As far as cluster workloads, this feature only
affects kubectl; not any cluster components. So it would not be possible to
impact running workloads.

###### What specific metrics should inform a rollback?

We will measure the **headers-added-round-trip-time** and compare it to the round-trip
time for the same API Server call without headers. This ratio will give us the
performance penalty for adding these headers. If this performance penalty exceeds
a specific threshold, users can opt-out by removing the client-side command line
feature environment variable to disable the headers.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This feature is completely within the client. The upgrades and rollback of cluster will not be affected by this change.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Not applicable.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [x] Other (treat as last resort)
  - Details: One can verify headers being sent to the cluster on the wire.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Not applicable.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Not applicable.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Not applicable.

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

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Negligible. This feature increases the size of the REST call from kubectl
to the API Server by adding more headers to the calls. Constraining the
size of the added headers through MAX_HEADERS will reduce the risk of
any performance degradation. We will monitor this request size increase
to ensure there is no deleterious effect.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Negligible. This feature increases the size of the REST call from kubectl
to the API Server by adding more headers to the calls. Constraining the
size of the added headers through MAX_HEADERS will reduce the risk of
any performance degradation. We will monitor this request size increase
to ensure there is no deleterious effect.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

`kubectl` is not resilient to API server unavailability.

###### What are other known failure modes?

Not applicable.

###### What steps should be taken if SLOs are not being met to determine the problem?

Not applicable.

## Implementation History

2019-02-22: Initial version of this document.
2021-02-10: [(Alpha) Kubectl command headers in requests: KEP 859](https://github.com/kubernetes/kubernetes/pull/98952).
2021-05-13: Beta promotion of this KEP.
2021-06-27: [kubectl command headers as default in beta](https://github.com/kubernetes/kubernetes/pull/103238).
2025-10-08: Stable promotion.

## Alternatives

Alternatives would be to use a wrapper around `kubectl` to inject additional headers.
