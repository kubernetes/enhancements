# Exposure Node Resources With NUMA Topology Information

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Design based on podresources interface of the kubelet](#design-based-on-podresources-interface-of-the-kubelet)
  - [API](#api)
  - [Integration into Node Feature Discovery](#integration-into-node-feature-discovery)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Annotation approach](#annotation-approach)
  - [NUMA specification in ResourceName](#numa-specification-in-resourcename)
  - [Design based on CRI](#design-based-on-cri)
    - [Drawbacks](#drawbacks)
<!-- /toc -->

## Summary

Kubernetes clusters composed of nodes with complex hardware topology are becoming more prevalent.
[Topology Manager](https://kubernetes.io/docs/tasks/administer-cluster/topology-manager/) was
introduced in kubernetes as part of kubelet in order to extract the best performance out of
these high performance hybrid systems. It performs optimizations related to resource allocation
in order to make it more likely for a given pod to perform optimally. In scenarios where
Topology Manager is unable to align topology of requested resources based on the selected
Topology Manager policy, the pod is rejected with Topology Affinity Error.
[This](https://github.com/kubernetes/kubernetes/issues/84869) kubernetes issue provides
further context on how runaway pods are created because the scheduler is topology-unaware.

In order to address this issue, scheduler needs to choose a node considering resource availability along with underlying resource topology and Topology Manager policy on the worker node.

## Motivation

In order to enable topology aware scheduling, resource topology information of the nodes in the cluster
needs to be exposed. This KEP describes how it would be implemented.

### Goals

Provisioning resources with topology information.

### Non-Goals

 - modification of any public API
 - improving and as a result modification of the TopologyManager and its policies

## Proposal

Add ability to expose resource information of the pod with NUMA topology into Node Feature
Discovery [daemon](https://github.com/kubernetes-sigs/node-feature-discovery).

## Design Details

The design consists of part which describes how datum collected and how it was provided.

Resources used by the pod could be obtained by [podresources](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/compute-device-assignment.md) interface of the kubelet.

To calculate available resources need to know all resources
which could be used by kubernetes. It could be calculated by
subtracting resources of kube cgroup and system cgroup from
system allocatable resources.


### Design based on podresources interface of the kubelet

Podresources interface of the kubelet is described in

[pkg/kubelet/apis/podresources/v1alpha1/api.proto](https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/apis/podresources/v1alpha1/api.proto)

it is available for every process on the worker node by
unix domain socket situated by the following path:

```go
filepath.Join(kl.getRootDir(), config.DefaultKubeletPodResourcesDirName)
```

it could be used to collect used resources on the worker node and to evaluate
its NUMA assignment (by device id).

Podresources could also be used to obtain initial information on resources of the worker node.

The PodResource API as it stands today:
* only provides information from Device Manager but not from CPU Manager.
* doesn't contain topology information as part of ContainerDevice.
* doesn't have the capability to let clients enumerate the resources.

This [KEP](https://github.com/kubernetes/enhancements/pull/1884) proposes extension of podresource api to address the above mentioned gaps.

With the changes proposed in the above KEP, this interface might look like as following:

```proto
syntax = "proto3";

package v1alpha1;


service PodResources {
    rpc List(ListPodResourcesRequest) returns (ListPodResourcesResponse) {}
    rpc GetAvailableResources(AvailableResourcesRequest) returns (AvailableResourcesResponse) {}
}

message ListPodResourcesRequest {}

message ListPodResourcesResponse {
    repeated PodResources pod_resources = 1;
}

message AvailableResourcesRequest {}

message AvailableResourcesResponse {
    repeated ContainerDevices devices = 1;
    repeated int64 cpu_ids = 2;
}

message ContainerDevices {
    string resource_name = 1;
    repeated string device_ids = 2;
    Topology topology = 3;
}
```

### API

Available resources with topology of the node should be stored in CRD. Format of the topology described
[in this document](https://docs.google.com/document/d/12kj3fK8boNuPNqob6F_pPU9ZTaNEnPGaXEooW1Cilwg/edit).


```go
// NodeResourceTopology is a specification for a Foo resource
type NodeResourceTopology struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	TopologyPolicies []string `json:"topologyPolicies"`
	Zones            ZoneMap  `json:"zones"`
}

// Zone is the spec for a NodeResourceTopology resource
type Zone struct {
	Type       string           `json:"type"`
	Parent     string           `json:"parent,omitempty"`
	Costs      map[string]int   `json:"costs,omitempty"`
	Attributes map[string]int   `json:"attributes,omitempty"`
	Resources  ResourceInfoMap  `json:"resources,omitempty"`
}

type ResourceInfo struct {
	Allocatable string `json:"allocatable"`
	Capacity    string `json:"capacity"`
}

type ZoneMap map[string]Zone
type ResourceInfoMap map[string]ResourceInfo
```

The code for working with it is generated by https://github.com/kubernetes/code-generator.git
One CRD instance contains information of available resources of the appropriate worker node.


### Integration into Node Feature Discovery

In order to allow the NFD-master Daemon to create, get, update, delete NodeResourceTopology CRD instances, ClusterRole and ClusterRoleBinding would have to be configured as below:

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

`serviceAccountName: noderesourcetopology-account` would have to be added to the manifest file of the Daemon.

### Graduation Criteria

* The feature has been stable and reliable in the past several releases.
* Documentation should exist for the feature.
* Test coverage of the feature is acceptable.


## Implementation History

- 2020-06-22: Initial KEP published.
- 2020-09-16: Updated to capture flexible/generic CRD specification. Moved design based on CRI as to the alternatives section because of its drawbacks.

## Alternatives

The provisioning of the resourcees could be implemented also by another way.
Daemon can keep resources in node annotation or in the pod's annotation.
Also kubelet can provide additional resources with NUMA information in ResourceName.

### Annotation approach

Annotation of the node or pod it's yet another place for arbitrary information.

This approach doesn't have known side effects.


### NUMA specification in ResourceName

The representation of resource consists of two parts subdomain/resourceName. Where
subdomain could be omitted. Subdomain contains vendor name. It doesn't suit well for
reflecting NUMA information of the node as well as / delimeter since subdomain is optional.
So new delimiter should be introduced to separate it from subdomain/resourceName.

It might look like:
numa%d///subdomain/resourceName

%d - number of NUMA node
/// - delimeter
numa%d/// - could be omitted

This approach may have side effects.

### Design based on CRI

The containerStatusResponse returned as a response to the ContainerStatus rpc contains `Info` field which is used by the container runtime for capturing ContainerInfo.
```go
message ContainerStatusResponse {
      ContainerStatus status = 1;
      map<string, string> info = 2;
}
```

Containerd has been used as the container runtime in the initial investigation. The internal container object info
[here](https://github.com/containerd/cri/blob/master/pkg/server/container_status.go#L130)

The Daemon set is responsible for the following:

- Parsing the info field to obtain container resource information
- Identifying NUMA nodes of the allocated resources
- Identifying total number of resources allocated on a NUMA node basis
- Detecting Node resource capacity on a NUMA node basis
- Updating the CRD instance per node indicating available resources on NUMA nodes, which is referred to the scheduler


#### Drawbacks

The content of the `info` field is free form, unregulated by the API contract. So, CRI-compliant container runtime engines are not required to add any configuration-specific information, like for example cpu allocation, here. In case of containerd container runtime, the Linux Container Configuration is added in the `info` map depending on the verbosity setting of the container runtime engine.

There is currently work going on in the community as part of the the Vertical Pod Autoscaling feature to update the ContainerStatus field to report back containerResources
[KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/20191025-kubelet-container-resources-cri-api-changes.md).
