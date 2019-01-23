---
kep-number: draft-20190121
title: Provide Conformance Document as part of the release docs for each Kubernetes minor release
authors:
  - "@brahmaroutu"
owning-sig: sig-testing
participating-sigs:
  - sig-release
  - sig-docs
  - wg-conformance
reviewers:
  - @spiffxp
  - @timothysc
  - @ixdy
  - @sig-docs-en-reviews
  - @zacharysarah
approvers:
  - TBD
editor: TBD
creation-date: 2019-01-21
last-updated: 2019-01-21
status: provisional
---

# Provide Conformance Document as part of the release docs for each Kubernetes minor release

## Table of Contents

- [Conformance Document as part of the release docs](#breaking-apart-the-kubernetes-test-tarball)
  - [Table of Contents](#table-of-contents)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Graduation Criteria](#graduation-criteria)
  - [References](#references)
  - [Implementation History](#implementation-history)

## Summary

The Kubernetes release artifacts must include conformance document as part of release documents.

This KEP proposes to add docs/conformance folder with a single conformance document that will 
contain list of all conformance tests that are part of this release.
file: docs/conformance/conformance.md 


## Motivation

As there is no proper place to publish conformance document per release, currently
conformance document is checked in under the repo cncf/k8s-conformance. Even though
this is a useful location but each release specific document should be part of the
kubernetes release process. 
If we can automatically build conformance document as part of each minor release it 
should be shipped along with the release tar that contains other documentation for
the release.

### Goals

* Build process needs to be updated to create a conformance binary that can generate conformance document
* Template the code to generate release information into the document header for each release (PR in progress)
* Bundle conformance document as part of the other docs

### Non-Goals

NA

## Proposal

* Update the BUILD files to create the conformance binary
* Update the release build to process and generate conformance document

### Risks and Mitigations

None

## Graduation Criteria

Checks to see if docs/conformance/conformance.md exists as part of the release tar(kubernetes.tar.gz).

## References
https://github.com/kubernetes/kubernetes/pull/72168 (Allow version field to be changed)

## Implementation History
Current version of Conformance Document exists under 
