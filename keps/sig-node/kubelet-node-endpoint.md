---
title: Kubelet Node Endpoint
authors:
  - "clamoriniere"
owning-sig: sig-node
participating-sigs: []
reviewers: []
approvers: []
creation-date: 2019-05-02
status: implementable
---

# Kubelet Node Endpoint

## Motivation

The Kubelet already proposes a set of endpoints (`/pods`, and soon `/metrics/resource/v1*`) that offer useful information for "local" node monitoring. In addition to those endpoints, we would like to introduce a new `/node` endpoint for displaying the node resource information. It will allow daemonset-based monitoring solutions to retrieve node information locally (on the same host) like "labels, annotations" instead of using a call to the API server, which will solve some scalability issues in big clusters.

![kubelet-node-endpoint](https://i.ibb.co/khVg8rd/Untitled-Diagram.png)

## Proposal

Kubelet already caches and sync its corresponding `Node` information from the API-server. The proposed solution opens a new endpoint "/node" that returns the `k8s.io/api/pkg/core/v1.Node` JSON representation of the Kubelet's Node. Only the Kubelet's Node resource is return.

For consistency the endpoint path "/node" looks similar to the "/pods" endpoint (no API versioning).

## Use Cases

* **Pod metrics and logs tagging with Node labels or annotation**: Monitoring workload solutions based on the kubelet API ("/pods" and "/metrics") used to decorate Pod metrics with additional context information; Information like the "Availability zone", "Provider", "Server type" etc are very useful for issue investigation.
The current workaround is to retrieve that information from the API-Server, that generates a consequent number of requests to the API-Server on a large kubernetes cluster.
* **Container images monitoring on Node**: Knowing which Images and Disk usage that it represents on a specific node can be useful for Node monitoring.
* **Node conditions**: Node's health information is available in the “status.Conditions” list, like DiskPressure, MemoryPressure, PIDPressure… All those conditions are useful for triggering Alert monitoring and used “locally” for correlation with Pod misbehavior.
* **Node Info and characteristic**: 
   - Knowing the container runtime can be used to configure monitoring checks that interact directly with the container runtime.
   - Knowing the operating system could simplify the configuration of the host monitoring.
   - Allocatable vs Capacity host resources monitoring.

## Implementation

Implementation is quite simple, in `k8s.io/pkg/kubelet/server.Server` a new `restful.WebService` needs to be registered

```golang
// patterns with the restful Container.
func (s *Server) InstallDefaultHandlers() {
	// ...
	wsNode := new(restful.WebService)
	wsNode.
		Path("/node").
		Produces(restful.MIME_JSON)
	wsNode.Route(wsNode.GET("").
		To(s.getNode).
		Operation("getNode"))
	s.restfulCont.Add(wsNode)
     // ...
}
```
 
and the corresponding restful.Request handler: `getNode()`. Node resource instance is already available thanks to the host HostInterface (composition of `k8s.io/pkg/kubelet/server/stats.Provider` interface) present in the `k8s.io/pkg/kubelet/server.Server` instance.

```golang
// getNode returns the Node bound to the Kubelet and it spec/status.
func (s *Server) getNode(request *restful.Request, response *restful.Response) {
	node, err := s.host.GetNode()
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	data, err := encodeNode(node)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	writeJSONResponse(response, data)
}
 
// encodeNode creates an v1.Node object from node and returns the encoded
// Node.
func encodeNode(node *v1.Node) (data []byte, err error) {
	codec := legacyscheme.Codecs.LegacyCodec(schema.GroupVersion{Group: v1.GroupName, Version: "v1"})
	return runtime.Encode(codec, node)
}

```
 
The Node instance is already cached by the kubelet, and so calling the `s.host.GetNode()` didn't trigger each time a call to the API-server.

## Alternatives considered

Stay with the current workaround: retrieve Node information from the API-Server.
