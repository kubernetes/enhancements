---
title: Explain Categories and Expand
authors:
  - "@feloy"
owning-sig: sig-cli
participating-sigs:
reviewers: TBD
approvers: TBD
editor: TBD
creation-date: 2019-07-21
last-updated: 2019-07-21
status: provisional
see-also:
replaces:
superseded-by:
---

# Explain Categories and Expand

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Categorize the fields of PodSpec](#categorize-the-fields-of-podspec)
    - [Expand some fields of Container](#expand-some-fields-of-container)
    - [Order fields of VolumeMount](#order-fields-of-volumemount)
- [Compatibility](#compatibility)
<!-- /toc -->

## Summary

Actually, documentation (especially `kubectl explain`) describing Kinds of the Kubernetes API
lists all the fields of a Kind in the alphabetic order, and there is not enough information in the declaration of these fields to list them in a specific order.

We propose to add an annotation `x-kubernetes-explain-category` applicable to fields, that represents both
the name of the category and the order of this category related to the other categories of the same Kind.

In addition, we propose to add an annotation `x-kubernetes-explain-expand` applicable to fields, 
that indicates if the field should be expanded in documentation in order to display inline the content 
of this field.

## Proposal

### User Stories

#### Categorize the fields of PodSpec

To categorize a field, it is necessary to add a comment to the field in `types.go`:

```
// +k8s:openapi-gen=x-kubernetes-explain-category:1.category-name
```

The value (here *1.category-name*) contains two parts, separated by a dot (.).
The first part is an integer that indicates the order of the category related to the
other categories of the field. The second part is a name of the category, with spaces
replaced by dashes (-).

Example:
```go
type PodSpec struct {
  // +k8s:openapi-gen=x-kubernetes-explain-category:2.volumes
  Volumes []Volume
  // +k8s:openapi-gen=x-kubernetes-explain-category:1.containers
  InitContainers []Container
  // +k8s:openapi-gen=x-kubernetes-explain-category:1.containers
  Containers []Container
  // +k8s:openapi-gen=x-kubernetes-explain-category:4.lifecycle
  RestartPolicy RestartPolicy
  [...]
```

As a result, the documentation of PodSpec would be written by `kubectl explain` as:
```
KIND:     Pod
VERSION:  v1

RESOURCE: spec <Object>

DESCRIPTION:
     ...
     
FIELDS:
  CONTAINERS

   initContainers        <[]Object>
     ...

   containers            <[]Object>
     ...

  VOLUMES

   volumes               <[]Object>
     ...

  LIFECYCLE
  
   restartPolicy         <Object>
    ...
```

#### Expand some fields of Container

To indicate that a field should be expanded in the documentation, it is necessary to add a comment to the field in `types.go`:

```
// +k8s:openapi-gen=x-kubernetes-explain-expand:
```

Example:

```go
type Container struct {
  // +k8s:openapi-gen=x-kubernetes-explain-expand:
  Ports []ContainerPort
  [...]
}
```

As a result, the documentation of Container would be written by `kubectl explain` as:

```
KIND:     Pod
VERSION:  v1

RESOURCE: containers <[]Object>

DESCRIPTION:
     ...

FIELDS:

   [first Container fields...]

   ports     <[]Object>
     <description of ports>

     hostIP     <string>
       <description of hostIP>

     hostPort   <sint32>
       <description of hostPort>

     [other ports fields]

   [other Container fields...]
```

#### Order fields of VolumeMount

In order to sort fields without indicating a category, it is necessary to add a comment to the field in `types.go`:

```
// +k8s:openapi-gen=x-kubernetes-explain-category:1.
```

The value (here *1.*) should be an integer followed by a dot (.), indicating 
the order of this field related to the other fields in the same Kind.

Example:
```go
type VolumeMount struct {
  // +k8s:openapi-gen=x-kubernetes-explain-category:1.
  Name string
  // +k8s:openapi-gen=x-kubernetes-explain-category:2.
  ReadOnly bool 
  // +k8s:openapi-gen=x-kubernetes-explain-category:2.
  MountPath string
  // +k8s:openapi-gen=x-kubernetes-explain-category:3.
  SubPath string 
}  
```

Without annotations, the fields would be displayed in the following order (alphabetically):
- mountPath
- name
- readOnly
- subPath

With the annotations, the fields can now be displayed in this order:
- name
- mountPath
- readOnly
- subPath

## Compatibility

In order to maintain compatibility with `types.go` that do not use 
these annotations or use them partially, the non annotated fields 
should be implicitly affected to a category `99.other` by `kubectl explain` or other documentation systems. So, these fields would be placed after the annotated ones, in alphabetic order.

If no category is explicitly defined for a Kind, the OTHER category will be the only one. In this case, the category name of this unique "OTHER" category should not be printed by `kubectl explain`.
