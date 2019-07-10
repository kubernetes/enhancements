---
title: Kubelet Node Endpoint
authors:
 - "clamoriniere"
owning-sig: sig-node
participating-sigs: []
reviewers: ["dashpole"]
approvers: []
creation-date: 2019-05-02
status: implementable
---

# Kubelet Node Endpoint

## Motivation

Monitoring a kubernetes cluster can be a challenging job, especially on large clusters. When you want to monitor something, the first requirement is to not impact the performance of the application that you want to monitor.

Most of the monitoring kubernetes solutions are based on daemonset `agent` deployments. This pattern allows to access information from the host and the kubelet responsible for starting containers on each node. Most of the host/container monitoring information is available without requiring to do a call to a service running outside the node.
But some node information is currently only available through the `api-server`.
Having each agent calling the `api-server` to retrieve information may result in an api-server DDOS, which can put in danger the stability of your infrastructure.
This is the main issue that we want to solve with this KEP.


## Proposal

The Kubelet already proposes a set of endpoints (`/pods`, and soon `/metrics/resource/v1*`) that offer useful information for "local" node monitoring. In addition to those endpoints, we would like to introduce a new `/node` endpoint for exposing node resource information. It will allow daemonset-based monitoring solutions to retrieve node information locally (on the same host) like "labels, annotations" instead of using a call to the API server, which will solve some scalability issues in big clusters.

![kubelet-node-endpoint](https://i.ibb.co/khVg8rd/Untitled-Diagram.png)

Kubelet already caches and syncs its corresponding `Node` information from the API-server. The proposed solution opens a new endpoint "/node" that returns the `k8s.io/api/pkg/core/v1.Node` JSON representation of the Kubelet's Node. Only the Kubelet's Node resource is returned.

For consistency the endpoint path "/node" looks similar to the "/pods" endpoint (no API versioning).

## Other Solution

A remediation solution has been already investigated and implemented but requires the deployment of an extra component.
This component is working as a proxy/cache for queries to the `api-server` done by the daemonset pods.
This solution resolves the initial problem of reducing pressure on the `api-server`, but in fact, it only moves the problem to a less sensitive component.

This is the solution currently used by the Datadog monitoring solution deployment in Kubernetes via the `cluster-agent`.

### Why /Node endpoint is a better solution

Compared to the previous solution we think that using a `local` communication with the `kubelet` is a better solution and solves that scalability issue that we see with other presented solutions.

With this solution, the load on the `api-server` from the daemonset pods does not depend on the number of nodes. And the load on each kubelet is very low.

We have also seen the same approach, using a daemonset deployment that retrieved kubelet information, in other monitoring solutions.
For example, GKE is deploying the `prometheus-to-sd` (stackdriver) daemonset pods that retrieve metrics on each kubelet separately.

## Use Cases

Node information, that is currently only available through the `api-server`, provides a lot of context for the metrics generated on application/process running in pods.

### Pod metrics and logs tagging with Node labels or annotations

Monitoring solutions based on the kubelet API ("/pods" and "/metrics") can decorate Pod metrics with additional context information; Information like the "Availability zone", "Provider", "Server type" etc are very useful for investigating issues.

The current workaround is to retrieve that information from the API-Server, which generates a non-trivial amount of requests to the API-Server on a large Kubernetes cluster.

### Container images monitoring on Node
Knowing which Images are available on a specific node, and how much disk usage this represents can be useful for Node monitoring.
However the current information present in the `Node.status.Images` can only be a subset of the images present on the node, and the images storage usage is not realistic since images can share layers, and as such use less storage than it is reported.

Given the scope of the issue we describe here, proposing a solution for offering metrics on a node's images resource consumption should be done in a separate KEP.

### Node Info and characteristics

#### Container runtime

Currently, knowing which container runtime is used is no easy task. You can retrieve it locally via the kubelet configuration file, but depending on the Kubernetes distribution, this file can be located in different places.

Knowing the container runtime can be used to configure specific monitoring checks that interact directly with the container runtime.

#### Node resources

Allocatable vs Capacity host resources monitoring.

### Node conditions

Node's health information is available in the “status.Conditions” list, like DiskPressure, MemoryPressure, PIDPressure… All those conditions are useful for triggering Alert monitoring.

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
The Node instance is already cached by the kubelet, so calling the `s.host.GetNode()` doesn’t trigger each time a call to the API-server.

## Alternatives considered

Keep using the current workaround: retrieve Node information from the API-Server from a third-party component that proxies / caches this information for consumption by the Daemonset pods.

