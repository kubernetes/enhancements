# Topology Aware Scheduling

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Topology format](#topology-format)
  - [CRD API](#crd-api)
  - [Plugin implementation details](#plugin-implementation-details)
    - [Topology information in the NodeResourceTopologyMatch plugin](#topology-information-in-the-noderesourcetopologymatch-plugin)
    - [Description of the Algorithm](#description-of-the-algorithm)
  - [Accessing NodeResourceTopology CRD](#accessing-noderesourcetopology-crd)
- [Use cases](#use-cases)
- [Known limitations](#known-limitations)
- [Test plans](#test-plans)
- [Graduation criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
- [Implementation history](#implementation-history)
<!-- /toc -->

# Summary

This document describes behaviour of the Kubernetes Scheduler which takes worker node topology into account .

# Motivation

After Topology Manager was introduced the problem of launching pod in the
cluster where worker nodes have different NUMA topology and different amount
of resources in that topology became actual. Pod could be scheduled on the node
where total amount of resources are enough, but resource distribution could not
satisfy the appropriate Topology policy. In this case the pod failed to start. Much
better behaviour for scheduler would be to select appropriate node where kubelet admit
handlers may pass.


## Goals

- Make scheduling process more precise when we have NUMA topology on the
worker node.

## Non-Goals

- Change the PodSpec to allow requesting a specific node topology manager policy
- This Proposal requires exposing NUMA topology information. This KEP doesn't
describe how to expose all necessary information it just declare what kind of
information is necessary.

# Proposal

Kube-scheduler built-in plugin will be added to the main tree. This plugin
implements a simplified version of Topology Manager and hence is different from original topology manager algorithm.
Plugin would be disabled by default and when enabled would check for the ability to run pod only in case of single-numa-node policy on the
node, since it is the most strict policy, it implies that the launch on the node with
other existing policies will be successful if the condition for single-numa-node policy passed for the worker node.
Proposed plugin will use [CRD][1] to identify which topology policy is enabled on the node.
To work, this plugin requires topology information of the available resource on the worker nodes.

## Topology format

Available resources with topology of the node should be stored in CRD. Format of the topology described
[in this document][1].

The daemon which runs outside of the kubelet will collect all necessary information on running pods, based on allocatable resources of the node and consumed resources by pods it will provide available resources in CRD, where one CRD instance represents one worker node. The name of the CRD instance is the name of the worker node.

## CRD API

[Code][3] responsible for working with NodeResourceTopology CRD API will be placed in the staging directory at path staging/src/k8s.io/noderesourcetopology-api.

## Plugin implementation details

Since topology of the node is stored in the CRD, kube-scheduler should be subscribed for updates of appropriate CRD type. Kube-scheduler will use informers which will be generated with the name NodeTopologyInformer. NodeTopologyInformer will run in NodeResourceTopologyMatch plugin.

### Topology information in the NodeResourceTopologyMatch plugin

Once NodeResourceTopology is received NodeResourceTopologyMatch plugin keeps it in its own state of type
NodeTopologyMap. This state will be used every time when scheduler needs to make a decidion based on node topology.

```go

type NodeTopologyMap map[string]topologyv1alpha1.NodeResourceTopology

// NodeResourceTopology is a specification for a Foo resource
type NodeResourceTopology struct {
    metav1.TypeMeta           `json:",inline"`
    metav1.ObjectMeta         `json:"metadata,omitempty"`

    TopologyPolicies []string `json:"topologyPolicies"`
    Zones map[string]Zone     `json:"zones"`
}

// Zone is the spec for a NodeResourceTopology resource
type Zone struct {
    Type       string
    Parent     string
    Costs      map[string]int
    Attributes map[string]int
    Resources  ResourceInfoMap
}

type ResourceInfo struct {
    Allocatable int
    Capacity    int
}

type ResourceInfoMap map[string]ResourceInfo
```
Where TopologyPolicies may have following values: none, best-effort, restricted, single-numa-node.
The current policies of TopologyManager can't coexist together at the same time, but in future such kind of policies could appear.
For example we can have policy for HyperThreading and it can live with NUMA policies.

To use these policy names both in kube-scheduler and in kubelet, string constants of these labels should be moved from pkg/kubelet/cm/topologymanager/ and pkg/kubelet/apis/config/types.go to pkg/apis/core/types.go a one single place.

NUMAID is an auxiliary field since scheduler version of Topology Manager doesn't make a real assignment.

### Description of the Algorithm

The algorithm which implements single-numa-node policy is following:

```go
	if qos == v1.PodQOSBestEffort {
		return nil
	}

	zeroQuantity := resource.MustParse("0")
	for _, container := range containers {
		bitmask := bm.NewEmptyBitMask()
		bitmask.Fill()
		for resource, quantity := range container.Resources.Requests {
			resourceBitmask := bm.NewEmptyBitMask()
			for _, numaNode := range nodes {
				numaQuantity, ok := numaNode.Resources[resource]
				// if can't find requested resource on the node - skip (don't set it as available NUMA node)
				// if unfound resource has 0 quantity probably this numa node can be considered
				if !ok && quantity.Cmp(zeroQuantity) != 0{
					continue
				}
				// Check for the following:
				// 1. set numa node as possible node if resource is memory or Hugepages (until memory manager will not be merged and
				// memory will not be provided in CRD
				// 2. set numa node as possible node if resource is cpu and it's not guaranteed QoS, since cpu will flow
				// 3. set numa node as possible node if zero quantity for non existing resource was requested (TODO check topology manaager behaviour)
				// 4. otherwise check amount of resources
				if resource == v1.ResourceMemory ||
					strings.HasPrefix(string(resource), string(v1.ResourceHugePagesPrefix)) ||
					resource == v1.ResourceCPU && qos != v1.PodQOSGuaranteed ||
					quantity.Cmp(zeroQuantity) == 0 ||
					numaQuantity.Cmp(quantity) >= 0 {
					resourceBitmask.Add(numaNode.NUMAID)
				}
			}
			bitmask.And(resourceBitmask)
		}
		if bitmask.IsEmpty() {
			// definitely we can't align container, so we can't align a pod
			return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("Can't align container: %s", container.Name))
		}
	}
	return nil
}
```
## Accessing NodeResourceTopology CRD

In order to allow the scheduler (deployed as a pod) to access NodeResourceTopology CRD instances, ClusterRole and ClusterRoleBinding would have to be configured as below:

``` yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: noderesourcetopology-handler
rules:
- apiGroups: ["topology.node.k8s.io"]
  resources: ["noderesourcetopologies"]
  verbs: ["*"]
- apiGroups: ["rbac.authorization.k8s.io"]
  resources: ["*"]
  verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: handle-noderesourcetopology
subjects:
- kind: ServiceAccount
  name: noderesourcetopology-account
  namespace: default
roleRef:
  kind: ClusterRole
  name: noderesourcetopology-handler
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: noderesourcetopology-account
```

`serviceAccountName: noderesourcetopology-account` would have to be added to the manifest file of the scheduler deployment file.

# Use cases

Numbers of kubernetes worker nodes on bare metal with NUMA topology. TopologyManager feature gate enabled on the nodes. In this configuration, the operator does not want that in the case of an unsatisfactory host topology, it should be re-scheduled for launch, but wants the scheduling to be successful the first time.

# Known limitations

Kube-scheduler makes an assumption about current resource usage on the worker node, since kube-scheduler knows which pod assigned to node. This assumption makes right after kube-scheduler choose a node. But in case of scheduling with NUMA topology only TopologyManager on the worker node knows exact NUMA node used by pod, this information about NUMA node delivers to kube-scheduler with latency. In this case kube-scheduler will not know actual NUMA topology until topology exporter will send it back. It could be mitigated if kube-scheduler in proposed plugin will add a hint on which NUMA id pod could be assigned, further Topology Manager on the worker node may take it into account.

# Test plans

It would be ensured that the components developed or modified for this feature can be easily tested.

* Unit Tests

Unit test for scheduler plugin (pkg/scheduler/framework/plugins/noderesources/node_resource_topology_match.go)
pkg/scheduler/framework/plugins/noderesources/node_resource_topology_match_test.go which test the plugin.

Separate tests for CRD informer also should be implemented.

* Integration Tests
   *  Default configuration (this plugin is disabled)
     * no side effect on basic scheduling flow (and performance)
     * no side effect no matter the CRD is installed or not

   *  Enable this plugin
     * basic workflow of this feature works (decision by scheduler is admitted by kubelet)
     * basic negative path of this feature works (decision by scheduler is rejected by kubelet)
     * verify the behavior when the CRD is and isn't installed

* End-to-end tests

Integration and End-to-end would Implementation of it does not constitute a difficulty, but requires appropriate multi-numa hardware for comprehensive testing of this feature. Comprehensive E2E testing of this would be done in order to graduate this feature from Alpha to Beta.

# Graduation criteria

* Alpha (v1.20)

Following changes are required:
- [ ] CRD informer used in kubernetes as staging project
- [ ] New `kube scheduler plugin` NodeResourceTopologyMatch.
    - [ ] Implementation of Filter
- [ ] Unit tests and integration tests from [Test plans](#test-plans).

* Beta
- [ ] Add node E2E tests.
- [ ] Provide beta-level documentation.

# Production Readiness Review Questionnaire

## Feature enablement and rollback
* **How can this feature be enabled / disabled in a live cluster?**
    - This plugin doesn't require special feature gate, but it expects: TopologyManager and CPUManager feature gate enabled on the worker node\

# Implementation history

- 2020-06-12: Initial KEP sent out for review, including Summary, Motivation, Proposal, Test plans and Graduation criteria.

[1]: https://docs.google.com/document/d/12kj3fK8boNuPNqob6F_pPU9ZTaNEnPGaXEooW1Cilwg/edit
[2]: https://github.com/kubernetes-sigs/node-feature-discovery
[3]: https://github.com/kubernetes/noderesourcetopology-api
