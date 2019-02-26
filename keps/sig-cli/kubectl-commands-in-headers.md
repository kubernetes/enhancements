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
- whether or not specific flags had specific values - e.g. `-f -` for stdin
  
### Example

```sh
$ kubectl apply -f - -o yaml
```

```
Kubectl-Command: apply
Kubectl-Flags: -f
Kubectl-Input: stdin
Kubectl-Output: yaml
```


```sh
$ kubectl apply -f ./local/file -o=custom-columns=NAME:.metadata.name
```

```
Kubectl-Command: apply
Kubectl-Flags: -f;-o
Kubectl-Input: local-files
Kubectl-Output: custom-columns
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


