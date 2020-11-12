# KEP-2143: Generic Node Resource Scheduler Plugin

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Configuration](#configuration)
  - [Test Plan](#test-plan)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The generic node resource scheduler plugin combines several in-tree and out-of-tree
scheduler plugins scoring nodes based on resource utilization,
as well as some requested ones, together into a generic one.
The generic plugin takes effects in the "Scoring" extension point of the scheduling framework,
while it could be configured in one place
to care about different portions (allocatable, allocated or available) of the resources,
valued on different values (absolute values, or ratio to some other values),
scored in different flavours (prefer higher, or lower or whatever values),

## Motivation

As this proposal is came up, several scheduler plugins are available when considering 
scheduling pods base on node resource utilization:

These plugins fit common use cases well, and have served a long time.
While there are certainly missing cases.
These missing functionalities are some way similar with existing ones, or combinations of them.
Instead of adding more plugins and copying code everywhere, like "NodeResourcesAllocatable" did,
a generic node resource scheduler plugin is proposed.
The generic plugin covers these resource considerations together,
and could be configured to enable or disable these functions individually, or configure each of them.

### Goals

- Replace previously mentioned scheduler plugins with a generic one.
- Make it easy to understand and configure these node resource related scoring functions by being configured in one place.
- Cover more use cases in similar situations.
- Easy to combine these considerations together.
- Let users to choose scoring styles they like.

## Proposal

A new scheduler plugin is proposed to score nodes based on their resources.
The proposed scheduler plugin is a successor of several in-tree and out-of-tree 
scheduler plugins working with node resources in different ways.
It will replace these separated plugins with a single generic one,
and covers more similar but different or more complicated use cases.
It defines a new type for users to configure the plugin in a easy-to-understand 
and unified way.

Following scheduler plugins will be replaced:
| plugin name                       | code location | cares about | values on            | prefers values  | comments                                                      |
| --------------------------------- | ------------- | ----------- | -------------------- | --------------- | ------------------------------------------------------------- |
| "NodeResourcesLeastAllocated"     | in tree       | left        | ratio to allocatable | greater         |                                                               |
| "NodeResourcesMostAllocated"      | in tree       | allocated   | ratio to allocatable | greater         |                                                               |
| "RequestedToCapacityRatio"        | in tree       | left        | ratio to allocatable | greater         | scores are mapped from values by a custom non-linear function |
| "NodeResourcesBalancedAllocation" | in tree       | requesting  | ratio to allocatable | balanced        |                                                               |
| "NodeResourcesAllocatable"        | out of tree   | allocatable | absolute value       | greater or less |                                                               |

And following new use cases will be covered:
- Put the pod on some node "just fits" the requests: "Least Left".
- Let small nodes be full earlier: "Least Allocatable" + "Least Left".

### User Stories

#### Story 1

I want my pods being scheduled to nodes with fewer gpus left, to make sure there will be
enough "empty" nodes for upcoming pods requesting all gpus in a node.
I could configure the plugin as follow:

```yaml
left:
  values: AbsoluteValue
  prefer: Least
  resources:
    nvidia.com/gpu: 1
```

#### Story 2

I want "small" nodes be full earlier.
I could use following configuration:

```yaml
allocatable:
  values: AbsoluteValue
  prefer: Least
  resources:
    cpu: 10000000000000
    memory: 10

left:
  values: RatioToAllocatable
  prefer: Least
  resources:
    cpu: 1000000000000
    memory: 1
```

### Risks and Mitigations

## Design Details

Node resources are defined as different portions:
- Allocatable: the resources of a node that are available for scheduling, 
  taken from Node.Status.Allocatable, or NodeInfo.Allocatable.
- Requesting: resources requested by current pod.
- Allocated: Total requested resources of all pods on this node with a minimum value
  applied to each container's CPU and memory requests, taken from NodeInfo.NonZeroRequested,
  plus Requesting.
- Left: Allocatable - Allocated.

These resources could be valued by:
- its absolute value,
- or the ratio to allocatable resources.

And then be scored with:
- higher sum,
- or lower sum,
- or more balanced among resources,
- or less balanced among resources.

Users could select one or more combinations of "resource portion" + "value method" + "score flavour"
and assign them each a map from resource names to weights to configure a desired scheduling behavior.

### Configuration

A type is defined for users to configure the plugin.

```go
// GenericNodeResourcesArgs configures how the generic node resource scheduler plugin behaviors
type GenericNodeResourcesArgs struct {
	metav1.TypeMeta

  // score nodes based on their allocatable resources
	// +optional
	Allocatable *NodeResourcesPartArgs
	// score nodes based on their resources already allocated
	// +optional
	Allocated *NodeResourcesPartArgs
	// score nodes based on their left resources
	// +optional
	Left *NodeResourcesPartArgs
	// score nodes based on (usually the ratio to nodes' allocatable/available resources of) resources requested by the pod
	// +optional
	Requesting *NodeResourcesPartArgs
}

type NodeResourcesPartArgs struct {
  // how to value the resources
  Values ValueResourceOn
  // how to map the values to a final score (to sort the values)
	Prefer ScoreResourceBy

	// Resources to be considered when scoring.
	// The default resource set includes "cpu" and "memory" with an equal weight.
	// Allowed weights go from 1 to 100.
	Resources []ResourceSpec
}

type ValueResourceOn string

const (
	ValuesOnAbsoluteValue      ValueResourceOn = "AbsoluteValue"
	ValuesOnRatioToAllocatable ValueResourceOn = "RatioToAllocatable"
)

type ScoreResourceBy string

const (
	PreferMostInSum     ScoreResourceBy = "Most"
  PreferLeastInSum    ScoreResourceBy = "Least"
  PreferMostBalanced  ScoreResourceBy = "MostBalanced"
  PreferLeastBalanced ScoreResourceBy = "LeastBalanced"
)
```

### Test Plan

Existing tests for the old scheduler plugins must be kept or considered.
More tests should be added when implementing.

TBD

## Alternatives

New use cases could be covered by adding other scheduler plugins to out-of-tree scheduler-plugins repo. 
But it will be hard to find and configure all these plugins.
