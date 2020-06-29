# Exposure Node Resources With NUMA Topology Information

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Design based on CRI](#design-based-on-cri)
    - [Drawbacks](#drawbacks)
  - [Design based on podresources interface of the kubelet](#design-based-on-podresources-interface-of-the-kubelet)
  - [API](#api)
  - [Integration into Node Feature Discovery](#integration-into-node-feature-discovery)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Annotation approach](#annotation-approach)
  - [NUMA specification in ResourceName](#numa-specification-in-resourcename)
<!-- /toc -->

## Summary

Cluster which contains several nodes with NUMA topology and
enabled TopologyManager feature gate is not rare now. In such cluster
could be a situation when TopologyManager's admit handler on the kubelet
side refuses pod launching since pod's resources request doesn't sutisfy
selected TopologyManager policy, in this case pod should be rescheduled.
Also it can get stuck in the rescheduling cycle.
To avoid such kind of problem scheduler should choose a node considering topology
of the node and TopologyManager policy on the worker node.

## Motivation

For the scheduling which is topology aware, resources with topology
information should be provisioned.
This KEP describes how it would be implemented.

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
These resources with additional NUMA topolog could be provided by podresource interface.
This interface might look like as following:

```proto
syntax = "proto3";

package v1alpha1;


service PodResourcesLister {
    rpc GetAvailableResources(Empty) returns (AvailableResourcesResponse) {}
}

message AvailableResourcesResponse {
    repeated ContainerDevice contdevices = 1;
    repeated int64 cpu_ids = 2;
}
```

Currently podresources interface doesn't provide information from CPUManager only from
DeviceManager as well as information about memory which was used,
and ContainerDevice doesn't contain numa_id.
But information provided by podresource could be easily extended.
This [KEP](https://github.com/kubernetes/enhancements/pull/1884) propose such functionality.
Approach based on podresources doesn't have sufficient drawbacks as approach based on CRI, it
looks like preferred way to collect resources consumed by the pod.


### API

Available resources with topology of the node should be stored in CRD. Format of the topology described
[in this document](https://docs.google.com/document/d/12kj3fK8boNuPNqob6F_pPU9ZTaNEnPGaXEooW1Cilwg/edit).


```go
// NodeResourceTopology is a specification for a Foo resource
type NodeResourceTopology struct {
    metav1.TypeMeta           `json:",inline"`
    metav1.ObjectMeta         `json:"metadata,omitempty"`

    TopologyPolicies []string `json:"topologyPolicies"`
    Zones map[string]Zone     `json:"zones"`
}

// Zone is the spec for a NodeResourceTopology resource
type Zone struct {
    Type string
    Parent string
    Consts map[int]int
    Attributes map[string]int
    Resources ResourceInfoMap
}

type ResourceInfo struct {
    Allocatable int
    Capacity int
}

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
