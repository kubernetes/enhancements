# Topology awareness in Kube-scheduler

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Changes to the API](changes-to-the-api)
  - [Scheduler Plugin implementation details](#scheduler-plugin-implementation-details)
  - [Description of the Scheduling Algorithm](#description-of-the-scheduling-algorithm)
- [Alternative Solution](#alternative-solution)
  - [Exporter Daemon Implementation Details](#exporter-daemon-implementation-details)
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

Kubernetes clusters composed of nodes with complex hardware topology are becoming more prevalent.
[Topology Manager](https://kubernetes.io/docs/tasks/administer-cluster/topology-manager/) was
introduced in kubernetes as part of kubelet in order to extract the best performance out of
these high performance hybrid systems. It performs optimizations related to resource allocation
in order to make it more likely for a given pod to perform optimally. In scenarios where
Topology Manager is unable to align topology of requested resources based on the selected
Topology Manager policy, the pod is rejected with Topology Affinity Error.
[This](https://github.com/kubernetes/kubernetes/issues/84869) kubernetes issue provides
further context on how runaway pods are created because the scheduler is topology-unaware.

In order to address this issue, scheduler needs to choose a node considering resource availability
along with underlying resource topology and Topology Manager policy on the worker node.

This document describes behaviour of the Kubernetes Scheduler which takes worker node topology into account.

# Motivation

After Topology Manager was introduced, the problem of launching pod in the cluster where worker
nodes have different NUMA topology and different amount of resources in that topology became
actual. Pod could be scheduled on the node where total amount of resources are enough, but
resource distribution could not satisfy the appropriate Topology policy. In this case the pod
failed to start. Much better behaviour for scheduler would be to select appropriate node where
kubelet admit handlers may pass.

In order to enable topology aware scheduling in Kubernetes, resource topology information of the
nodes in the cluster needs to be exposed to the scheduler so that it can use it to make a more
informed scheduling decision. This KEP describes how it would be implemented.

## Goals

- Make scheduling process more precise when we have NUMA topology on the
worker node.
- Enhance the node object to capture topology information which can be referred to 
by the scheduler.

## Non-Goals

- Change the PodSpec to allow requesting a specific node topology manager policy
- This Proposal requires exposing NUMA topology information. This KEP doesn't
describe how to expose all necessary information it just declare what kind of
information is necessary.
- Changes to the TopologyManager and its policies.

# Proposal

Kube-scheduler plugin will be moved from kuberntes-sigs/scheduler-plugin (or out-of-tree)
plugin into the main tree as a built-in plugin. This plugin implements a simplified version of Topology Manager and hence is different from original topology manager algorithm. Plugin would
be disabled by default and when enabled would check for the ability to run pod only in case of single-numa-node policy on the node, since it is the most strict policy, it implies that the launch on the node with other existing policies will be successful if the condition for single-numa-node policy passed for the worker node.

To work, this plugin requires topology information of the available resource on the worker nodes.

Kubelet will be responsible for collecting all necessary resource information of the pods,
based on allocatable resources on the node and allocated resources to pods. The NUMA nodes
would be represented as Zones in Kubelet and the NodeResourceTopology would capture the 
resource information at a zone level granularity.

Once the information is captured in the NodeResourceTopology API, the scheduler can refer to
it like it refers to Node Capacity and Allocatable while making a Topology-aware Scheduling decision.


## Changes to the API

Code responsible for working with NodeResourceTopology API will be placed in the stagingit g directory
at path staging/src/k8s.io/api/node/v1/types.go.

```go

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeResourceTopology is a specification for a hardware resources
type NodeResourceTopology struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	TopologyPolicy []string `json:"topologyPolicies"`
	Zones          ZoneMap  `json:"zones"`
}

// Zone is the spec for a NodeResourceTopology resource
type Zone struct {
    Name       string           `json:"name"`
    Type       string           `json:"type"`
    Parent     string           `json:"parent,omitempty"`
    Costs      CostList         `json:"costs,omitempty"`
    Attributes AttributeList    `json:"attributes,omitempty"`
    Resources  ResourceInfoList `json:"resources,omitempty"`
}

type ResourceInfo struct {
    Name        string `json:"name"`
    Allocatable string `json:"allocatable"`
    Capacity    string `json:"capacity"`
}

type ZoneList []Zone
type ResourceInfoList []ResourceInfo

type CostInfo struct {
    Name  string `json:"name"`
    Value int	  `json:"value"`
}

type AttributeInfo struct {
    Name  string `json:"name"`
    Value string `json:"value"`
}

type CostList []CostInfo
type AttributeList []AttributeInfo

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeResourceTopologyList is a list of NodeResourceTopology resources
type NodeResourceTopologyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NodeResourceTopology `json:"items"`
}

```


Where TopologyPolicy may have following values: none, best-effort, restricted, single-numa-node.
The current policies of TopologyManager can't coexist together at the same time, but in future such kind of policies could appear.
For example we can have policy for HyperThreading and it can live with NUMA policies.

To use these policy names both in kube-scheduler and in kubelet, string constants of these labels should be moved from pkg/kubelet/cm/topologymanager/ and pkg/kubelet/apis/config/types.go to pkg/apis/core/types.go a one single place.

## Plugin implementation details

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
			for _, numaNode := range zones {
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



# Alternative Solution
Enable an external daemon to expose resource information along with NUMA topology of a node as a
[CRD][1]. One way of doing this is to enhance Node Feature Discovery [daemon](https://github.com/kubernetes-sigs/node-feature-discovery) or a standalone component like [Resource Topology Exporter](https://github.com/k8stopologyawareschedwg/resource-topology-exporter) that runs on each node in the cluster as a daemonset and collect resources allocated to running pods along with associated topology (NUMA nodes) and provides information of the available resources (with numa node granularity) through a CRD instance created per node. The CRs created
per node are then later used by the scheduler to identify which topology policy is enabled and make a Topology aware placement decision.

# Exporter Daemon Implementation Details

Podresources interface of the kubelet is described in

[pkg/kubelet/apis/podresources/v1/api.proto](https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/apis/podresources/v1/api.proto)

it is available for every process on the worker node by
unix domain socket situated by the following path:

```go
filepath.Join(kl.getRootDir(), config.DefaultKubeletPodResourcesDirName)
```

it could be used to collect used resources on the worker node and to evaluate
its NUMA assignment (by device id).

Podresources can be used to obtain initial information on resources of the worker node.


```proto
syntax = "proto3";

package v1;

// PodResourcesLister is a service provided by the kubelet that provides information about the
// node resources consumed by pods and containers on the node
service PodResourcesLister {
    rpc List(ListPodResourcesRequest) returns (ListPodResourcesResponse) {}
    rpc GetAllocatableResources(AllocatableResourcesRequest) returns (AllocatableResourcesResponse) {}
}

message AllocatableResourcesRequest {}

// AvailableResourcesResponses contains informations about all the devices known by the kubelet
message AllocatableResourcesResponse {
    repeated ContainerDevices devices = 1;
    repeated int64 cpu_ids = 2;
}

// ListPodResourcesRequest is the request made to the PodResources service
message ListPodResourcesRequest {}

// ListPodResourcesResponse is the response returned by List function
message ListPodResourcesResponse {
    repeated PodResources pod_resources = 1;
}

// PodResources contains information about the node resources assigned to a pod
message PodResources {
    string name = 1;
    string namespace = 2;
    repeated ContainerResources containers = 3;
}

// ContainerResources contains information about the resources assigned to a container
message ContainerResources {
    string name = 1;
    repeated ContainerDevices devices = 2;
    repeated int64 cpu_ids = 3;
}

// Topology describes hardware topology of the resource
message TopologyInfo {
	repeated NUMANode nodes = 1;
}

// NUMA representation of NUMA node
message NUMANode {
	int64 ID = 1;
}

// ContainerDevices contains information about the devices assigned to a container
message ContainerDevices {
    string resource_name = 1;
    repeated string device_ids = 2;
    TopologyInfo topology = 3;
}

```



## Topology format

Available resources with topology of the node should be stored in CRD. Format of the topology described
[in this document][1].

The daemon which runs outside of the kubelet will collect all necessary information on running pods, based on allocatable resources of the node and consumed resources by pods it will provide available resources in CRD, where one CRD instance represents one worker node. The name of the CRD instance is the name of the worker node.

## CRD API

Format of the topology is described [in this document](https://docs.google.com/document/d/12kj3fK8boNuPNqob6F_pPU9ZTaNEnPGaXEooW1Cilwg/edit).

[Code][3] responsible for working with NodeResourceTopology CRD API will be placed in the staging directory at path staging/src/k8s.io/noderesourcetopology-api.

At the time of writing this KEP, the CRD API is stored in Topology-aware Scheduling github organization in [noderesourcetopology-api](https://github.com/k8stopologyawareschedwg/noderesourcetopology-api)

## Plugin implementation details

Since topology of the node is stored in the CRD, kube-scheduler subscribes for updates of appropriate CRD type. Kube-scheduler uses informers generated with the name NodeTopologyInformer. NodeTopologyInformer runs in NodeResourceTopologyMatch plugin.

### Topology information in the NodeResourceTopologyMatch plugin

Once NodeResourceTopology is received NodeResourceTopologyMatch plugin keeps it in its own state of type NodeTopologyMap. This state is used every time when scheduler needs to make a decision based on node topology.

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

Separate tests for changes to Kubelet will also should be implemented.

* Integration Tests
   *  Default configuration (this plugin is disabled)
     * no side effect on basic scheduling flow (and performance)

   *  Enable this plugin
     * basic workflow of this feature works (decision by scheduler is admitted by kubelet)
     * basic negative path of this feature works (decision by scheduler is rejected by kubelet)

* End-to-end tests

Integration and End-to-end would Implementation of it does not constitute a difficulty, but requires appropriate multi-numa hardware for comprehensive testing of this feature. Comprehensive E2E testing of this would be done in order to graduate this feature from Alpha to Beta.

# Graduation criteria

* Alpha (v1.23)

Following changes are required:
- [ ] Introducing a Topolgy information as part of Node API
- [ ] New `kube scheduler plugin` NodeResourceTopologyMatch.
    - [ ] Implementation of Filter
- [ ] Unit tests and integration tests from [Test plans](#test-plans).

* Beta
- [ ] Add node E2E tests.
- [ ] Provide beta-level documentation.

# Production Readiness Review Questionnaire

# TBD
    

# Implementation history

- 2021-06-10: Initial KEP sent out for review, including Summary, Motivation, Proposal, Test plans and Graduation criteria.

[1]: https://docs.google.com/document/d/12kj3fK8boNuPNqob6F_pPU9ZTaNEnPGaXEooW1Cilwg/edit
[2]: https://github.com/kubernetes-sigs/node-feature-discovery
[3]: https://github.com/kubernetes/noderesourcetopology-api