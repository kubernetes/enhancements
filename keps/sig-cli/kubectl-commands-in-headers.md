---
title: Kubectl Commands In Headers
authors:
  - "@pwittrock"
owning-sig: sig-cli
participating-sigs:
  - sig-api-machinery
reviewers:
  - "@lavalamp"
  - "@kow3ns"
approvers:
  - "@soltysh"
  - "@seans3"
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

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
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

NA

## Proposal

Include in requests made from kubectl to the apiserver:

- the kubectl subcommand
- which flags were specified (but not the values)
- enum values for stdin and stdout
- a generated session id

### Kubectl-Command Header

The `Kubectl-Command` Header contains the kubectl sub command.  It contains
the path to the subcommand (e.g. `kubectl apply`) to disambiguate sub commands
that might have the same name and different paths.

Examples:

- `Kubectl-Command: kubectl apply`
- `Kubectl-Command: kubectl create secret tls` 
- `Kubectl-Command: kubectl delete`
- `Kubectl-Command: kubectl get`

### Kubectl-Flags Header

The `Kubectl-Flags` Header contains a list of the kubectl flags that were provided to the sub
command.  It does *not* contain the flag values, only the names of the flags.  It
does not normalize into short or long form.  If a flag is provided multiple times
it will appear multiple times in the list.  Flags are sorted alpha-numerically and
separated by a ','.

Examples:

- `Kubectl-Flags: --filename,--recursive`
- `Kubectl-Flags: -f,-f,-f,-R` 
- `Kubectl-Flags: --dry-run,-o`

### Kubectl-Input Header

The `Kubectl-Input` Header contains the types of input used.  Because the flag values are not
provided, this can be used to determine if the command input was from STDIN, Local Files or
Remote Files.  Format is a list of the types of input provided.

Examples:

- `Kubectl-Input: stdin`
- `Kubectl-Input: file` 
- `Kubectl-Input: http`
- `Kubectl-Input: file,stdin`
- `Kubectl-Input: file,file,http`

### Kubectl-Output Header

The `Kubectl-Output` Header contains the type of output used.  Because the flag values are
not provided, this can be used to determine if the output is yaml,json,go-template,wide.
Note: it does *not* contain non-enumeration values, such as the custom-columns values for
for custom-columns output.

Examples:

- `Kubectl-Output: yaml`
- `Kubectl-Output: json`
- `Kubectl-Output: wide`
- `Kubectl-Output: default`
- `Kubectl-Output: jsonpath`
- `Kubectl-Output: custom-columns`
- `Kubectl-Output: custom-columns-file`

### Kubectl-Session Header

The `Kubectl-Session` Header contains a Session ID that can be used to identify that multiple
requests were made from the same kubectl command invocation.  The Session Header is generated
once for each kubectl process.

- `Kubectl-Session: 67b540bf-d219-4868-abd8-b08c77fefeca`

### Example

```sh
$ kubectl apply -f - -o yaml
```

```
Kubectl-Command: kubectl apply
Kubectl-Flags: -f
Kubectl-Input: stdin
Kubectl-Output: yaml
Kubectl-Session: 67b540bf-d219-4868-abd8-b08c77fefeca
```


```sh
$ kubectl apply -f ./local/file -o=custom-columns=NAME:.metadata.name
```

```
Kubectl-Command: kubectl apply
Kubectl-Flags: -f;-o
Kubectl-Input: file
Kubectl-Output: custom-columns
Kubectl-Session: 0087f200-3079-458e-ae9a-b35305fb7432
```

### Risks and Mitigations

Unintentionally including sensitive information in the request headers - such as local directory paths
or cluster names.

Mitigations: only include the following non-sensative information:

- The sub command itself
- Which flags were specified, but not their values.  e.g. `-f`, but not the argument
- What type of input was specified (e.g. stdin, local files, remote files)
- What type of output was specified (e.g. yaml, json, wide, default, name, etc)

## Design Details

### Test Plan

- Verify the Header is sent for commands and has the right value
- Verify the Header is sent for flags and has the right value

### Graduation Criteria

NA

### Upgrade / Downgrade Strategy

NA

### Version Skew Strategy

NA

## Implementation History


