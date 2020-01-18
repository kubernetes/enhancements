---
title: Extend Kustomize Patches to Multiple Targets
authors:
 - "@Liujingfang1"
owning-sig: sig-cli
participating-sigs:
 - sig-apps
reviewers:
 - "@pwittrock"
 - "@mengqiy"
approvers:
 - "@monopole"
editor: "@Liujingfang1"
creation-date: 2019-03-14
last-updated: 2019-03-18
status: implementable
see-also:
replaces:
superseded-by:
 - n/a
---

# Extend Kustomize Patches to Multiple Targets

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Docs](#docs)
- [Test plan](#test-plan)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary
Currently, there are different types of patches supported in Kustomize:
[strategic merge patch](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/strategic-merge-patch.md) and [JSON patch](https://tools.ietf.org/html/rfc6902).

```
patchesStrategicMerge:
- service_port_8888.yaml
- deployment_increase_replicas.yaml
- deployment_increase_memory.yaml

patchesJson6902:
- target:
    version: v1
    kind: Deployment
    name: my-deployment
  path: add_init_container.yaml
- target:
    version: v1
    kind: Service
    name: my-service
  path: add_service_annotation.yaml
```

Both types need group, version, kind and name(GVKN) of a Kubernetes resource to find
the unique target to perform the patching. In strategic merge patch, GVKN is included
in the patch itself. In JSON patch, the GVKN is specified in `kustomization.yaml`.

There have been [requests](https://github.com/kubernetes-sigs/kustomize/issues/720) for patching multiple targets by one patch for different purposes: 
- override one field for all objects of one type
- add or remove common command arguments for all containers
- inject a [sidecar proxy](https://istio.io/docs/setup/kubernetes/sidecar-injection/) as in istio to all containers

## Motivation

Extend current patching mechanism of strategic merge patch and JSON patch from one target
to multiple targets.

### Goals

Allow users to patch multiple target resources by one patch in Kustomize.
The target resources can be matched by the intersection of resources selected by
- LabelSelector
- Annotations
- Group, Version, Kind and Name(Name can be regex)


### Non-Goals
- Add a different type of patches

## Proposal

Add field to `kustomization.yaml`: `patches`. It has a block specifying the targets,
a relative file path pointing to the patch file, and a type string specifying either
using strategic merge patch or JSON patch.

```go

type Kustomization struct {
	// ...

	Patches []Patch `json:"patches"`
	// ...
}

type Patch struct {
	// Path is a relative file path to the patch file.
	Path string `json:"path,omitempty"`

	// Target points to the resources that the patch is applied to
	Target PatchTarget `json:"target,omitempty"`

	// Type is one of `StrategicMergePatch` or `JsonPatch`
	Type string `json:"type,omitempty"`
}

// PatchTarget specifies a set of resources
type PatchTarget struct {
	// Group of the target
	Group string `json:"group,omitempty"`

	// Version of the target
	Version string `json:"version,omitemtpy"`

	// Kind of the target
	Kind string `json:"kind,omitempty"`

	// Name of the target
	// The name could be with wildcard to match a list of Resources
	Name string `json:"name,omitempty"`

	// MatchAnnotations is a map of key-value pairs.
	// A Resource matches it will be appied the patch
	MatchAnnotations map[string][string] `json:"matchAnnotations,omitempty"`

	// LabelSelector is a map of key-value pairs.
	// A Resource matches it will be applied the patch.
	LabelSelector map[string][string] `json:"labelSelector,omitempty"`
}
```

**Example:**

```yaml
patches:
- file: path/to/patch.yaml
  type: StrategicMergePatch
  target:
    matchAnnotations:
      foo: bar
- file: path/to/patch2.yaml
  type: JsonPatch
  target:
    labelSelector:
      app: test
- file: path/to/patch3.yaml
  type: JsonPatch
  target:
    Kind: Deployment
    Name: app1-*
    labelSelector:
      env: dev
- file: path/to/patch4.yaml
```

### Risks and Mitigations
This change is compatible with Kustomize 2.0.*,
but need to bump the minor version for feature implementation.

## Graduation Criteria
Since the proposed `patches` can cover current `patchesJson6902` and
`patchesStrategicMerge`, those two fields can be deprecated in
Kustomize 3.0.0.

## Docs

Update Kustomize and Kubectl docs with this new capability.

## Test plan

Unit test matching Resources and performing customizations.

## Implementation History
- Add `Patch` struct in `Kustomization` type.
- Update the patching transformer to recognize `Patch` and match
  multiple resources
- Add unit test and integration test

## Alternatives
