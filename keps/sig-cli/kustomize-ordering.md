---
title: Kustomize Resource Ordering
authors:
 - "@pwittrock"
owning-sig: sig-cli
participating-sigs:
 - sig-apps
 - sig-api-machinery
reviewers:
 - "@apelisse"
 - "@anguslees"
approvers:
 - "@monopole"
editors:
 - "@pwittrock"
creation-date: 2019-03-1
last-updated: 2019-03-1
status: implementable
see-also:
replaces:
superseded-by:
 - n/a
---

# Kustomize Resource Ordering

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

Kustomize orders Resource creation by sorting the Resources it emits based off their type.  While
this works well for most types (e.g. create Namespaces before other things), it doesn't work
in all cases and users will need the ability to break glass.

See [kubernetes-sigs/kustomize#836] for an example.

## Motivation

Users may need direct control of the Resource create / update / delete ordering in cases were
sorting by Resource Type is insufficient.

### Goals

- Provide the ability for users to break glass and override Kustomize's sort ordering.

### Non-Goals

## Proposal

Add a new field `resourceOrdering` that is a list of `key, value, type` tuples to define the ordering.

Example:

```
resourceOrdering:
- name: first
  - type: label
    key: some-label-name-1
    value: some-label-value-1
  - type: annotation
    key: some-annotation-name-a
    value: some-label-value-a
- name: second
  - type: annotation
    key: some-annotation-name-b
    value: some-annotation-value-b
- name: third
  type: label
  key: some-label-name-2
  value: some-label-value-2
```

The explicit user defined ordering using labels and annotations would take precedence over the
 ordering based off types.  Types would be used as a secondary sorting function.

- Resources labeled with `some-label-name-1=some-label-value-1` *and* annotated
  with `some-annotation-name-a=some-label-value-a` are first
  - These Resources are sorted by type
- Resources annotated with `some-annotation-name-a=some-annotation-value-b` are second
  - These Resources are sorted by type
- Resources labeled with `some-label-name-2=some-label-value-2` are third
  - These Resources are sorted by type
- Resources not matching any label or annotation are last
  - These Resources are sorted by type

Resources matching multiple orderings (e.g. have multiple matching labels and annotations) appear
in the position matching the earliest label / annotation.

### Risks and Mitigations

Risk: Users build complex orderings that are hard to reason about.
Mitigation: Good documentation and recommendations about how to keep things simple.

## Graduation Criteria

NA


## Docs

Update Kubectl Book and Kustomize Documentation

## Test plan

Unit Tests for:

- [ ] Resources with labels and annotations follow the ordering override semantics
- [ ] Resources matching multiple orderings land in the right spot
- [ ] Resources are sorted by type as a secondary factor
- [ ] Having multiple labels / annotations with the same key and different values works correctly
- [ ] Having multiple labels / annotations with different keys and the same values works correctly
- [ ] Mixing labels and annotations works
- [ ] Unrecognized type values throw and error
- [ ] Resources that don't appear in the ordering overrides appear last and are sorted by type

### Version Skew Tests

## Implementation History

## Alternatives
