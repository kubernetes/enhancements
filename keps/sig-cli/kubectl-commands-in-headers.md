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

A table of contents is helpful for quickly jumping to sections of a KEP and for highlighting any additional information provided beyond the standard KEP template.
[Tools for generating][] a table of contents from markdown are available.

- [Table of Contents](#table-of-contents)
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
- [Goals](#goals)
- [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
- [Test Plan](#test-plan)
- [Implementation History](#implementation-history)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

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

Requests sent to the apiserver from kubectl include http request headers with context about the kubectl command that
created the request.  This information could be used by cluster administrators for debugging or
to gather telemetry about how users are interacting with the cluster.


## Motivation

Kubectl generates requests sent to the apiserver for commands such as `apply`, `delete`, `edit`, `run`, however
the context of the command for the requests is lost and unavailable to cluster administrators.  Context would be
useful to cluster admins both for debugging the cause of requests as well as providing telemetry about how users
are interacting with the cluster.

### Goals

- Allow cluster administrators to identify how requests in the logs were generated from
  kubectl commands.

### Non-Goals

### Anti-Goals

*We explicitly don't want the following*

- Make decisions of any sort in the apiserver based on these headers.
  - This information is intended to be used by humans-only, specifically **for debugging and telemetry**.

## Proposal

Include in requests made from kubectl to the apiserver:

- the kubectl subcommand
- which flags were specified as well as whitelisted enum values for flags (never arbitrary values)
- a generated session id
- never include the flag values directly, only use a predefined enumeration
- never include arguments to the commands

### Kubectl-Command Header

The `Kubectl-Command` Header contains the kubectl sub command.  It contains
the path to the subcommand (e.g. `create secret tls`) to disambiguate sub commands
that might have the same name and different paths.

Examples:

- `Kubectl-Command: apply`
- `Kubectl-Command: create secret tls` 
- `Kubectl-Command: delete`
- `Kubectl-Command: get`

### Kubectl-Flags Header

The `Kubectl-Flags` Header contains a list of the kubectl flags that were provided to the sub
command.  It does *not* contain the raw flag values, but may contain enumerations for
whitelisted flag values.  (e.g. for `-f` it may contain whether a local file, stdin, or remote file
was provided).  It does not normalize into short or long form.  If a flag is
provided multiple times it will appear multiple times in the list.  Flags are sorted
alpha-numerically and separated by a ','.

Examples:

- `Kubectl-Flags: --filename=local,--recursive,--context`
- `Kubectl-Flags: -f=local,-f=local,-f=remote,-R` 
- `Kubectl-Flags: -f=stdin` 
- `Kubectl-Flags: --dry-run,-o=custom-columns`

#### Enumerated Flag Values

- `-f`, `--filename`: `local`, `remote`, `stdin`
- `-o`, `--output`: `json`,`yaml`,`wide`,`name`,`custom-columns`,`custom-columns-file`,`go-template`,`go-template-file`,`jsonpath`,`jsonpath-file`
- `--type` (for patch subcommand): `json`, `merge`, `strategic`

### Kubectl-Session Header

The `Kubectl-Session` Header contains a Session ID that can be used to identify that multiple
requests which were made from the same kubectl command invocation.  The Session Header is generated
once for each kubectl process.

- `Kubectl-Session: 67b540bf-d219-4868-abd8-b08c77fefeca`

### Example

```sh
$ kubectl apply -f - -o yaml
```

```
Kubectl-Command: apply
Kubectl-Flags: -f=stdin,-o=yaml
Kubectl-Session: 67b540bf-d219-4868-abd8-b08c77fefeca
```


```sh
$ kubectl apply -f ./local/file -o=custom-columns=NAME:.metadata.name
```

```
Kubectl-Command: apply
Kubectl-Flags: -f=local,-o=custom-columns
Kubectl-Session: 0087f200-3079-458e-ae9a-b35305fb7432
```

```sh
kubectl patch pod valid-pod --type='json' -p='[{"op": "replace", "path": "/spec/containers/0/image", "value":"new
image"}]'
```

```
Kubectl-Command: patch
Kubectl-Flags: --type=json,-p
Kubectl-Session: 0087f200-3079-458e-ae9a-b35305fb7432
```

### Risks and Mitigations

Unintentionally including sensitive information in the request headers - such as local directory paths
or cluster names.  This won't be a problem as the command arguments and flag values are never directly
included.

## Design Details

### Test Plan

- Verify the Header is sent for commands and has the right value
- Verify the Header is sent for flags and has the right value
- Verify the Header is sent for the Session and has a legitimate value

### Graduation Criteria

NA

### Upgrade / Downgrade Strategy

NA

### Version Skew Strategy

NA

## Implementation History


