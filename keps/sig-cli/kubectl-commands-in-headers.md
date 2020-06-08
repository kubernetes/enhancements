---
title: Kubectl Commands In Headers
authors:
  - "@pwittrock"
owning-sig: sig-cli
participating-sigs:
  - sig-api-machinery
reviewers:
  - "@deads2k"
  - "@kow3ns"
  - "@lavalamp"
approvers:
  - "@seans3"
  - "@soltysh"
editor: TBD
creation-date: 2019-02-22
last-updated: 2019-02-22
status: implementable
see-also:
replaces:
superseded-by:
---

# kubectl-commands-in-headers


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
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
  - [#859](https://github.com/kubernetes/enhancements/issues/859)
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
  - Standard Unit and Integration testing should be sufficient
- [x] Graduation criteria is in place
  - This is not a user facing API change
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

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

NA

### Version Skew Strategy

NA

## Implementation History


