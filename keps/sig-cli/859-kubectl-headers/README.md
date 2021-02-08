# kubectl commands in headers


## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [Anti-Goals](#anti-goals)
- [Proposal](#proposal)
  - [X-Kubectl-Command Header](#x-kubectl-command-header)
  - [X-Kubectl-Flags Header](#x-kubectl-flags-header)
    - [Enumerated Flag Values](#enumerated-flag-values)
  - [X-Kubectl-Session Header](#x-kubectl-session-header)
  - [X-Kubectl-Deprecated Header](#x-kubectl-deprecated-header)
  - [X-Kubectl-Build Header](#x-kubectl-build-header)
  - [Example](#example)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Requests sent to the apiserver from kubectl already include a User Agent header with
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

### X-Kubectl-Command Header

The `X-Kubectl-Command` Header contains the kubectl sub command.

It contains the path to the subcommand (e.g. `create secret tls`) to disambiguate sub commands
that might have the same name and different paths.

Examples:

- `X-Kubectl-Command: apply`
- `X-Kubectl-Command: create secret tls` 
- `X-Kubectl-Command: delete`
- `X-Kubectl-Command: get`

### X-Kubectl-Flags Header

The `X-Kubectl-Flags` Header contains a list of the kubectl flags that were provided to the sub
command.  It does *not* contain the raw flag values, but may contain enumerations for
whitelisted flag values.  (e.g. for `-f` it may contain whether a local file, stdin, or remote file
was provided).  It does not normalize into short or long form.  If a flag is
provided multiple times it will appear multiple times in the list.  Flags are sorted
alpha-numerically and separated by a ',' to simplify human readability.

Examples:

- `X-Kubectl-Flags: --filename=local,--recursive,--context`
- `X-Kubectl-Flags: -f=local,-f=local,-f=remote,-R` 
- `X-Kubectl-Flags: -f=stdin` 
- `X-Kubectl-Flags: --dry-run,-o=custom-columns`

#### Enumerated Flag Values

- `-f`, `--filename`: `local`, `remote`, `stdin`
- `-o`, `--output`: `json`,`yaml`,`wide`,`name`,`custom-columns`,`custom-columns-file`,`go-template`,`go-template-file`,`jsonpath`,`jsonpath-file`
- `--type` (for patch subcommand): `json`, `merge`, `strategic`

### X-Kubectl-Session Header

The `X-Kubectl-Session` Header contains a Session ID that can be used to identify that multiple
requests which were made from the same kubectl command invocation.  The Session Header is generated
once and used for all requests for each kubectl process.

- `X-Kubectl-Session: 67b540bf-d219-4868-abd8-b08c77fefeca`

### X-Kubectl-Deprecated Header

The `X-Kubectl-Deprecated` Header is set to inform cluster admins that the command being run
has been deprecated.  This may be used by organizations to determine if they are likely
to be impacted by command deprecation and removal before they upgrade.

The `X-Kubectl-Deprecated` Header is set if the command that was run is marked as deprecated.

- The Header may have a value of `true` if the command has been deprecated, but has no removal date.
- The Header may have a value of a specific Kubernetes release.  If it does, this is the release
  that the command will be removed in.

- `X-Kubectl-Deprecated: true`
- `X-Kubectl-Deprecated: v1.16`


### X-Kubectl-Build Header

The `X-Kubectl-Build` Header may be set by building with a specific `-ldflags` value.  By default the Header
is unset, but may be set if kubectl is built from source, forked, or vendored into another command.
Organizations that distribute one or more versions of kubectl which they maintain internally may
set a flag at build time and this header will be populated with the value.

- `X-Kubectl-Build: some-value`

### Example

```sh
$ kubectl apply -f - -o yaml
```

```
X-Kubectl-Command: apply
X-Kubectl-Flags: -f=stdin,-o=yaml
X-Kubectl-Session: 67b540bf-d219-4868-abd8-b08c77fefeca
```


```sh
$ kubectl apply -f ./local/file -o=custom-columns=NAME:.metadata.name
```

```
X-Kubectl-Command: apply
X-Kubectl-Flags: -f=local,-o=custom-columns
X-Kubectl-Session: 0087f200-3079-458e-ae9a-b35305fb7432
```

```sh
kubectl patch pod valid-pod --type='json' -p='[{"op": "replace", "path": "/spec/containers/0/image", "value":"new
image"}]'
```

```
X-Kubectl-Command: patch
X-Kubectl-Flags: --type=json,-p
X-Kubectl-Session: 0087f200-3079-458e-ae9a-b35305fb7432
```


```sh
kubectl run nginx --image nginx
```

```
X-Kubectl-Command: run
X-Kubectl-Flags: --image
X-Kubectl-Session: 0087f200-3079-458e-ae9a-b35305fb7432
X-Kubectl-Deprecated: true
```

### Risks and Mitigations

Unintentionally including sensitive information in the request headers - such as local directory paths
or cluster names.  This won't be a problem as the command arguments and flag values are never directly
included.

## Design Details

### Test Plan

- Verify the Command Header is sent for commands and has the correct value
- Verify the Flags Header is sent for flags and has the correct value
- Verify the Session Header is sent for the Session and has a legitimate value
- Verify the Deprecation Header is sent for the deprecated commands and has the correct value
- Verify the Build Header is sent when the binary is built with the correct ldflags value
  specified and has the correct value

### Graduation Criteria

- Determine if additional information would be valuable to operators of clusters.
- Consider building and publishing tools for cluster operators to run which make use of the data
  - Look for deprecated command invocations
  - Build graphs of usage
  - Identify most used commands

### Upgrade / Downgrade Strategy

Not applicable. There are no cluster components affected by this feature.


### Version Skew Strategy

Not applicable. There is nothing required of the API Server, so there
can be no version skew.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**

  This feature is enabled with the KUBECTL_COMMANDS_HEADER environment
  variable set on the client command line. Since it only affects the client
  kubectl, it does not affect any cluster components.

  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: KUBECTL_COMMANDS_HEADERS
    - Components depending on the feature gate: kubectl
  - [X] Other
    - Describe the mechanism: setting an client-side environment variable for kubectl.
    - Will enabling / disabling the feature require downtime of the control
      plane? No
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
      No.

* **Does enabling the feature change any default behavior?**

  No. This feature is not user facing, so it does not change behavior.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**

  Yes. This feature can be disabled by simply removing an environment variable
  on the client command line.

* **What happens if we reenable the feature if it was previously rolled back?**

  Re-enabling this feature is simply accomplished by setting the feature
  environment variable on the client command line. There is no state, and there
  is no consequence for re-enabling the feature.

* **Are there any tests for feature enablement/disablement?**

  There will be unit tests and integration tests which test this
  feature enablement and disablement by setting the environment
  variable.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**

  A danger in the rollout of this feature is adding too much data (too many
  headers) to each of the API Server calls. We intend to mitigate this risk
  by defining a MAX_HEADERS concept to ensure the headers to not grow above a
  certain size. MAX_HEADERS will provide a mechanism to ensure the headers data
  does not grow without bound. As far as cluster workloads, this feature only
  affects kubectl; not any cluster components. So it would not be possible to
  impact running workloads.

* **What specific metrics should inform a rollback?**

  We will measure the **headers-added-round-trip-time** and compare it to the round-trip
  time for the same API Server call without headers. This ratio will give us the
  performance penalty for adding these headers. If this performance penalty exceedes
  a specific threshold, users can opt-out by removing the client-side command line
  feature environment variable to disable the headers.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

  Enabling and disabling the client-side command line environment variable will
  be tested.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**

  No

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

  Operators will see this feature in use when the **X-Kubectl-Command** header
  arrives with REST calls to the API Server.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**

  Since the cardinality is low, we will initially support the kubectl subcommand
  header (X-Kubectl-Command) in this feature. This simple metric will provide valuable
  insight into kubectl usage, and allow users to see if the feature is being used.

  - [X] Metrics
    - Metric name: X-Kubectl-Command header
    - [Optional] Aggregation method:
    - Components exposing the metric: kubectl
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

  TBD

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**

  We believe all the other specified **X-Header** will be useful, but we are starting
  out simply in alpha to prove the value and feasibility of the feature. We will add
  others as we approach beta.

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
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

### Scalability

* **Will enabling / using this feature result in any new API calls?**

  No

* **Will enabling / using this feature result in introducing new API types?**

  No

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

  No

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**

  No

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**

  Possibly. This feature increases the size of the REST call from kubectl
  to the API Server by adding more headers to the calls. We will monitor
  this request size increase to ensure there is no deleterious effect.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**

  Possibly. This feature increases the size of the REST call from kubectl
  to the API Server by adding more headers to the calls. We will monitor
  this request size increase to ensure there is no deleterious effect.

## Implementation History


