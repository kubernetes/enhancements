---
title: Kustomize File Processing Integration
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
editor: "@pwittrock"
creation-date: 2019-01-17
last-updated: 2019-03-18
status: implemented
see-also:
  - "kustomize-subcommand-integration.md"
replaces:
superseded-by:
  - n/a
---

# Kustomize File Processing Integration

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Specifics](#specifics)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
  - [Evaluate and decide](#evaluate-and-decide)
  - [Implement](#implement)
- [Docs](#docs)
- [Test plan](#test-plan)
  - [Version Skew Tests](#version-skew-tests)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

This is a follow up to [KEP Kustomize Subcommand Integration](kustomize-subcommand-integration.md)

[Kustomize](https://github.com/kubernetes-sigs/kustomize) was introduced as 
subcommand of kubectl to allow users to build their kustomizations directly.
However users need to pipe the kustomize output to other commands in order
to use the kustomizations.

This KEP proposes integrating the kustomization libraries into the cli-runtime
file processing libraries.  Doing so will provide a cleaner, simpler UX
and provide a path for addressing issues around error handling and messaging.

## Motivation

- It is capable of removing friction that requires deeper integration - such as producing errors referencing line
  numbers of the original files (rather than the output files) and exiting with the right error code if kustomization
  fails.
- It works for tools that wrap and exec kubectl or vendor kubectl without additional steps
- It is more consistent with the UX workflow of other commands and flags (other commands don't require pipes)
- It has a cleaner and simpler UX than requiring a pipe - fewer characters to type
- It is clear which commands it that support it - apply, get, delete, etc.
- It can be more clear in the documentation when running `--help` (e.g. the -k flag is shown)

### Goals

- Provide a clean and integrated user experience when working with files from kubectl.
- Provide consistent UX across kubectl commands for working with kustomized applications.

### Non-Goals

## Proposal

Integrate kustomize directly into libraries that enable file processing for cli-runtime (e.g. resource builder).
Kubectl commands taking the common flags (`-f`, `--filename`, `-R`, `--recursive`) will support `kustomization.yaml`
files through a new flag `-k` or `--kustomize`.

Cli-runtime will add the flags `-k, --kustomize`, which will be registered along side the other file processing
flags.  If the `-k` flag is provided to a command, the experience will be similar to if the user had piped
kustomize to stdin - e.g. `kubectl kustomize <value> | kubectl <command> -f -`.  It will differ in that it provides
improved error handling and messaging.

Example: `kubectl apply -k <dir-containing-kustomization>`

Tools outside kubectl that use the cli-runtime to register file processing flags and build resources will get the
`-k` by default, but can opt-out if  they do not want the functionality.

### Specifics

- The `-f` and `-k` flags will initially be mutually exclusive
- The `-k` flag can be specified at most once
- The `-k` flag can only point to a *directory* or url containing a file named `kustomization.yaml` file
  (same as `kustomize`)

### Risks and Mitigations

Low:

When run against a `kustomization.yaml` with multiple bases, kubectl may perform multiple requests as part of the
preprocessing.  Since `-k` is a separate flag from `-f`, it is transparent to a user whether they are running
against a kustomization file or a directory of Resource Config.

## Graduation Criteria

Note: The flag itself does not have an alpha, beta, ga version.  Graduation is taken to mean - proposed iterative
improvements to the functionality.

Graduation criteria for the `-k, --kustomize` flag

### Evaluate and decide

- Determine if flag usage should be less restrictive:
  - Enable specifying multiple times?
  - Specifying the kustomization file itself?
  - Specifying it along with `-f` (separately)?
- If / when available, gather usage metrics of the `-k` flag in kubectl commands to evaluate adoption
- Gather feedback on overall flag experience from users (issues, slack, outreach, etc)
- Should we add in-kubectl documentation for kustomization format? - e.g. `kubectl kustomize --help` would 
  give information about the kustomization.yaml format

### Implement

- Figure out better error messaging w.r.t. errors from the apiserver on output files vs input files
- Feedback from evaluation

## Docs

- update all kubectl documentation that recommends piping `kustomize | kubectl` to use `-k`
- update kubectl docs on k8s.io that create configmaps and secrets from Resource Config to also show` kustomization.yaml`
- update kubectl docs on k8s.io that use `-n` to set namespace for apply to also show `kustomization.yaml`
- update imperative kubectl docs on k8s.io that set namespaces, labels, annotations to also show the declarative
  approach using kustomize
  
- Update cobra (e.g. `--help`) examples for apply, delete, get, etc to include the `-k` flag.
- Update cobra docs for `-n` flag with apply to suggest using a declarative kustomization.yaml instead
- Update cobra examples for imperative set, create commands that can be generated to call out the declarative
  approaches.

## Test plan

The following should be tests written for kubectl.

- unit test to validate that the `-k` flag correctly invokes the kustomization library
  - should invoke the library
- unit test to validate that the `--kustomize` flag works the same as the `-k` flag
  - should invoke the library
- test to validate that the `-k` flag results in the kustomized resource config being provided to the commands
  - should provide the expanded files
- test to validate the `-k` flag there are resource files in the same directory
  - should only pick up the kustomization, not other files
- test to validate what happens if the `-k` flag is provided multiple times
  - should fail
- test to validate the `-k` flag if no kustomization file is present, but there are resource files
  - should fail
- test to validate the `-k` flag points to a kustomization.yaml
  - should fail - directories only
- test to validate the `-k` flag if `-f` or `--filename` is also provided
  - should fail
- test to validate the `-k` flag if it points directory with a file containing a kustomization resource
  (group version kind), but not named `kustomization.yaml`
  - should fail - kustomization.yaml only
- test to validate that the `-k` flag can be opt-out in the cli-runtime.
  - flag should not be registered if opt-out
- TODO: add more tests here

### Version Skew Tests

NA.  This flag is a client only feature.

## Implementation History

## Alternatives
