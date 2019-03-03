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

- Provide the ability for users to override the order that Kustomize emits Resources
  - Used by `kubectl apply` or order Resource operations
  - Used to override creation order for meta-Resources such as Namespaces

### Non-Goals

- Ensure certain Resources are *Settled* or *Ready* before other Resources
  - Ensure dependencies between Workloads

## Proposal

Provide a simple mechanism allowing users to override the order that
Resource operations are Applied.  Add a new field `sortOrder` that is
a list of `ResourceSelector`s which match Resources based
off their annotations.

```go

type Kustomization struct {
	// ...
	
	// sortOrder defines a precedence for ordering Resources.  Resources
	// will be sorted in the order the match a ResourecSelector before
	// they are emitted.
	SortOrder []ResourceSelector `json:"sortOrder"`

	// ...
}

// ResourceSelector selects Resources that it matches
type ResourceSelector struct {
	// matchAnnotations is a map of {key,value} pairs.  A Resource matches if it has
	// *all* of the annotations in matchAnnotations appear in the Resource metaData.
	// Null and empty values are not allowed.
	// +optional
	MatchAnnotations map[string]string `json:"matchAnnotations,omitempty"`
	
	// TODO: Consider adding field: MatchExpressions []AnnotationSelectorRequirement 
}

```
Example:

```
sortOrder:
- matchAnnotations:
    some-annotation-name-1: some-annotation-value-1
    some-annotation-name-a: some-label-value-a
- matchAnnotations:
    some-annotation-name-1: some-annotation-value-1
- matchAnnotations:
    some-annotation-name-2: some-annotation-value-2
```

The explicit user defined ordering using annotations will take precedence over the type based
orderings.  Types would be used as a fallback sorting function amongst Resource with equal
precedence in the explicit ordering.

- Resources annotated with `some-annotation-name-1: some-annotation-value-1` *and* annotated
  with `some-annotation-name-a: some-annotation-value-a` are first
  - These Resources are sorted by type
- Resources annotated with `some-annotation-name-1=some-annotation-value-1`
  (without `some-annotation-name-a: some-annotation-value-a`) appear second
  - These Resources are sorted by type
- Resources annotated with `some-annotation-name-2=some-annotation-value-2` appear third
  - These Resources are sorted by type
- Resources not matching any annotation are last
  - These Resources are sorted by type

Resources matching multiple orderings (e.g. have multiple matching annotations) appear
in the position matching the earliest label / annotation.

**Note:** Throw an error if there is a selector that selects a superset of another selector 
and it appears first.  e.g. this should throw an error because the first selector will match
everything that the second selector does, and it will have no effect.

```
sortOrder:
- matchAnnotations:
    some-annotation-name-1: some-annotation-value-1
- matchAnnotations:
    some-annotation-name-1: some-annotation-value-1
    some-annotation-name-a: some-label-value-a
```

### Risks and Mitigations

Risk: Users build complex orderings that are hard to reason about.
Mitigation: Good documentation and recommendations about how to keep things simple.

## Graduation Criteria

Use customer feedback to determine if we should support:

- `LabelSelectors`
- `AnnotationSelectorRequirement` (like `LabelSelectorRequirement` but for annotations)
- Explicitly order "default" (e.g. Resources that don't match any of the selectors) instead of
  them being last.


## Docs

Update Kubectl Book and Kustomize Documentation

## Test plan

Unit Tests for:

- [ ] Resources with annotations follow the ordering override semantics
- [ ] Resources matching multiple orderings land in the right spot
- [ ] Throw an error if a superset selector appears after its subset (e.g. a more restrict selector appears later)
  - This will have no effect, as everything will have already been matched
- [ ] Resources are sorted by type as a secondary factor
- [ ] Having multiple annotations with the same key and different values works correctly
- [ ] Having multiple annotations with different keys and the same values works correctly
- [ ] Empty and Null `MatchAnnotations` throw an error
- [ ] Resources that don't appear in the ordering overrides appear last and are sorted by type

### Version Skew Tests

## Implementation History

## Alternatives
