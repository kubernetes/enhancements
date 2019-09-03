---
title: Kustomize Default Configuration Adjustments
authors:
 - "@jbrette"
owning-sig: sig-cli
participating-sigs:
 - sig-apps
 - sig-api-machinery
reviewers:
 - "@monopole"
 - "@Liujingfang1"
approvers:
 - "@Liujingfang1"
editors:
 - "@jbrette"
creation-date: 2019-09-01
last-updated: 2019-09-01
status: provisional
see-also:
replaces:
superseded-by:
 - n/a
---

# Kustomize Default Configuration Adjustments

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

Kustomize builtin transformers are provided with default configurations describing which
resource fields are subject of a particular transformation. kustomize provides the ability
to add fields to the default configuration but does not let the user adjust that configuration
in order to skip a transformation for a specific resource or resource kind.

We are aiming at addressing those issues:
- Make cluster level kind configurable [#617](https://github.com/kubernetes-sigs/kustomize/issues/617)
- Do not add namespace to cluster scoped CRDs [#552](https://github.com/kubernetes-sigs/kustomize/issues/552) 
- Skipping nameprefix for some resource kind [#519](https://github.com/kubernetes-sigs/kustomize/issues/617)
- Adding labels only to metadata.labels [#330](https://github.com/kubernetes-sigs/kustomize/issues/330)
- Skip adding labels to certain paths [#519](https://github.com/kubernetes-sigs/kustomize/issues/519)
- Unable to disable commonLabels injection using transformer config: Error: conflicting fieldspecs [#817](https://github.com/kubernetes-sigs/kustomize/issues/817)

## Motivation


### Goals

- Provide the ability for the user to skip transformation for specific group/version/kind especially usefull if the default
  configuration is using a wild card (see namespace and prefixsuffix transformer configuration).
- Provide the ability of the user to replace or remove an entry in the default configuration which is not matching his needs
  (see commonLabel transformer configuration)

  
### Non-Goals

- Provide backward compatibility with the transformer default configurations.
- Provide a way for the user to exclude a kind from a wild card based set as opposed to have to list all the possible kinds/gvks.
- Provide a way for the user to remove a configuration entry from the default configuration if it is not aligned with the needs
  of his project.
- Provide a simple method to the transformer developers to select the list of "mutable" fields based on the gvk of the resources.

## Proposal

The propsal has three parts:

1. Update the FieldSpec definition to add a "skip" field as follow (It is also useful external transformers needing wild card):

```go
type FieldSpec struct {
	gvk.Gvk            `json:",inline,omitempty" yaml:",inline,omitempty"`
	Path               string `json:"path,omitempty" yaml:"path,omitempty"`
	CreateIfNotPresent bool   `json:"create,omitempty" yaml:"create,omitempty"`
	SkipTransformation bool   `json:"skip,omitempty" yaml:"skip,omitempty"`
}
```

   
2. Wrap []FieldSpec into FieldSpecs and add high level method to ease implementation of transformers, during loop accross
   resources and fieldspec.

```go
type FieldSpecs []FieldSpec

func (s FieldSpecs) ApplicableFieldSpecs(x gvk.Gvk) FieldSpecs {
// ....
}
```

3. Wrap FieldSpec into a FieldSpecConfig, to let the user remove and replace configuration entries embedded in the default configuration
   provided with kustomize
   
```go
type FieldSpecConfig struct {
	FieldSpec `json:",inline,omitempty" yaml:",inline,omitempty"`
 // Behavior legal values are "", "add", "replace", "remove"
	Behavior  string `json:"behavior,omitempty" yaml:"behavior,omitempty"`
}
```



### prefixsuffix transformer example:

The name prefixsuffix transformer is by default more or less modifying the name of every single resource since it is configured to use the wildcard as a kind criteria. 

The default configuration more or less looks like this:

```yaml
namePrefix:
- path: metadata/name
```

The following adjustement, let the transformer indicates to the transformer to perform its tasks normally on the metadata/name field expect for Namespace, MyCRD and Ingress.

```yaml
namePrefix:
- path: metadata/name
  version: v1
  kind: Namespace
  skip: true
- path: metadata/name
  version: v1alpha1
  group: my.org
  kind: MyCRD
  skip: true
- path: metadata/name
  group: extensions
  version: v1beta1
  kind: Ingress
  skip: true
```


### namespace transformer example:

In the same way, the namespace transformer is, by default, adding a namespace entry to any object which is not in the internal list of cluster wide resources. This performed by using a wildcard for kind criteria and excluding kind members of predefined cluster wide objects. 

The default configuration more or less looks like this:

```yaml
namespace:
- path: metadata/namespace
  create: true
```

The following example describes how indicated to kustomize not to add a metadata/namespace field to cluster wide CRD called MyCRD:

```yaml
namespace:
- path: metadata/namespace
  version: v1alpha1
  group: my.org
  kind: MyCRD
  skip: true
```


### commonlabels transformer example:

Here is an extract of the default configuration for commonLabels

```yaml
commonLabels:
- path: metadata/labels
  create: true
- path: spec/template/spec/affinity/podAffinity/requiredDuringSchedulingIgnoredDuringExecution/labelSelector/matchLabels
  create: false
  group: apps
  kind: Deployment
```

As for the namespace and prefixsuffix handling, the user can add a "skip" entry to exclude the CustomResourceDefinition from the wild card. In the following example, commonLabels will not be added for the metadata/labels section of the CustomResourceDefinition.

```yaml
    
commonLabels:
- path: metadata/labels
  version: v1beta1
  group: apiextensions.k8s.io
  kind: CustomResourceDefinition
  skip: true
```

The remaining case is linked to the handling of a none wild card entry. By default kustomize adds commonLabel of the selectors
of development. Depending on the project, this can be an issue. The `behavior: remove` or `behavior: replace` can be used to
changed the handlig of the field by the transformer.

```yaml
- path: spec/template/spec/affinity/podAntiAffinity/requiredDuringSchedulingIgnoredDuringExecution/labelSelector/matchLabels
  create: false
  group: apps
  kind: Deployment
  behavior: remove
```

### Risks and Mitigations

- Risk: Configuration drifting from Kubernetes best pratices.
- Mitigation: Good documentation and recommendations about how to keep things simple.

## Graduation Criteria

Use customer feedback to determine if we should support:

- Needs to be backward compatible with current builtin and external plugins.
- Needs to get feedback on the "skip: true" concept for wild card handling. "skip: true" can be used for external transformers.
- Needs to get feedback on the "behavior: replace" and "behavior: remove". This proposal reuse the concept used for configmapgenerator
  and secretgenerator. Concept is all really close from JSONPath behavior.


## Doc

Update Kubectl Book and Kustomize Documentation

## Test plan

Unit Tests for:

- [ ] All existing test needs to be successful.
- [ ] Can skip wild card metadata/name, metadata/namespace and metadata/labels in new CRDs.
- [ ] Can remove "selectors" fields for the list of fields mutated by the labels transformer.


### Version Skew Tests

## Implementation History

The following POC PR have been proposed:

- [feat: skip name prefix/suffix by kind #1485](https://github.com/kubernetes-sigs/kustomize/pull/1485)
- [skip kustomize transformers for paths #1491](https://github.com/kubernetes-sigs/kustomize/pull/1491)

## Alternatives

During the discussion thread related to "skip" feature [here](https://github.com/kubernetes-sigs/kustomize/pull/1485), alternative have been proposed.

Regardless if the main proposal, alternative 1 or alternative 2 is choosen, those options have in common that a set of wrapper functions and structs needs to be provided by core kustomize packages to ease the selection of the fields to mutate by the transformers.


### Alternative 1

Creating a new SkipSpec field like follow:

```go
type SkipSpec struct {
        FieldSpec             `json:",inline,omitempty" yaml:",inline,omitempty"`
        Transformers []string `json:"transformers,omitempty" yaml:"transformers,omitempty"`
}
```

The transformer configurations would include a new field skipTransformation:

```yaml
skipTransformation:
  - path: spec/selector
     kind: Service
     transformers:
     - commonlabels
  - kind: CustomResourceDefinition
     transformers:
     - nameprefix
  - kind: MyKind
     transformers:
     - nameprefix
     - namespace
  - kind: ClusterLeveledCRDs
     transformers:
     - namespace
 ```
     
### Alternative 2

The other considered alternate is to create the following struct:

```go
type SkipSpec struct {
                           FieldSpec   `json:",inline,omitempty" yaml:",inline,omitempty"`
        SkipTransformation bool        `json:"skip,omitempty" yaml:"skip,omitempty"`
}
```

and change the Transformerconfiguration passed to the transformer from:

```go
type ATransformer struct {
  FieldSpecs []config.FieldSpec
}
```

to

```go
type ATransformer struct {
  FieldSpecs []config.FieldSpec
  SkipFieldSpecs []config.FieldSpec
}
```



