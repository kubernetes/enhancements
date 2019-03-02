---
title: Kustomize PodSpec Patch
authors:
 - "@pwittrock"
owning-sig: sig-cli
participating-sigs:
 - sig-apps
reviewers:
 - "@Liujingfang1"
 - "@janetkuo"
approvers:
 - "@monopole"
editors:
 - "@pwittrock"
creation-date: 2019-03-01
last-updated: 2019-03-01
status: provisional
see-also:
replaces:
superseded-by:
 - n/a
---

# Kustomize PodSpec Patch

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

A common need for users is to inject sidecar containers across many different workloads.
While overlays and patches allow injection on a per-Resource basis, they don't
provide a mechanism for injecting the same patch across multiple workloads.

## Motivation

Cross-cutting PodSpec customizations are necessary for usecases such as:

- Injecting Envoy proxies into all Workloads
- Injecting log rotation containers into all Workloads
- Setting the ServiceAccount for Workloads matching a query

### Goals

Allow users to define cross-cutting PodSpec customizations by matching the Resource annotations.


### Non-Goals

## Proposal

Add field to `kustomization.yaml`: `podSpecPatches`.  It has both a file with a `PodSpec`
that will be patched, and a `annotationSelector` defining the Resources that will be patched.

Resources matching the annotationSelector that have a PodSpec in their schema will be
patched.

```go

type Kustomization struct {
	// ...
	
	patchesPodSpec []PatchPodSpec `json:"patchesPodSpec"`
	// ...
}

// ResourceSelector selects Resources that it matches
type PatchPodSpec struct {
	// matchAnnotations is a map of {key,value} pairs.  A Resource matches if it has
	// *all* of the annotations in matchAnnotations appear in the Resource metaData.
	// Null and empty values are not allowed.
	// +optional
	MatchAnnotations map[string]string `json:"matchAnnotations,omitempty"`
	
	// file is the path to the file with the PodSpec patch
	File string `json:"file,omitempty"`
	
	// TODO: Consider adding field: MatchExpressions []AnnotationSelectorRequirement 
}
```

**Example:**

```yaml
patchesPodSpec:
- file: path/to/patch.yaml
  matchAnnotations:
    foo: bar
- file: ../patch/to/another/patch.yaml
  matchAnnotations:
    foo: bar
```

### Risks and Mitigations


## Graduation Criteria

Use customer feedback to determine if we should support:

- `LabelSelectors`
- `AnnotationSelectorRequirement` (like `LabelSelectorRequirement` but for annotations)
- Cross-Cutting patches for arbitrary fields, not just PodSpec


## Docs

Update Kustomize and Kubectl docs with this new capability.

## Test plan

Unit test matching Resources and performing customizations.

## Implementation History

## Alternatives
