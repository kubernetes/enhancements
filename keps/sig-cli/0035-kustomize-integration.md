---
kep-number: 34
title: Integrate Kustomize into cli-runtime
authors:
 - "@pwittrock"
owning-sig: sig-cli
participating-sigs:
 - sig-cli
reviewers:
 - "@liggitt"
 - "@seans3"
 - "@soltysh"
 - "@monopole"
approvers:
 - "@liggitt"
 - "@seans3"
 - "@soltysh"
 - "@monopole"
editors:
 - "@pwittrock"
creation-date: 2019-01-17
last-updated: 2019-01-17
status: provisional
see-also:
 - "[KEP-0031](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cli/0031-kustomize.md)"
replaces:
superseded-by:
 - n/a
---

# Enable kustomize subcommand in kubectl

## Table of Contents
* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
  * [Goals](#goals)
  * [Non-Goals](#non-goals)
* [Proposal](#proposal)
  * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)
* [Alternatives](#alternatives)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Summary

This is a follow up to [KEP-0031](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cli/0031-kustomize.md)

[Kustomize](https://github.com/kubernetes-sigs/kustomize) is a subcommand of kubectl.  However it is
not tightly integrated into the file processing libraries, creating user friction around error
handling and messaging.

## Motivation

- It is capable of removing friction that requires deeper integration - such as producing errors referencing line
  numbers of the original files (rather than the output files) and providing the right error code if kustomization
  fails.
- It is more consistent with UX workflow with other commands and flags
- It has a cleaner and simpler UX than pipes
- It is clear which commands it should be used with - apply, get, delete, etc.

### Goals

- Provide a clean and integrated user experience when working with files from kubectl.
- Provide consistent UX across kubectl commands for working with kustomized applications.

### Non-Goals

## Proposal

Integrate kustomize directly into libraries that enable file processing for cli-runtime (e.g. resource builder).
Kubectl commands taking the common flags (`-f`, `--filename`, `-R`, `--recursive`) will support `kustomization.yaml`
files.

Cli-runtime will add the flags `-k, --kustomize=[]`, which will be registered along side the other file processing
flags.  If the `-k` flags are provided to a command, the experience will be similar to if the user had piped
kustomize to stdin - e.g. `kubectl kustomize <value> | kubectl <command> -f -`.  It will differ in that it provides
improved error handling and messaging.

Example: `kubectl apply -k <dir-containing-kustomization>`

Tools outside kubectl that use the cli-runtime to register file processing flags and build resources will get the
`-k` by default, but can opt-out if they do not want the functionality.

### Risks and Mitigations

Low:

When run against a kustomization.yaml with multiple bases, kubectl may perform multiple requests as part of the
preprocessing.  Since `-k` is a separate flag from `-f`, it is transparent to a user whether they are running
against a kustomization file or a directory of Resource Config.

## Graduation Criteria

NA

## Implementation History

## Alternatives
