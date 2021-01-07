# KEP-1819: Scheduler extender

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Design Details](#design-details)
  - [Configuration](#configuration)
  - [Interface](#interface)
    - [Filter](#filter)
    - [Prioritize](#prioritize)
    - [Bind](#bind)
    - [Preempt](#preempt)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
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
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubernetes scheduler schedules based on resources managed by Kubernetes.
Scheduling based on opaque resource counting helps extending this further. But
when there is a need for contextual scheduling for resources managed outside of
kubernetes, there is no mechanism to do it today.

The proposal is to make kubernetes scheduler extensible by adding the capability
to make http calls out to another endpoint to help achieve this functionality.

**NOTE**: This KEP was done in retrospect and it was based on the original
[extender proposal](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/scheduler_extender.md).

## Motivation

There are three ways to add new scheduling rules to Kubernetes:

- Update existing or add new scheduler plugins and recompiling
- Implementing your own scheduler process that runs instead of, or alongside of, the standard Kubernetes scheduler
- Implementing a "scheduler extender" process that the standard Kubernetes scheduler calls out to as a final pass when making scheduling decisions

The first and the second approach have a high bar for beginners. Scheduler
plugins must be written in Go and are compiled with Kubernetes scheduler code.
Implementing a new scheduler from scratch to replace the default one requires a
lot of effort if supporting all features is required (e.g. spreading,
affinity, taints).

Extending the functionality of scheduler via a separate process has some
drawbacks, such as bad performance and cache inconsistency, but it also has
several advantages:

- It can extend the functionality of an existing scheduler without recompiling
  the binary
- The extender can be written in any languages
- Once implemented, it can be used to extend different versions of kube-scheduler

This document describes the third approach. If you don't care about latency and
cache consistency and don't want to maintain your own build of the default
scheduler, then this approach is likely the better choice. If you care about
performance and want maximum ability to customize the scheduler's behavior,
then developing a new plugin is likely the better choice.

### Goals

- Extend default scheduler by registering external webhooks to tweak scheduling decision on different phases

### Non-Goals

- Solve limitations due to running extender as a separate process, e.g. cache
  inconsistency, performance

## Design Details

When scheduling a pod, the extender allows an external process to filter and
prioritize nodes. Two separate http/https calls are issued to the extender, one
for "filter" and one for "prioritize" actions. If the pod cannot be scheduled,
the scheduler will try to preempt lower priority pods from nodes and send them
to extender "preempt" verb if configured. The extender can return a subset of
nodes and new victims back to the scheduler. Additionally, the extender can
choose to bind the pod to apiserver by implementing the "bind" action.

### Configuration

To use the extender, you must create a scheduler configuration file. The
configuration specifies how to reach the extender, whether to use http or https
and the timeout.

```go
// Holds the parameters used to communicate with the extender. If a verb is unspecified/empty,
// it is assumed that the extender chose not to provide that extension.
type ExtenderConfig struct {
	// URLPrefix at which the extender is available
	URLPrefix string `json:"urlPrefix"`
	// Verb for the filter call, empty if not supported. This verb is appended to the URLPrefix when issuing the filter call to extender.
	FilterVerb string `json:"filterVerb,omitempty"`
	// Verb for the preempt call, empty if not supported. This verb is appended to the URLPrefix when issuing the preempt call to extender.
	PreemptVerb string `json:"preemptVerb,omitempty"`
	// Verb for the prioritize call, empty if not supported. This verb is appended to the URLPrefix when issuing the prioritize call to extender.
	PrioritizeVerb string `json:"prioritizeVerb,omitempty"`
	// Verb for the bind call, empty if not supported. This verb is appended to the URLPrefix when issuing the bind call to extender.
	// If this method is implemented by the extender, it is the extender's responsibility to bind the pod to apiserver.
	BindVerb string `json:"bindVerb,omitempty"`
	// The numeric multiplier for the node scores that the prioritize call generates.
	// The weight should be a positive integer
	Weight int `json:"weight,omitempty"`
	// EnableHttps specifies whether https should be used to communicate with the extender
	EnableHttps bool `json:"enableHttps,omitempty"`
	// TLSConfig specifies the transport layer security config
	TLSConfig *ExtenderTLSConfig `json:"tlsConfig,omitempty"`
	// HTTPTimeout specifies the timeout duration for a call to the extender. Filter timeout fails the scheduling of the pod. Prioritize
	// timeout is ignored, k8s/other extenders priorities are used to select the node.
	HTTPTimeout time.Duration `json:"httpTimeout,omitempty"`
	// NodeCacheCapable specifies that the extender is capable of caching node information,
	// so the scheduler should only send minimal information about the eligible nodes
	// assuming that the extender already cached full details of all nodes in the cluster
	NodeCacheCapable bool `json:"nodeCacheCapable,omitempty"`
	// ManagedResources is a list of extended resources that are managed by
	// this extender.
	// - A pod will be sent to the extender on the Filter, Prioritize and Bind
	//   (if the extender is the binder) phases iff the pod requests at least
	//   one of the extended resources in this list. If empty or unspecified,
	//   all pods will be sent to this extender.
	// - If IgnoredByScheduler is set to true for a resource, kube-scheduler
	//   will skip checking the resource in filter plugins.
	// +optional
	ManagedResources []ExtenderManagedResource `json:"managedResources,omitempty"`
	// Ignorable specifies if the extender is ignorable, i.e. scheduling should not
	// fail when the extender returns an error or is not reachable.
	Ignorable bool `json:"ignorable,omitempty"
}
```

A sample scheduler configuration file with extender specs:

```yaml
apiVersion: kubescheduler.config.k8s.io/v1beta1
kind: KubeSchedulerConfiguration
extenders:
- urlPrefix: "http://127.0.0.1:12345/api/scheduler"
  filterVerb: "filter"
  preemptVerb: "preempt"
  prioritizeVerb: "prioritize"
  bindVerb: "bind"
  enableHttps: false
  nodeCacheCapable: false
  managedResources:
  - name: opaqueFooResource
    ignoredByScheduler: true
  ignorable: false
```

Multiple extenders can be configured and will be called sequentially by the
scheduler.

### Interface

#### Filter

Arguments passed to the `FilterVerb` endpoint on the extender are the set of
nodes filtered through the k8s filter plugins and the pod.

```go
// ExtenderArgs represents the arguments needed by the extender to filter/prioritize
// nodes for a pod.
type ExtenderArgs struct {
	// Pod being scheduled
	Pod   api.Pod      `json:"pod"`
	// List of candidate nodes where the pod can be scheduled
	Nodes api.NodeList `json:"nodes"`
	// List of candidate node names where the pod can be scheduled; to be
	// populated only if Extender.NodeCacheCapable == true
	NodeNames *[]string
}
```

The "filter" call returns a list of nodes:

```go
// FailedNodesMap represents the filtered out nodes, with node names and failure messages
type FailedNodesMap map[string]string

// ExtenderFilterResult represents the results of a filter call to an extender
type ExtenderFilterResult struct {
	// Filtered set of nodes where the pod can be scheduled; to be populated
	// only if Extender.NodeCacheCapable == false
	Nodes *v1.NodeList
	// Filtered set of nodes where the pod can be scheduled; to be populated
	// only if Extender.NodeCacheCapable == true
	NodeNames *[]string
	// Filtered out nodes where the pod can't be scheduled and the failure messages
	FailedNodes FailedNodesMap
	// Filtered out nodes where the pod can't be scheduled and preemption would
	// not change anything. The value is the failure message same as FailedNodes.
	// Nodes specified here takes precedence over FailedNodes.
	FailedAndUnresolvableNodes FailedNodesMap
	// Error message indicating failure
	Error string
}
```

The "filter" call may prune the set of nodes based on its filter plugins.

Nodes in both `FailedNodesMap` and `FailedAndUnresolvableNodes` are
unschedulable, except the nodes in the latter will be skipped in preemption
phase.

When multiple extenders are configured, unschedulable nodes will not be passed
to subsequent extenders. It's recommended to order the extenders that may
report `UnschedulableAndUnresolvable` ahead of others. This can improve the
preemption performance.

#### Prioritize

Arguments passed to the `PrioritizeVerb` endpoint on the extender are the set of
nodes filtered through the k8s filter plugins and the pod.

It's the same as the arguments of Filter call, see [Filter](#filter).

The "prioritize" call returns priorities for each node:

```go
// HostPriority represents the priority of scheduling to a particular host, higher priority is better.
type HostPriority struct {
	// Name of the host
	Host string
	// Score associated with the host
	Score int64
}

// HostPriorityList declares a []HostPriority type.
type HostPriorityList []HostPriority
```

Scores returned by the "prioritize" call are added to the k8s scores (computed
through its priority functions) and used for final host selection.

#### Bind

The "bind" call is used to delegate the bind of a pod to a node to the extender. It can be optionally implemented by the extender. When it is implemented, it is the extender's responsibility to issue the bind call to the apiserver. Pod name, namespace, UID and Node name are passed to the extender.

```go
// ExtenderBindingArgs represents the arguments to an extender for binding a pod to a node.
type ExtenderBindingArgs struct {
	// PodName is the name of the pod being bound
	PodName string
	// PodNamespace is the namespace of the pod being bound
	PodNamespace string
	// PodUID is the UID of the pod being bound
	PodUID types.UID
	// Node selected by the scheduler
	Node string
}

// ExtenderBindingResult represents the result of binding of a pod to a node from an extender.
type ExtenderBindingResult struct {
	// Error message indicating failure
	Error string
}
```

#### Preempt

The "preempt" call makes the extender to return a subset of given nodes and new victims on the nodes.

```
// ExtenderPreemptionArgs represents the arguments needed by the extender to preempt pods on nodes.
type ExtenderPreemptionArgs struct {
	// Pod being scheduled
	Pod *v1.Pod
	// Victims map generated by scheduler preemption phase
	// Only set NodeNameToMetaVictims if Extender.NodeCacheCapable == true. Otherwise, only set NodeNameToVictims.
	NodeNameToVictims     map[string]*Victims
	NodeNameToMetaVictims map[string]*MetaVictims
}

// ExtenderPreemptionResult represents the result returned by preemption phase of extender.
type ExtenderPreemptionResult struct {
	NodeNameToMetaVictims map[string]*MetaVictims
}
```

## Implementation History

- 2015-09-04 First [proposal and implementation PR](https://github.com/kubernetes/kubernetes/pull/13580) sent out for review
- 2017-04-25 Implements [Bind interface](https://github.com/kubernetes/kubernetes/pull/44883)
- 2018-01-24 Implements [Preemption interface](https://github.com/kubernetes/kubernetes/pull/58717)
- 2020-06-01 Convert old design proposal to a KEP and sent it out for review
- 2020-06-09 Extend `ExtenderFilterResult` to include unresolvable nodes and sent it out for review
