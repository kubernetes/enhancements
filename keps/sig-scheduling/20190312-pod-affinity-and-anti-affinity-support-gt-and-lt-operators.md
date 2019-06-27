---
title: Pod affinity/anti-affinity supports Gt and Lt operators
authors:
  - "@wgliang"
owning-sig: sig-scheduling
reviewers:
  - "@bsalamat"
  - "@k82cn"
  - "@Huang-Wei"
approvers:
  - "@bsalamat"
  - "@k82cn"
creation-date: 2019-02-22
last-updated: 2019-04-23
status: provisional
---

 # Pod affinity/anti-affinity supports Gt and Lt operators

 ## Table of Contents

<!-- toc -->

* [Summary](#summary)
* [Motivation](#motivation)
  * [Goals](#goals)
  * [Non-Goals](#non-goals)
* [Proposal](#proposal)
  * [User Stories](#user-stories)
  * [Risks and Mitigations](#risks-and-mitigations)
* [Design Details](#design-details)
  * [Content](#content)
  * [Test Plan](#test-plan)
  * [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)

 ## Summary

 Extend the `Pod` affinity/anti-affinity operators to support `Gt` and `Lt` to provide 
 users with more elegant Pod label selection capabilities.

 ## Motivation

  We know that `Node` affinity/anti-affinity currently supports `In`, `NotIn`, `Exists`, 
`DoesNotExist`, `Gt`, `Lt`. But Pod affinity/anti-affinity only works with regular 
label selectors: `In`, `NotIn`, `Exists`, `DoesNotExist`.

 This is not an ideal situation if users want to put pods based on the label range. 
 `Pod` affinity/anti-affinity support for `Gt` and `Lt` operators will give users more 
 control.

 ### Goals

- `Pod` affinity/anti-affinity support for `Gt` and `Lt` operators.
- `Gt` and `Lt` will have the same status and influence as the original operators (`In`, 
`NotIn`, `Exists`, `DoesNotExist`).
- `Gt` and `Lt` will work with `requiredDuringSchedulingIgnoredDuringExecution`(predicate, hard requirements) and `preferredDuringScheduling`(priority, soft requirements).

 ### Non-Goals

- Changing the behavior of other label selectors, such as `ReplicaSets`, `Daemonsets`, etc.

 ## Proposal

 ### User Stories

 As an application developer, I want my application pods to be scheduled onto
one node that has pod with the "foo" tag and the tag value between "20" and "30".

- if we use the original `In` operator to implement, then we will write affinity 
like this:
 ```yaml
spec:
  affinity:
    podAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
          - key: foo
            operator: In
            values:
            - 20
            - 21
            - 22
            - 23
            ...
            - 28
            - 29
            - 30
        topologyKey: failure-domain.beta.kubernetes.io/zone
```

 This is not an ideal solution. A promising solution is to provide users with `Gt` and `Lt` 
 operators, giving users the ability to specify the scope of the tag. Users can achieve 
 this in this way:
 ```yaml
spec:
  affinity:
    podAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
          - key: foo
            operator: Gt
            values:
            - 20
          - key: foo
            operator: Lt
            values:
            - 30
        topologyKey: failure-domain.beta.kubernetes.io/zone
```

 ### Risks and Mitigations

 Along with this feature, the biggest risk should be performance. This may also be the reason 
 why the `Gt` and `Lt` operators are not supported when Pod affinity/anti-affinity is 
 first proposed. In order to understand the impact of these changes, we need to understand the their performance implications. We will perform performance/benchmark tests on this change and backward compatibility test.

 Compared to the time that inter-pod affinity was introduced, performance of the scheduler has been greatly improved(https://github.com/kubernetes/kubernetes/pull/74041#issuecomment-466191359), and the processing of Pod affinity/anti-affinity has been surprisingly Optimized(https://github.com/kubernetes/kubernetes/pull/67788).

 ## Design Details
 ### Content
  We will abstract the `Node` and `Pod` label selection feature, and finally the `NodeSelector` and `PodSelector` will be based on the `NumericAwareSelectorRequirement` implementation:

 The definition of `NodeSelector` will not change,it is defined as below:
 ```go
type NodeSelector struct {
  NodeSelectorTerms []NodeSelectorTerm
}
```

For the definition of `NodeSelectorTerm`, today the api looks like this:
 ```go
type NodeSelectorTerm struct {
  MatchExpressions []NodeSelectorRequirement
  MatchFields []NodeSelectorRequirement
}
```

We will use `NumericAwareSelectorRequirement` to replace the original `NodeSelectorRequirement` as the base selector of `NodeSelector`. So it looks like this:
 ```go
type NodeSelectorTerm struct {
  MatchExpressions []NumericAwareSelectorRequirement
  MatchFields []NumericAwareSelectorRequirement
}
```

`PodSelector` is our new selector. A pod selector represents the union of the results of one or more label queries over a set of pods. The of `PodSelector` is defined as below:
 ```go
type PodSelector struct {
  MatchLabels map[string]string
  MatchExpressions []NumericAwareSelectorRequirement
}
```

`NumericAwareSelectorRequirement` is an abstract selector implementation of `NodeSelector` and `PodSelector`. And the `NumericAwareSelectorRequirement` is defined as below:
 ```go
type NumericAwareSelectorRequirement struct {
  Key string
  Operator LabelSelectorOperator
  Values []string
}

type LabelSelectorOperator string

const (
  LabelSelectorOpIn           LabelSelectorOperator = "In"
  LabelSelectorOpNotIn        LabelSelectorOperator = "NotIn"
  LabelSelectorOpExists       LabelSelectorOperator = "Exists"
  LabelSelectorOpDoesNotExist LabelSelectorOperator = "DoesNotExist"
  LabelSelectorOpNumericallyGreater           LabelSelectorOperator = "Gt"
  LabelSelectorOpNumericallyLessthan           LabelSelectorOperator = "Lt"
)
```

 ### Test Plan

The feature has correctness, integration, performance, and e2e tests. These tests are run regularly as a part of Kubernetes presubmit and CI/CD pipeline.

<!-- /toc -->

#### Correctness Tests
Here is a list of unit tests for various modules of the feature:

- `NodeSelector` related tests
- `PodSelector` related tests
- `Gt` and `Lt` functional tests
- `NumericAwareSelectorRequirement` related tests
- Backwards compatibility - pods made with the new types should still be updatable if apiserver version is rolled back
- Forwards compatibility - all pods created today are wire-compatible (both proto and json) with the new api

#### Integration Tests
- Integration tests for `PodSelector`

#### Performance Tests
- Performance test of `Gt` and `Lt` operators

#### E2E tests
- End to end tests for `PodSelector`

 ### Graduation Criteria

 _To be filled until targeted at a release._

 ## Implementation History

 - 2019-03-12: Initial KEP sent out for reviewing.